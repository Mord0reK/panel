package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"agent/internal/collector"
	"agent/internal/docker"
	"agent/internal/output"
)

const updateCheckConcurrency = 4

func main() {
	if len(os.Args) < 2 {
		runStats()
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "stats":
		runStats()
	case "info":
		runInfo()
	case "stop":
		runContainerCmd(os.Args[2:], "stop")
	case "start":
		runContainerCmd(os.Args[2:], "start")
	case "restart":
		runContainerCmd(os.Args[2:], "restart")
	case "check-updates":
		runCheckUpdates(os.Args[2:])
	case "update":
		runUpdate(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		fmt.Println("Available commands:")
		fmt.Println("  stats              - Collect and write stats to JSON (default)")
		fmt.Println("  info               - Show system information (hostname, CPU, RAM, etc.)")
		fmt.Println("  stop <container>   - Stop a container")
		fmt.Println("  start <container>  - Start a container")
		fmt.Println("  restart <container> - Restart a container")
		fmt.Println("  check-updates [container|project] - Check for updates")
		fmt.Println("  update <container|project> - Update container or compose project")
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

	filename := "stats.json"
	if err := output.WriteToJSON(filename, sysMetrics, dockerMetrics); err != nil {
		log.Fatalf("Error writing metrics: %v", err)
	}

	log.Printf("Metrics written to %s", filename)
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
