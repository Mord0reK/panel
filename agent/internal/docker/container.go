package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
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

// CgroupCache holds per-container data that rarely changes between ticks.
// Caching this avoids redundant syscalls on every collection cycle.
//
//   - cgroupPath — computed once via os.Stat; stable for the container's lifetime.
//   - memMax     — container memory limit; changes only via "docker update".
//   - pid        — init process PID from cgroup.threads; refreshed with a TTL
//     because it changes on container restart.
//
// Persistent file descriptors (fdCPUStat, fdMemCurrent, etc.) are kept open
// across ticks. Each tick does Seek(0)+Read() instead of open()+read()+close(),
// eliminating ~4 syscalls per file per container per second.
// On any I/O error the FD is closed and set to nil; the next tick reopens it.
type CgroupCache struct {
	cgroupPath string
	memMax     uint64
	memMaxSet  bool
	pid        string
	pidChecked time.Time

	// Persistent FDs — kept open to avoid open()/close() per tick.
	fdCPUStat       *os.File // <cgroup>/cpu.stat
	fdMemCurrent    *os.File // <cgroup>/memory.current
	fdMemStat       *os.File // <cgroup>/memory.stat
	fdIOStat        *os.File // <cgroup>/io.stat
	fdCgroupThreads *os.File // <cgroup>/cgroup.threads
	fdNetDev        *os.File // /proc/<pid>/net/dev
	fdNetDevPid     string   // pid for which fdNetDev is currently open

	// readBuf is reused across all reads within a single tick.
	// Safe: each CgroupCache is accessed by exactly one goroutine at a time
	// (keyed by container ID, no two goroutines share the same cache).
	readBuf [16384]byte
}

// Close releases all open file descriptors held by this cache.
// Must be called when the associated container is removed from the registry.
func (c *CgroupCache) Close() {
	for _, fd := range []*os.File{
		c.fdCPUStat, c.fdMemCurrent, c.fdMemStat, c.fdIOStat, c.fdCgroupThreads, c.fdNetDev,
	} {
		if fd != nil {
			fd.Close()
		}
	}
	c.fdCPUStat = nil
	c.fdMemCurrent = nil
	c.fdMemStat = nil
	c.fdIOStat = nil
	c.fdCgroupThreads = nil
	c.fdNetDev = nil
	c.fdNetDevPid = ""
}

// cgroupPIDCacheTTL is how long we reuse a cached container PID before
// re-reading cgroup.threads. A container restart changes the PID; 5 s gives
// us at most one missed net-stats tick after a restart.
const cgroupPIDCacheTTL = 5 * time.Second

// readVirtualFileFD reads a virtual (kernel-generated) file via a persistent
// file descriptor. On the first call it opens the file and stores the FD in
// *fd. Subsequent calls do Seek(0)+Read() instead of open()+read()+close(),
// eliminating two syscalls per file per tick.
//
// On any I/O error the FD is closed and set to nil; the next call reopens it.
// buf is a caller-supplied scratch buffer; the returned slice is a sub-slice of buf.
func readVirtualFileFD(fd **os.File, path string, buf []byte) ([]byte, error) {
	if *fd == nil {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		*fd = f
	}
	if _, err := (*fd).Seek(0, io.SeekStart); err != nil {
		(*fd).Close()
		*fd = nil
		return nil, err
	}
	n, err := (*fd).Read(buf)
	if err != nil && err != io.EOF {
		(*fd).Close()
		*fd = nil
		return nil, err
	}
	return buf[:n], nil
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

// parseUint64Bytes parses a decimal uint64 from a byte slice without allocations.
// Skips leading/trailing whitespace. Returns (0, false) on empty input or parse error.
func parseUint64Bytes(b []byte) (uint64, bool) {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t' || b[0] == '\n' || b[0] == '\r') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t' || b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	if len(b) == 0 {
		return 0, false
	}
	var v uint64
	for _, c := range b {
		if c < '0' || c > '9' {
			return v, false
		}
		v = v*10 + uint64(c-'0')
	}
	return v, true
}

// scanLine returns the first line from data (without newline) and the remaining bytes.
func scanLine(data []byte) (line, rest []byte) {
	nl := bytes.IndexByte(data, '\n')
	if nl < 0 {
		return data, nil
	}
	return data[:nl], data[nl+1:]
}

// findField scans a "key value\n" kernel file and returns the value bytes for key.
// key must include the trailing space (e.g., []byte("usage_usec ")).
// No allocations — operates entirely on the input byte slice.
func findField(data, key []byte) ([]byte, bool) {
	for len(data) > 0 {
		var line []byte
		line, data = scanLine(data)
		if bytes.HasPrefix(line, key) {
			return line[len(key):], true
		}
	}
	return nil, false
}

// readCgroupStatsWithCache populates RealtimeContainerInfo from cgroup v2 files,
// using cache to skip syscalls that would return the same value every tick.
//
// CPU is returned as a raw cumulative microsecond counter (CPURawUsec);
// the actual CPU% is computed by SnapshotCollector using delta between ticks.
func readCgroupStatsWithCache(containerID string, info *RealtimeContainerInfo, cache *CgroupCache) {
	// ── cgroup path ────────────────────────────────────────────────────
	// Resolved once: the systemd/legacy choice is fixed for the container's
	// lifetime, so we skip the os.Stat() on every subsequent tick.
	if cache.cgroupPath == "" {
		cache.cgroupPath = cgroupBasePath(containerID)
	}
	base := cache.cgroupPath

	// ── CPU ────────────────────────────────────────────────────────────
	if data, err := readVirtualFileFD(&cache.fdCPUStat, base+"/cpu.stat", cache.readBuf[:]); err == nil {
		if val, ok := findField(data, []byte("usage_usec ")); ok {
			if usec, ok := parseUint64Bytes(val); ok {
				info.CPURawUsec = usec
			}
		}
	}

	// ── Memory ─────────────────────────────────────────────────────────
	if memData, err := readVirtualFileFD(&cache.fdMemCurrent, base+"/memory.current", cache.readBuf[:]); err == nil {
		if memUsed, ok := parseUint64Bytes(memData); ok {
			// Subtract inactive_file (page cache) the same way docker stats does.
			if statData, err := readVirtualFileFD(&cache.fdMemStat, base+"/memory.stat", cache.readBuf[:]); err == nil {
				if val, ok := findField(statData, []byte("inactive_file ")); ok {
					if inactive, ok := parseUint64Bytes(val); ok && memUsed > inactive {
						memUsed -= inactive
					}
				}
			}
			info.MemUsed = memUsed

			// memory.max (container limit) is stable — read once and cache.
			// It changes only via "docker update --memory", which is extremely rare.
			if !cache.memMaxSet {
				if limitData, err := os.ReadFile(base + "/memory.max"); err == nil {
					limitStr := strings.TrimSpace(string(limitData))
					if limitStr != "max" {
						if limit, err := strconv.ParseUint(limitStr, 10, 64); err == nil && limit > 0 {
							cache.memMax = limit
							cache.memMaxSet = true
						}
					}
				}
			}
			if cache.memMaxSet && cache.memMax > 0 {
				info.MemPercent = float64(memUsed) / float64(cache.memMax) * 100.0
			}
		}
	}

	// ── Block I/O ──────────────────────────────────────────────────────
	if data, err := readVirtualFileFD(&cache.fdIOStat, base+"/io.stat", cache.readBuf[:]); err == nil {
		rbytesKey := []byte("rbytes=")
		wbytesKey := []byte("wbytes=")
		remaining := data
		for len(remaining) > 0 {
			var line []byte
			line, remaining = scanLine(remaining)
			for len(line) > 0 {
				// skip leading spaces
				for len(line) > 0 && line[0] == ' ' {
					line = line[1:]
				}
				// extract next field
				sp := bytes.IndexByte(line, ' ')
				var field []byte
				if sp >= 0 {
					field = line[:sp]
					line = line[sp:]
				} else {
					field = line
					line = nil
				}
				if bytes.HasPrefix(field, rbytesKey) {
					if v, ok := parseUint64Bytes(field[len(rbytesKey):]); ok {
						info.DiskUsed += v
					}
				} else if bytes.HasPrefix(field, wbytesKey) {
					if v, ok := parseUint64Bytes(field[len(wbytesKey):]); ok {
						info.DiskUsed += v
					}
				}
			}
		}
	}

	// ── Network ────────────────────────────────────────────────────────
	// cgroup v2 does not expose network counters — we read /proc/<pid>/net/dev
	// where <pid> is the container's init thread from cgroup.threads.
	// The PID is cached with cgroupPIDCacheTTL: it changes only on restart,
	// at which point we pick up the new PID within one TTL period.
	if cache.pid == "" || time.Since(cache.pidChecked) > cgroupPIDCacheTTL {
		if data, err := readVirtualFileFD(&cache.fdCgroupThreads, base+"/cgroup.threads", cache.readBuf[:]); err == nil {
			line, _ := scanLine(data)
			line = bytes.TrimSpace(line)
			if len(line) > 0 {
				cache.pid = string(line) // one alloc per TTL period (every 5s max)
				cache.pidChecked = time.Now()
			}
		}
	}
	if cache.pid != "" {
		readProcNetDev(cache.pid, cache, info)
	}
}

// readProcNetDev reads network counters from /proc/<pid>/net/dev for the
// network namespace of the given PID (i.e. the container's netns).
// It uses a persistent FD from cache to avoid open()/close() per tick.
// When the PID changes (container restart) the old FD is closed and reopened.
// Parsing is zero-allocation: operates entirely on the cache.readBuf byte slice.
func readProcNetDev(pid string, cache *CgroupCache, info *RealtimeContainerInfo) {
	// If PID changed (container restart), close the stale FD.
	if cache.fdNetDevPid != pid {
		if cache.fdNetDev != nil {
			cache.fdNetDev.Close()
			cache.fdNetDev = nil
			cache.fdNetDevPid = ""
		}
	}

	path := fmt.Sprintf("/proc/%s/net/dev", pid)
	data, err := readVirtualFileFD(&cache.fdNetDev, path, cache.readBuf[:])
	if err != nil {
		return
	}
	cache.fdNetDevPid = pid

	// Skip the two header lines.
	for i := 0; i < 2; i++ {
		nl := bytes.IndexByte(data, '\n')
		if nl < 0 {
			return
		}
		data = data[nl+1:]
	}

	loBytes := []byte("lo")
	for len(data) > 0 {
		var line []byte
		line, data = scanLine(data)

		// trim leading spaces
		for len(line) > 0 && line[0] == ' ' {
			line = line[1:]
		}
		if len(line) == 0 {
			continue
		}

		// Parse up to 10 whitespace-separated fields.
		// Field 0: "iface:"  Field 1: rx_bytes  Field 9: tx_bytes
		var fields [10][]byte
		n := 0
		rem := line
		for n < 10 && len(rem) > 0 {
			for len(rem) > 0 && rem[0] == ' ' {
				rem = rem[1:]
			}
			if len(rem) == 0 {
				break
			}
			sp := bytes.IndexByte(rem, ' ')
			if sp < 0 {
				fields[n] = rem
				n++
				break
			}
			fields[n] = rem[:sp]
			rem = rem[sp:]
			n++
		}
		if n < 10 {
			continue
		}

		iface := fields[0]
		// strip trailing colon from interface name
		if len(iface) > 0 && iface[len(iface)-1] == ':' {
			iface = iface[:len(iface)-1]
		}
		if bytes.Equal(iface, loBytes) {
			continue
		}

		rx, _ := parseUint64Bytes(fields[1])
		tx, _ := parseUint64Bytes(fields[9])
		info.NetRx += rx
		info.NetTx += tx
	}
}

// CollectRealtimeContainerMetrics collects per-container metrics using cgroup v2
// filesystem reads instead of the Docker Stats API.
// CPU is returned as CPURawUsec (cumulative); SnapshotCollector computes the %.
//
// cgroupCaches is a caller-managed map keyed by full container ID. Entries are
// created here for new containers; callers are responsible for pruning entries
// for containers that no longer exist.
// CollectRealtimeContainerMetrics collects per-container metrics using cgroup v2
// filesystem reads. It uses the ContainerRegistry cache (populated at startup and
// kept fresh by Docker events) instead of calling ContainerList every second,
// eliminating the Docker API call from the hot metrics path.
func CollectRealtimeContainerMetrics(ctx context.Context, registry *ContainerRegistry, cgroupCaches map[string]*CgroupCache) (*RealtimeContainerMetrics, error) {
	metrics := &RealtimeContainerMetrics{
		Timestamp: time.Now(),
	}

	containers := registry.List()

	// Pre-populate cache entries for containers we haven't seen before.
	// Must happen before goroutines are spawned to avoid concurrent map writes.
	for _, c := range containers {
		if _, ok := cgroupCaches[c.ID]; !ok {
			cgroupCaches[c.ID] = &CgroupCache{}
		}
	}

	var wg sync.WaitGroup
	results := make(chan RealtimeContainerInfo, len(containers))

	// Worker pool with semaphore to limit concurrent goroutines.
	// Reduces scheduler overhead and syscall contention by processing
	// containers in batches of 8 instead of spawning N goroutines at once.
	const maxWorkers = 8
	semaphore := make(chan struct{}, maxWorkers)

	for _, c := range containers {
		// Each goroutine receives its own cache pointer — no concurrent access
		// to the same CgroupCache since container IDs are unique.
		cache := cgroupCaches[c.ID]
		wg.Add(1)

		// Acquire semaphore before spawning goroutine
		semaphore <- struct{}{}

		go func(cID, cName, cImage, cState, cStatus string, labels map[string]string, cache *CgroupCache) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
			}()

			info := RealtimeContainerInfo{
				ContainerID: cID[:12],
				Name:        cName,
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
				readCgroupStatsWithCache(cID, &info, cache)
			}

			results <- info
		}(c.ID, c.Name, c.Image, c.State, c.Status, c.Labels, cache)
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
