package docker

import (
	"context"
	"encoding/json"
	"fmt"
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
	workingDir  string                 // internal use for compose group
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
	Timestamp   int64   `json:"timestamp"`
	CPU         float64 `json:"cpu_percent"`
	MemUsed     uint64  `json:"mem_used"`
	MemPercent  float64 `json:"mem_percent"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskPercent float64 `json:"disk_percent"`
	NetRx       uint64  `json:"net_rx_bytes"`
	NetTx       uint64  `json:"net_tx_bytes"`
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

func NewDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return cli, nil
}

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

func CollectRealtimeContainerMetrics(ctx context.Context, cli *client.Client) (*RealtimeContainerMetrics, error) {
	metrics := &RealtimeContainerMetrics{
		Timestamp: time.Now(),
	}

	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: false})
	if err != nil {
		return nil, fmt.Errorf("failed to list running containers: %w", err)
	}

	var wg sync.WaitGroup
	results := make(chan RealtimeContainerInfo, len(containers.Items))

	for _, c := range containers.Items {
		if string(c.State) != "running" {
			continue
		}

		wg.Add(1)
		go func(containerID, containerName, containerImage string) {
			defer wg.Done()

			name := containerName
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}

			info := RealtimeContainerInfo{
				ContainerID: containerID[:12],
				Name:        name,
				State:       "running",
				Image:       containerImage,
				Timestamp:   time.Now().Unix(),
			}

			statsResp, err := cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{Stream: false})
			if err == nil {
				var stats container.StatsResponse
				if err := json.NewDecoder(statsResp.Body).Decode(&stats); err == nil {
					cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
					systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
					onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
					if onlineCPUs == 0 {
						onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
					}
					if systemDelta > 0 && cpuDelta > 0 {
						info.CPU = (cpuDelta / systemDelta) * onlineCPUs * 100.0
					}

					memUsed := stats.MemoryStats.Usage
					if stats.MemoryStats.Stats != nil {
						if inactiveFile, ok := stats.MemoryStats.Stats["inactive_file"]; ok && memUsed > inactiveFile {
							memUsed -= inactiveFile
						} else if cache, ok := stats.MemoryStats.Stats["cache"]; ok && memUsed > cache {
							memUsed -= cache
						}
					}
					info.MemUsed = memUsed
					memLimit := stats.MemoryStats.Limit
					if memLimit > 0 {
						info.MemPercent = float64(info.MemUsed) / float64(memLimit) * 100.0
					}

					if len(stats.BlkioStats.IoServiceBytesRecursive) > 0 {
						for _, bio := range stats.BlkioStats.IoServiceBytesRecursive {
							if bio.Op == "read" || bio.Op == "Read" {
								info.DiskUsed += bio.Value
							} else if bio.Op == "write" || bio.Op == "Write" {
								info.DiskUsed += bio.Value
							}
						}
					}

					if stats.Networks != nil {
						for _, net := range stats.Networks {
							info.NetRx += net.RxBytes
							info.NetTx += net.TxBytes
						}
					}
				}
				_ = statsResp.Body.Close()
			}

			results <- info
		}(c.ID, c.Names[0], c.Image)
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
	// docker stats subtracts cache (inactive_file) from usage, we do the same
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
