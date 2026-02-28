package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type ContainerInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Image       string                 `json:"image"`
	Status      string                 `json:"status"`
	State       string                 `json:"state"`
	Created     int64                  `json:"created"`
	Stats       *ContainerStats        `json:"stats,omitempty"`
	NetworkInfo map[string]NetworkInfo `json:"network_info,omitempty"`
	Project     string                 `json:"project,omitempty"`
	Service     string                 `json:"service,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty"`
	workingDir  string
}

type ContainerMetrics struct {
	Timestamp            time.Time       `json:"timestamp"`
	Containers           []ContainerInfo `json:"containers"`
	ComposeGroups        []ComposeGroup  `json:"compose_groups,omitempty"`
	StandaloneContainers []ContainerInfo `json:"standalone_containers,omitempty"`
}

type RealtimeContainerInfo struct {
	ContainerID string  `json:"container_id"`
	Name        string  `json:"name"`
	Image       string  `json:"image"`
	Project     string  `json:"project"`
	Service     string  `json:"service"`
	State       string  `json:"state"`
	Health      string  `json:"health"`
	Status      string  `json:"status"`
	Timestamp   int64   `json:"timestamp"`
	CPU         float64 `json:"cpu_percent"`
	MemUsed     uint64  `json:"mem_used"`
	MemPercent  float64 `json:"mem_percent"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskPercent float64 `json:"disk_percent"`
	NetRx       uint64  `json:"net_rx_bytes"`
	NetTx       uint64  `json:"net_tx_bytes"`

	// Raw cumulative CPU usage in microseconds (from cgroups).
	// Used by SnapshotCollector to compute CPU% delta between ticks.
	// Not sent over the wire as a meaningful value — set to 0 after snapshot normalisation.
	CPURawUsec uint64 `json:"cpu_raw_usec,omitempty"`
}

type RealtimeContainerMetrics struct {
	Timestamp  time.Time               `json:"timestamp"`
	Containers []RealtimeContainerInfo `json:"containers"`
}

type ComposeGroup struct {
	Name       string          `json:"name"`
	Project    string          `json:"project"`
	WorkingDir string          `json:"working_dir,omitempty"`
	Containers []ContainerInfo `json:"containers"`
}

type ContainerStats struct {
	CPU     CPUStats     `json:"cpu"`
	Memory  MemoryStats  `json:"memory"`
	BlockIO BlockIOStats `json:"block_io"`
	Network NetworkStats `json:"network"`
	PIDs    int64        `json:"pids"`
}

type CPUStats struct {
	Percent      float64 `json:"percent"`
	CPUContainer float64 `json:"cpu_container"`
	CPUSystem    float64 `json:"cpu_system"`
	CPUUser      float64 `json:"cpu_user"`
	OnlineCPUs   int64   `json:"online_cpus"`
}

type MemoryStats struct {
	Usage   uint64  `json:"usage"`
	Limit   uint64  `json:"limit"`
	Percent float64 `json:"percent"`
	Cache   uint64  `json:"cache"`
	RSS     uint64  `json:"rss"`
	Swap    uint64  `json:"swap"`
}

type BlockIOStats struct {
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
}

type NetworkStats struct {
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	TxPackets uint64 `json:"tx_packets"`
	RxErrors  uint64 `json:"rx_errors"`
	TxErrors  uint64 `json:"tx_errors"`
}

type NetworkInfo struct {
	NetworkMode string `json:"network_mode"`
	IPAddress   string `json:"ip_address"`
	Gateway     string `json:"gateway"`
	MacAddress  string `json:"mac_address"`
}

// parseHealthFromStatus extracts health status from Docker's Status string.
func parseHealthFromStatus(status string) string {
	start := strings.LastIndex(status, "(")
	end := strings.LastIndex(status, ")")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	inner := strings.TrimSpace(status[start+1 : end])
	if strings.HasPrefix(inner, "health: ") {
		inner = strings.TrimPrefix(inner, "health: ")
	}
	switch inner {
	case "healthy", "unhealthy", "starting":
		return inner
	}
	return ""
}

func NewDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return cli, nil
}

// cgroupBasePath returns the cgroup v2 path for a given full container ID.
// Tries the systemd slice path first (most common on Ubuntu/Debian with Docker),
// then the legacy docker-managed path as fallback.
func cgroupBasePath(containerID string) string {
	systemdPath := fmt.Sprintf("/sys/fs/cgroup/system.slice/docker-%s.scope", containerID)
	if _, err := os.Stat(systemdPath); err == nil {
		return systemdPath
	}
	return fmt.Sprintf("/sys/fs/cgroup/docker/%s", containerID)
}

// readUint64File reads a single uint64 value from a cgroup file.
func readUint64File(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var v uint64
	_, err = fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &v)
	return v, err
}

// readCgroupStats populates RealtimeContainerInfo from cgroup v2 files.
// CPU is returned as a raw cumulative microsecond counter (CPURawUsec);
// the actual CPU% is computed by SnapshotCollector using delta between ticks.
func readCgroupStats(containerID string, info *RealtimeContainerInfo) {
	base := cgroupBasePath(containerID)

	// ── CPU ────────────────────────────────────────────────────────────
	// cpu.stat contains "usage_usec <N>" — cumulative nanoseconds of CPU time.
	if data, err := os.ReadFile(base + "/cpu.stat"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "usage_usec ") {
				var usec uint64
				fmt.Sscanf(strings.TrimPrefix(line, "usage_usec "), "%d", &usec)
				info.CPURawUsec = usec
				break
			}
		}
	}

	// ── Memory ─────────────────────────────────────────────────────────
	if memUsed, err := readUint64File(base + "/memory.current"); err == nil {
		// Subtract inactive_file (page cache) the same way docker stats does.
		if statData, err := os.ReadFile(base + "/memory.stat"); err == nil {
			for _, line := range strings.Split(string(statData), "\n") {
				if strings.HasPrefix(line, "inactive_file ") {
					var inactive uint64
					fmt.Sscanf(strings.TrimPrefix(line, "inactive_file "), "%d", &inactive)
					if memUsed > inactive {
						memUsed -= inactive
					}
					break
				}
			}
		}
		info.MemUsed = memUsed

		if limitData, err := os.ReadFile(base + "/memory.max"); err == nil {
			limitStr := strings.TrimSpace(string(limitData))
			if limitStr != "max" {
				var limit uint64
				if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && limit > 0 {
					info.MemPercent = float64(memUsed) / float64(limit) * 100.0
				}
			}
		}
	}

	// ── Block I/O ──────────────────────────────────────────────────────
	// io.stat format per line: "major:minor rbytes=N wbytes=N ..."
	if data, err := os.ReadFile(base + "/io.stat"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			for _, field := range strings.Fields(line) {
				switch {
				case strings.HasPrefix(field, "rbytes="):
					var v uint64
					fmt.Sscanf(strings.TrimPrefix(field, "rbytes="), "%d", &v)
					info.DiskUsed += v
				case strings.HasPrefix(field, "wbytes="):
					var v uint64
					fmt.Sscanf(strings.TrimPrefix(field, "wbytes="), "%d", &v)
					info.DiskUsed += v
				}
			}
		}
	}

	// ── Network ────────────────────────────────────────────────────────
	// cgroup v2 does not expose network counters — we read /proc/<pid>/net/dev
	// where <pid> is the first thread listed in cgroup.threads.
	if data, err := os.ReadFile(base + "/cgroup.threads"); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			readProcNetDev(strings.TrimSpace(lines[0]), info)
		}
	}
}

// readProcNetDev reads network counters from /proc/<pid>/net/dev for the
// network namespace of the given PID (i.e. the container's netns).
func readProcNetDev(pid string, info *RealtimeContainerInfo) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%s/net/dev", pid))
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	// First two lines are headers.
	for _, line := range lines[2:] {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		iface := strings.TrimSuffix(fields[0], ":")
		if iface == "lo" {
			continue
		}
		var rx, tx uint64
		fmt.Sscanf(fields[1], "%d", &rx)
		fmt.Sscanf(fields[9], "%d", &tx)
		info.NetRx += rx
		info.NetTx += tx
	}
}

// CollectRealtimeContainerMetrics collects per-container metrics using cgroup v2
// filesystem reads instead of the Docker Stats API.
// CPU is returned as CPURawUsec (cumulative); SnapshotCollector computes the %.
func CollectRealtimeContainerMetrics(ctx context.Context, cli *client.Client) (*RealtimeContainerMetrics, error) {
	metrics := &RealtimeContainerMetrics{
		Timestamp: time.Now(),
	}

	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var wg sync.WaitGroup
	results := make(chan RealtimeContainerInfo, len(containers.Items))

	for _, c := range containers.Items {
		wg.Add(1)
		go func(cID, cName, cImage, cState, cStatus string, labels map[string]string) {
			defer wg.Done()

			name := cName
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}

			info := RealtimeContainerInfo{
				ContainerID: cID[:12],
				Name:        name,
				State:       cState,
				Health:      parseHealthFromStatus(cStatus),
				Status:      cStatus,
				Image:       cImage,
				Timestamp:   time.Now().Unix(),
			}

			if labels != nil {
				if project, ok := labels["com.docker.compose.project"]; ok {
					info.Project = project
				}
				if service, ok := labels["com.docker.compose.service"]; ok {
					info.Service = service
				}
			}

			if cState == "running" {
				readCgroupStats(cID, &info)
			}

			results <- info
		}(c.ID, c.Names[0], c.Image, string(c.State), string(c.Status), c.Labels)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for info := range results {
		metrics.Containers = append(metrics.Containers, info)
	}

	return metrics, nil
}

// CollectContainerMetrics collects full container details (used for docker-details
// command and update checks). Still uses Docker API — this path is not hot.
func CollectContainerMetrics(ctx context.Context, cli *client.Client) (*ContainerMetrics, error) {
	metrics := &ContainerMetrics{
		Timestamp: time.Now(),
	}

	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers.Items {
		name := c.Names[0]
		if len(name) > 0 && name[0] == '/' {
			name = name[1:]
		}

		info := ContainerInfo{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			State:   string(c.State),
			Created: c.Created,
		}

		if c.Labels != nil {
			if project, ok := c.Labels["com.docker.compose.project"]; ok {
				info.Project = project
			}
			if service, ok := c.Labels["com.docker.compose.service"]; ok {
				info.Service = service
			}
			if wd, ok := c.Labels["com.docker.compose.project.working_dir"]; ok {
				info.workingDir = wd
			}
			info.Labels = filterLabels(c.Labels)
		}

		if string(c.State) == "running" {
			statsResp, err := cli.ContainerStats(ctx, c.ID, client.ContainerStatsOptions{Stream: false})
			if err == nil {
				var stats container.StatsResponse
				if err := json.NewDecoder(statsResp.Body).Decode(&stats); err == nil {
					info.Stats = parseContainerStats(&stats)
				}
				_ = statsResp.Body.Close()
			}
		}

		inspect, err := cli.ContainerInspect(ctx, c.ID, client.ContainerInspectOptions{})
		if err == nil && inspect.Container.NetworkSettings != nil {
			info.NetworkInfo = make(map[string]NetworkInfo)
			for netName, net := range inspect.Container.NetworkSettings.Networks {
				ipAddr := ""
				if net.IPAddress.IsValid() {
					ipAddr = net.IPAddress.String()
				}
				gateway := ""
				if net.Gateway.IsValid() {
					gateway = net.Gateway.String()
				}
				macAddr := ""
				if len(net.MacAddress) > 0 {
					macAddr = net.MacAddress.String()
				}
				info.NetworkInfo[netName] = NetworkInfo{
					NetworkMode: net.NetworkID,
					IPAddress:   ipAddr,
					Gateway:     gateway,
					MacAddress:  macAddr,
				}
			}
		}

		metrics.Containers = append(metrics.Containers, info)
	}

	grouped := GroupContainersByCompose(metrics.Containers)
	metrics.ComposeGroups = grouped.Groups
	metrics.StandaloneContainers = grouped.StandaloneContainers

	return metrics, nil
}

func parseContainerStats(stats *container.StatsResponse) *ContainerStats {
	result := &ContainerStats{}

	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0 && cpuDelta > 0 {
		result.CPU.Percent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	result.CPU.CPUContainer = float64(stats.CPUStats.CPUUsage.TotalUsage)
	result.CPU.CPUSystem = float64(stats.CPUStats.SystemUsage)
	result.CPU.OnlineCPUs = int64(stats.CPUStats.OnlineCPUs)

	result.Memory.Limit = stats.MemoryStats.Limit
	if stats.MemoryStats.Stats != nil {
		result.Memory.Cache = stats.MemoryStats.Stats["cache"]
		result.Memory.RSS = stats.MemoryStats.Stats["rss"]
		result.Memory.Swap = stats.MemoryStats.Stats["swap"]
	}
	memUsed := stats.MemoryStats.Usage
	if stats.MemoryStats.Stats != nil {
		if inactiveFile, ok := stats.MemoryStats.Stats["inactive_file"]; ok && memUsed > inactiveFile {
			memUsed -= inactiveFile
		} else if result.Memory.Cache > 0 && memUsed > result.Memory.Cache {
			memUsed -= result.Memory.Cache
		}
	}
	result.Memory.Usage = memUsed
	if result.Memory.Limit > 0 {
		result.Memory.Percent = float64(result.Memory.Usage) / float64(result.Memory.Limit) * 100.0
	}

	if len(stats.BlkioStats.IoServiceBytesRecursive) > 0 {
		for _, bio := range stats.BlkioStats.IoServiceBytesRecursive {
			switch bio.Op {
			case "read", "Read":
				result.BlockIO.ReadBytes += bio.Value
			case "write", "Write":
				result.BlockIO.WriteBytes += bio.Value
			}
		}
	}

	if stats.Networks != nil {
		for _, net := range stats.Networks {
			result.Network.RxBytes += net.RxBytes
			result.Network.TxBytes += net.TxBytes
			result.Network.RxPackets += net.RxPackets
			result.Network.TxPackets += net.TxPackets
			result.Network.RxErrors += net.RxErrors
			result.Network.TxErrors += net.TxErrors
		}
	}

	result.PIDs = int64(stats.PidsStats.Current)
	return result
}

type GroupedContainers struct {
	Groups               []ComposeGroup
	StandaloneContainers []ContainerInfo
}

func GroupContainersByCompose(containers []ContainerInfo) GroupedContainers {
	groupsMap := make(map[string]ComposeGroup)
	var standalone []ContainerInfo

	for _, c := range containers {
		if c.Project != "" {
			key := c.Project
			if group, exists := groupsMap[key]; exists {
				group.Containers = append(group.Containers, c)
				if c.workingDir != "" && group.WorkingDir == "" {
					group.WorkingDir = c.workingDir
				}
				groupsMap[key] = group
			} else {
				groupsMap[key] = ComposeGroup{
					Name:       c.Project,
					Project:    c.Project,
					WorkingDir: c.workingDir,
					Containers: []ContainerInfo{c},
				}
			}
		} else {
			standalone = append(standalone, c)
		}
	}

	var groups []ComposeGroup
	for _, g := range groupsMap {
		groups = append(groups, g)
	}

	return GroupedContainers{
		Groups:               groups,
		StandaloneContainers: standalone,
	}
}

func filterLabels(labels map[string]string) map[string]string {
	keepKeys := []string{
		"com.docker.compose.project",
		"com.docker.compose.service",
		"com.docker.compose.project.working_dir",
		"com.docker.compose.config-hash",
		"maintainer",
		"org.opencontainers.image.version",
		"org.opencontainers.image.revision",
		"org.opencontainers.image.source",
		"org.opencontainers.image.title",
	}

	filtered := make(map[string]string)
	for _, key := range keepKeys {
		if val, ok := labels[key]; ok {
			filtered[key] = val
		}
	}
	return filtered
}
