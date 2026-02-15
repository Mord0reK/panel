package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"agent/internal/collector"
	"agent/internal/docker"
	"agent/internal/output"
	"agent/internal/websocket"

	"github.com/moby/moby/client"
)

const updateCheckConcurrency = 4

var backendURL string

func init() {
	flag.StringVar(&backendURL, "backend-url", "", "WebSocket backend URL (or use BACKEND_URL env var)")
}

func main() {
	flag.Parse()

	if envURL := os.Getenv("BACKEND_URL"); envURL != "" && backendURL == "" {
		backendURL = envURL
	}

	args := flag.Args()
	if len(args) < 1 {
		runStats()
		return
	}

	cmd := args[0]

	switch cmd {
	case "stats":
		runStats()
	case "info":
		runInfo()
	case "ws":
		runWebSocket()
	case "stop":
		runContainerCmd(flag.Args()[1:], "stop")
	case "start":
		runContainerCmd(flag.Args()[1:], "start")
	case "restart":
		runContainerCmd(flag.Args()[1:], "restart")
	case "check-updates":
		runCheckUpdates(flag.Args()[1:])
	case "update":
		runUpdate(flag.Args()[1:])
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		fmt.Println("Available commands:")
		fmt.Println("  stats              - Collect and write stats to JSON (default)")
		fmt.Println("  info               - Show system information (hostname, CPU, RAM, etc.)")
		fmt.Println("  ws                 - Run agent in WebSocket mode (requires BACKEND_URL)")
		fmt.Println("  stop <container>   - Stop a container")
		fmt.Println("  start <container>  - Start a container")
		fmt.Println("  restart <container> - Restart a container")
		fmt.Println("  check-updates [container|project] - Check for updates")
		fmt.Println("  update <container|project> - Update container or compose project")
		fmt.Println("")
		fmt.Println("WebSocket actions:")
		fmt.Println("  stats              - system + full docker metrics")
		fmt.Println("  info               - static system information")
		fmt.Println("  docker-details     - full docker details (labels, network, compose groups)")
		fmt.Println("")
		fmt.Println("Environment variables:")
		fmt.Println("  BACKEND_URL        - WebSocket server URL for 'ws' command")
	}
}

func runStats() {
	ctx := context.Background()

	log.Println("Starting agent...")

	sysMetrics, err := collector.CollectSystemMetrics(ctx)
	if err != nil {
		log.Printf("Error collecting system metrics: %v", err)
		sysMetrics = nil
	}

	var dockerMetrics interface{}
	dockerCli, err := docker.NewDockerClient()
	if err != nil {
		log.Printf("Docker client error (is Docker running?): %v", err)
	} else {
		defer dockerCli.Close()
		dm, err := docker.CollectContainerMetrics(ctx, dockerCli)
		if err != nil {
			log.Printf("Error collecting docker metrics: %v", err)
		} else {
			dockerMetrics = dm
		}
	}

	filename := "data/stats.json"
	if err := output.WriteToJSON(filename, sysMetrics, dockerMetrics); err != nil {
		log.Fatalf("Error writing metrics: %v", err)
	}

	log.Printf("Metrics written to %s", filename)
}

func runWebSocket() {
	if backendURL == "" {
		fmt.Println("Error: BACKEND_URL not set")
		fmt.Println("Usage: agent ws")
		fmt.Println("  --backend-url string   WebSocket backend URL")
		fmt.Println("  Or set BACKEND_URL environment variable")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wsClient := websocket.NewClient(backendURL)

	if err := wsClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	log.Printf("WebSocket connected to %s", backendURL)

	dockerCli, err := docker.NewDockerClient()
	if err != nil {
		log.Printf("Warning: failed to create Docker client: %v", err)
	}
	defer func() {
		if dockerCli != nil {
			dockerCli.Close()
		}
	}()

	go sendMetricsLoop(ctx, wsClient, dockerCli)

	for {
		err := wsClient.Listen(ctx, func(cmd websocket.Command) error {
			return handleWebSocketCommand(ctx, wsClient, dockerCli, cmd)
		})

		log.Printf("WebSocket connection lost: %v", err)
		log.Printf("Reconnecting in 10 seconds...")

		select {
		case <-ctx.Done():
			wsClient.Close()
			return
		case <-time.After(10 * time.Second):
			if err := wsClient.Reconnect(ctx); err != nil {
				log.Printf("Reconnect failed: %v", err)
			}
		}
	}
}

func sendMetricsLoop(ctx context.Context, wsClient *websocket.Client, dockerCli *client.Client) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !wsClient.IsConnected() {
				continue
			}

			sysMetrics, err := collector.CollectSystemMetrics(ctx)
			if err != nil {
				log.Printf("Error collecting system metrics: %v", err)
				continue
			}

			var dockerMetrics interface{}
			if dockerCli != nil {
				dm, _ := docker.CollectRealtimeContainerMetrics(ctx, dockerCli)
				dockerMetrics = dm
			}

			msg := websocket.MetricsMessage{
				Type:      "metrics",
				Timestamp: sysMetrics.Timestamp,
				System:    sysMetrics,
				Docker:    dockerMetrics,
			}

			if err := wsClient.SendMessage("metrics", msg); err != nil {
				log.Printf("Error sending metrics: %v", err)
			}
		}
	}
}

func handleWebSocketCommand(ctx context.Context, wsClient *websocket.Client, dockerCli *client.Client, cmd websocket.Command) error {
	log.Printf("Received command: type=%s action=%s target=%s", cmd.Type, cmd.Action, cmd.Target)

	var result map[string]interface{}

	switch cmd.Action {
	case "stats":
		sysMetrics, err := collector.CollectSystemMetrics(ctx)
		if err != nil {
			result = map[string]interface{}{"error": err.Error()}
		} else {
			var dockerMetrics interface{}
			if dockerCli != nil {
				dm, _ := docker.CollectContainerMetrics(ctx, dockerCli)
				dockerMetrics = dm
			}
			result = map[string]interface{}{"system": sysMetrics, "docker": dockerMetrics}
		}

	case "info":
		info, err := collector.CollectSystemInfo(ctx)
		if err != nil {
			result = map[string]interface{}{"error": err.Error()}
		} else {
			result = map[string]interface{}{"info": info}
		}

	case "docker-details":
		if dockerCli == nil {
			result = map[string]interface{}{"error": "docker client unavailable"}
		} else {
			details, err := docker.CollectContainerMetrics(ctx, dockerCli)
			if err != nil {
				result = map[string]interface{}{"error": err.Error()}
			} else {
				result = map[string]interface{}{"docker": details}
			}
		}

	case "stop":
		if cmd.Target == "" {
			result = map[string]interface{}{"error": "target is required"}
		} else {
			err := runStopContainer(ctx, dockerCli, cmd.Target)
			result = map[string]interface{}{"success": err == nil, "error": func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}()}
		}

	case "start":
		if cmd.Target == "" {
			result = map[string]interface{}{"error": "target is required"}
		} else {
			err := runStartContainer(ctx, dockerCli, cmd.Target)
			result = map[string]interface{}{"success": err == nil, "error": func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}()}
		}

	case "restart":
		if cmd.Target == "" {
			result = map[string]interface{}{"error": "target is required"}
		} else {
			err := runRestartContainer(ctx, dockerCli, cmd.Target)
			result = map[string]interface{}{"success": err == nil, "error": func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}()}
		}

	case "check-updates":
		updates, err := runCheckUpdatesForTarget(ctx, dockerCli, cmd.Target)
		result = map[string]interface{}{"updates": updates, "error": func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}()}

	case "update":
		if cmd.Target == "" {
			result = map[string]interface{}{"error": "target is required"}
		} else {
			results, err := runUpdateTarget(ctx, dockerCli, cmd.Target)
			result = map[string]interface{}{"results": results, "error": func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}()}
		}

	default:
		result = map[string]interface{}{"error": "unknown action: " + cmd.Action}
	}

	return wsClient.SendMessage("result", result)
}

func runStopContainer(ctx context.Context, dockerCli *client.Client, containerName string) error {
	containerID, err := docker.FindContainerByName(ctx, dockerCli, containerName)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	manager := docker.NewContainerManager(dockerCli)
	return manager.StopContainer(ctx, containerID)
}

func runStartContainer(ctx context.Context, dockerCli *client.Client, containerName string) error {
	containerID, err := docker.FindContainerByName(ctx, dockerCli, containerName)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	manager := docker.NewContainerManager(dockerCli)
	return manager.StartContainer(ctx, containerID)
}

func runRestartContainer(ctx context.Context, dockerCli *client.Client, containerName string) error {
	containerID, err := docker.FindContainerByName(ctx, dockerCli, containerName)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	manager := docker.NewContainerManager(dockerCli)
	return manager.RestartContainer(ctx, containerID)
}

func runCheckUpdatesForTarget(ctx context.Context, dockerCli *client.Client, target string) ([]interface{}, error) {
	manager := docker.NewContainerManager(dockerCli)

	if target != "" {
		containerID, err := docker.FindContainerByName(ctx, dockerCli, target)
		if err == nil && containerID != "" {
			updates, err := manager.CheckForUpdates(ctx, containerID)
			if err != nil {
				return nil, err
			}
			return convertUpdates(updates), nil
		}

		workingDir := ""
		metrics, _ := docker.CollectContainerMetrics(ctx, dockerCli)
		for _, g := range metrics.ComposeGroups {
			if g.Project == target || g.Name == target {
				workingDir = g.WorkingDir
				break
			}
		}

		updates, err := manager.CheckComposeUpdates(ctx, target, workingDir)
		if err != nil {
			return nil, err
		}
		return convertUpdates(updates), nil
	}

	return nil, nil
}

func runUpdateTarget(ctx context.Context, dockerCli *client.Client, target string) ([]interface{}, error) {
	manager := docker.NewContainerManager(dockerCli)

	metrics, _ := docker.CollectContainerMetrics(ctx, dockerCli)
	for _, g := range metrics.ComposeGroups {
		if g.Project == target || g.Name == target {
			results, err := manager.UpdateComposeGroup(ctx, target, g.WorkingDir)
			if err != nil {
				return nil, err
			}
			return convertResults(results), nil
		}
	}

	containerID, err := docker.FindContainerByName(ctx, dockerCli, target)
	if err == nil && containerID != "" {
		results, err := manager.UpdateContainer(ctx, containerID)
		if err != nil {
			return nil, err
		}
		return convertResults(results), nil
	}

	return nil, fmt.Errorf("not found: %s", target)
}

func convertUpdates(updates []docker.UpdateInfo) []interface{} {
	result := make([]interface{}, len(updates))
	for i, u := range updates {
		result[i] = map[string]interface{}{
			"container_name":   u.ContainerName,
			"current_image":    u.CurrentImage,
			"current_version":  u.CurrentVersion,
			"latest_image":     u.LatestImage,
			"latest_version":   u.LatestVersion,
			"update_available": u.UpdateAvailable,
			"status":           u.Status,
			"error":            u.Error,
			"project":          u.Project,
			"service":          u.Service,
		}
	}
	return result
}

func convertResults(results []docker.UpdateResult) []interface{} {
	result := make([]interface{}, len(results))
	for i, r := range results {
		result[i] = map[string]interface{}{
			"container": r.Container,
			"success":   r.Success,
			"message":   r.Message,
		}
	}
	return result
}

func runInfo() {
	ctx := context.Background()

	info, err := collector.CollectSystemInfo(ctx)
	if err != nil {
		log.Fatalf("Failed to collect system info: %v", err)
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal info: %v", err)
	}

	fmt.Println(string(data))
}

func runContainerCmd(args []string, action string) {
	if len(args) < 1 {
		fmt.Printf("Usage: agent %s <container>\n", action)
		os.Exit(1)
	}

	containerName := args[0]
	ctx := context.Background()

	dockerCli, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerCli.Close()

	containerID, err := docker.FindContainerByName(ctx, dockerCli, containerName)
	if err != nil {
		log.Fatalf("Container not found: %v", err)
	}

	manager := docker.NewContainerManager(dockerCli)

	switch action {
	case "stop":
		if err := manager.StopContainer(ctx, containerID); err != nil {
			log.Fatalf("Failed to stop container: %v", err)
		}
		fmt.Printf("Container %s stopped successfully\n", containerName)
	case "start":
		if err := manager.StartContainer(ctx, containerID); err != nil {
			log.Fatalf("Failed to start container: %v", err)
		}
		fmt.Printf("Container %s started successfully\n", containerName)
	case "restart":
		if err := manager.RestartContainer(ctx, containerID); err != nil {
			log.Fatalf("Failed to restart container: %v", err)
		}
		fmt.Printf("Container %s restarted successfully\n", containerName)
	}
}

func runCheckUpdates(args []string) {
	ctx := context.Background()

	dockerCli, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerCli.Close()

	manager := docker.NewContainerManager(dockerCli)

	if len(args) < 1 {
		metrics, err := docker.CollectContainerMetrics(ctx, dockerCli)
		if err != nil {
			log.Fatalf("Failed to collect metrics: %v", err)
		}

		fmt.Println("Compose Groups:")

		type groupResult struct {
			name    string
			updates []docker.UpdateInfo
			err     error
		}

		var wg sync.WaitGroup
		results := make(chan groupResult, len(metrics.ComposeGroups))
		groupLimiter := make(chan struct{}, updateCheckConcurrency)

		for _, group := range metrics.ComposeGroups {
			wg.Add(1)
			go func(g docker.ComposeGroup) {
				defer wg.Done()
				groupLimiter <- struct{}{}
				defer func() { <-groupLimiter }()
				wd := getWorkingDirFromGroup(g)
				updates, err := manager.CheckComposeUpdates(ctx, g.Project, wd)
				results <- groupResult{name: g.Name, updates: updates, err: err}
			}(group)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		type groupState struct {
			status string
			err    string
		}
		groupStatus := make(map[string]groupState)
		for r := range results {
			if r.err != nil {
				fmt.Printf("  %s: error checking updates: %v\n", r.name, r.err)
				continue
			}
			for _, u := range r.updates {
				groupStatus[r.name] = groupState{status: u.Status, err: u.Error}
			}
		}

		for _, group := range metrics.ComposeGroups {
			if state, ok := groupStatus[group.Name]; ok {
				switch state.status {
				case docker.UpdateStatusAvailable:
					fmt.Printf("  %s: update available\n", group.Name)
				case docker.UpdateStatusRateLimit:
					if strings.TrimSpace(state.err) != "" {
						fmt.Printf("  %s: rate limited (%s)\n", group.Name, state.err)
					} else {
						fmt.Printf("  %s: rate limited\n", group.Name)
					}
				case docker.UpdateStatusUnknown:
					if strings.TrimSpace(state.err) != "" {
						fmt.Printf("  %s: unknown (%s)\n", group.Name, state.err)
					} else {
						fmt.Printf("  %s: unknown\n", group.Name)
					}
				case docker.UpdateStatusLocal:
					fmt.Printf("  %s: local\n", group.Name)
				default:
					fmt.Printf("  %s: up to date\n", group.Name)
				}
			}
		}

		fmt.Println("\nStandalone Containers:")

		var standaloneWG sync.WaitGroup
		standaloneResults := make(chan docker.UpdateInfo, len(metrics.StandaloneContainers))
		standaloneLimiter := make(chan struct{}, updateCheckConcurrency)

		for _, c := range metrics.StandaloneContainers {
			standaloneWG.Add(1)
			go func(containerID, containerName string) {
				defer standaloneWG.Done()
				standaloneLimiter <- struct{}{}
				defer func() { <-standaloneLimiter }()
				updates, err := manager.CheckForUpdates(ctx, containerID)
				if err != nil {
					fmt.Printf("  %s: error checking updates: %v\n", containerName, err)
					return
				}
				for _, u := range updates {
					standaloneResults <- u
				}
			}(c.ID, c.Name)
		}

		go func() {
			standaloneWG.Wait()
			close(standaloneResults)
		}()

		for u := range standaloneResults {
			switch u.Status {
			case docker.UpdateStatusAvailable:
				fmt.Printf("  %s: image=%s status=update_available\n", u.ContainerName, u.CurrentImage)
			case docker.UpdateStatusRateLimit:
				if strings.TrimSpace(u.Error) != "" {
					fmt.Printf("  %s: image=%s status=rate_limited error=%s\n", u.ContainerName, u.CurrentImage, u.Error)
				} else {
					fmt.Printf("  %s: image=%s status=rate_limited\n", u.ContainerName, u.CurrentImage)
				}
			case docker.UpdateStatusUnknown:
				if strings.TrimSpace(u.Error) != "" {
					fmt.Printf("  %s: image=%s status=unknown error=%s\n", u.ContainerName, u.CurrentImage, u.Error)
				} else {
					fmt.Printf("  %s: image=%s status=unknown\n", u.ContainerName, u.CurrentImage)
				}
			default:
				fmt.Printf("  %s: image=%s status=up_to_date\n", u.ContainerName, u.CurrentImage)
			}
		}
		return
	}

	target := args[0]

	containerID, err := docker.FindContainerByName(ctx, dockerCli, target)
	if err == nil && containerID != "" {
		updates, err := manager.CheckForUpdates(ctx, containerID)
		if err != nil {
			log.Fatalf("Failed to check updates: %v", err)
		}
		printUpdates(updates)
		return
	}

	project := target
	workingDir := ""
	metrics, err := docker.CollectContainerMetrics(ctx, dockerCli)
	if err == nil {
		for _, g := range metrics.ComposeGroups {
			if g.Project == project || g.Name == project {
				for _, c := range g.Containers {
					wd, _ := c.Labels["com.docker.compose.project.working_dir"]
					workingDir = wd
					break
				}
				break
			}
		}
	}

	updates, err := manager.CheckComposeUpdates(ctx, project, workingDir)
	if err != nil {
		log.Fatalf("Failed to check updates: %v", err)
	}
	printUpdates(updates)
}

func runUpdate(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: agent update <container|project>")
		os.Exit(1)
	}

	target := args[0]
	ctx := context.Background()

	dockerCli, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerCli.Close()

	manager := docker.NewContainerManager(dockerCli)

	// ═══════════════════════════════════════════════════════════
	// PRIORYTET 1: Szukaj compose project (EXACT MATCH)
	// ═══════════════════════════════════════════════════════════
	metrics, err := docker.CollectContainerMetrics(ctx, dockerCli)
	if err == nil {
		for _, g := range metrics.ComposeGroups {
			// EXACT match na project name
			if g.Project == target || g.Name == target {
				fmt.Printf("Updating compose project: %s\n", target)
				results, err := manager.UpdateComposeGroup(ctx, target, g.WorkingDir)
				if err != nil {
					log.Fatalf("Failed to update compose group: %v", err)
				}
				printResults(results)
				return
			}
		}
	}

	// ═══════════════════════════════════════════════════════════
	// PRIORYTET 2: Jak nie ma project, szukaj kontenera (EXACT MATCH)
	// ═══════════════════════════════════════════════════════════
	containerID, err := docker.FindContainerByName(ctx, dockerCli, target)
	if err == nil && containerID != "" {
		results, err := manager.UpdateContainer(ctx, containerID)
		if err != nil {
			log.Fatalf("Failed to update container: %v", err)
		}
		printResults(results)
		return
	}

	// Nic nie znaleziono
	log.Fatalf("Not found: %s (no compose project or container with this name)", target)
}

func printUpdates(updates []docker.UpdateInfo) {
	data, err := json.MarshalIndent(updates, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal updates: %v", err)
	}
	fmt.Println(string(data))
}

func printResults(results []docker.UpdateResult) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal results: %v", err)
	}
	fmt.Println(string(data))
}

func getWorkingDirFromGroup(group docker.ComposeGroup) string {
	return group.WorkingDir
}
