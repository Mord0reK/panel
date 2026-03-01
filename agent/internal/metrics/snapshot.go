package metrics

import (
	"context"
	"runtime"
	"sync"
	"time"

	"agent/internal/config"
	"agent/internal/docker"

	gocpu "github.com/shirou/gopsutil/v4/cpu"
	godisk "github.com/shirou/gopsutil/v4/disk"
	gomem "github.com/shirou/gopsutil/v4/mem"
	gonet "github.com/shirou/gopsutil/v4/net"
)

type HostSnapshot struct {
	Timestamp int64   `json:"timestamp"`
	CPU       float64 `json:"cpu_percent"`

	MemUsed     uint64  `json:"mem_used"`
	MemPercent  float64 `json:"mem_percent"`
	MemoryTotal uint64  `json:"memory_total"`

	DiskReadBytesPerSec  uint64  `json:"disk_read_bytes_per_sec"`
	DiskWriteBytesPerSec uint64  `json:"disk_write_bytes_per_sec"`
	NetRxBytesPerSec     uint64  `json:"net_rx_bytes_per_sec"`
	NetTxBytesPerSec     uint64  `json:"net_tx_bytes_per_sec"`
	DiskUsedPercent      float64 `json:"disk_used_percent"`

	DiskReadBytesTotal  uint64 `json:"disk_read_bytes_total"`
	DiskWriteBytesTotal uint64 `json:"disk_write_bytes_total"`
	NetRxBytesTotal     uint64 `json:"net_rx_bytes_total"`
	NetTxBytesTotal     uint64 `json:"net_tx_bytes_total"`
}

type Snapshot struct {
	Timestamp  int64                          `json:"timestamp"`
	Host       HostSnapshot                   `json:"host"`
	Containers []docker.RealtimeContainerInfo `json:"containers"`
}

type hostCounters struct {
	Ts        time.Time
	NetRx     uint64
	NetTx     uint64
	DiskRead  uint64
	DiskWrite uint64

	CPUTotal float64
	CPUIdle  float64
}

type containerCounters struct {
	Ts     time.Time
	NetRx  uint64
	NetTx  uint64
	DiskIO uint64
	// CPUUsec is the cumulative CPU usage in microseconds read from cgroup.
	// Used to compute per-tick CPU% without relying on Docker Stats API.
	CPUUsec uint64
}

type SnapshotCollector struct {
	mu             sync.Mutex
	prevHost       *hostCounters
	prevContainers map[string]containerCounters

	// registry is the source of container metadata. Populated at startup and
	// kept fresh by Docker events via registry.WatchEvents. nil disables
	// container metrics collection (useful in tests and when Docker is absent).
	registry *docker.ContainerRegistry

	// disk usage is cached and refreshed every diskCacheInterval.
	// Disk usage changes slowly — no need to call statfs() every second.
	cachedDiskPct float64
	lastDiskCheck time.Time

	// cgroupCaches holds per-container cgroup data that rarely changes.
	// Keyed by full container ID. Entries for stopped/removed containers
	// are pruned in normalizeContainerRates together with prevContainers.
	cgroupCaches map[string]*docker.CgroupCache
}

const diskCacheInterval = 60 * time.Second

func NewSnapshotCollector(registry *docker.ContainerRegistry) *SnapshotCollector {
	return &SnapshotCollector{
		registry:       registry,
		prevContainers: make(map[string]containerCounters),
		cgroupCaches:   make(map[string]*docker.CgroupCache),
	}
}

func (c *SnapshotCollector) Collect(ctx context.Context) (*Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	ts := now.Unix()

	// ── Memory ─────────────────────────────────────────────────────────────
	// Use fast direct /proc/meminfo read instead of gopsutil (3x faster)
	memUsed, memTotal, err := FastMemoryStats()
	if err != nil {
		// Fallback to gopsutil if direct read fails
		memStats, err := gomem.VirtualMemoryWithContext(ctx)
		if err != nil {
			return nil, err
		}
		memUsed = memStats.Used
		memTotal = memStats.Total
	}
	memPercent := float64(memUsed) / float64(memTotal) * 100

	// ── Network totals ─────────────────────────────────────────────────────
	netRxTotal, netTxTotal := collectNetworkTotals(ctx)

	// ── Disk I/O totals ────────────────────────────────────────────────────
	diskReadTotal, diskWriteTotal := collectDiskIOTotals(ctx)

	// ── CPU times ──────────────────────────────────────────────────────────
	// Use fast direct /proc/stat read instead of gopsutil (3x faster)
	cpuTotal, cpuIdle, err := FastCPUTimes()
	if err != nil {
		// Fallback to gopsutil if direct read fails
		cpuTimes, err := gocpu.TimesWithContext(ctx, false)
		if err == nil && len(cpuTimes) > 0 {
			cpuTotal = cpuTimes[0].Total()
			cpuIdle = cpuTimes[0].Idle
		}
	}

	host := HostSnapshot{
		Timestamp:           ts,
		CPU:                 0, // overwritten below once we have a previous sample
		MemUsed:             memUsed,
		MemPercent:          memPercent,
		MemoryTotal:         memTotal,
		DiskReadBytesTotal:  diskReadTotal,
		DiskWriteBytesTotal: diskWriteTotal,
		NetRxBytesTotal:     netRxTotal,
		NetTxBytesTotal:     netTxTotal,
		DiskUsedPercent:     c.getDiskUsedPercent(ctx),
	}

	if c.prevHost != nil {
		elapsed := now.Sub(c.prevHost.Ts).Seconds()
		if elapsed > 0 {
			deltaTotal := cpuTotal - c.prevHost.CPUTotal
			deltaIdle := cpuIdle - c.prevHost.CPUIdle
			if deltaTotal > 0 {
				host.CPU = (1 - deltaIdle/deltaTotal) * 100
			}
		}

		host.DiskReadBytesPerSec = toRate(diskReadTotal, c.prevHost.DiskRead, elapsed)
		host.DiskWriteBytesPerSec = toRate(diskWriteTotal, c.prevHost.DiskWrite, elapsed)
		host.NetRxBytesPerSec = toRate(netRxTotal, c.prevHost.NetRx, elapsed)
		host.NetTxBytesPerSec = toRate(netTxTotal, c.prevHost.NetTx, elapsed)
	}

	c.prevHost = &hostCounters{
		Ts:        now,
		NetRx:     netRxTotal,
		NetTx:     netTxTotal,
		DiskRead:  diskReadTotal,
		DiskWrite: diskWriteTotal,
		CPUTotal:  cpuTotal,
		CPUIdle:   cpuIdle,
	}

	var containers []docker.RealtimeContainerInfo
	if c.registry != nil {
		realtime, err := docker.CollectRealtimeContainerMetrics(ctx, c.registry, c.cgroupCaches)
		if err == nil && realtime != nil {
			containers = realtime.Containers
		}
	}

	containers = c.normalizeContainerRates(now, ts, containers)

	return &Snapshot{
		Timestamp:  ts,
		Host:       host,
		Containers: containers,
	}, nil
}

// getDiskUsedPercent returns the cached disk usage percent for the root
// partition, refreshing the cache at most once every diskCacheInterval.
// This avoids calling statfs() on every partition every second — on Docker
// hosts /proc/mounts contains overlay2 mounts for every running container.
// NOTE: must be called with c.mu held.
func (c *SnapshotCollector) getDiskUsedPercent(ctx context.Context) float64 {
	if time.Since(c.lastDiskCheck) < diskCacheInterval {
		return c.cachedDiskPct
	}

	// Pass all=true to obtain the Fstype field for filtering.
	parts, err := godisk.PartitionsWithContext(ctx, true)
	if err != nil {
		return c.cachedDiskPct
	}

	// Prefer the root filesystem; fall back to the first non-ignored entry.
	var fallbackPct float64
	foundFallback := false
	for _, part := range parts {
		if config.IsIgnoredMount(part.Mountpoint) || config.IsIgnoredFsType(part.Fstype) {
			continue
		}
		usage, err := godisk.UsageWithContext(ctx, part.Mountpoint)
		if err != nil {
			continue
		}
		if part.Mountpoint == "/" {
			c.cachedDiskPct = usage.UsedPercent
			c.lastDiskCheck = time.Now()
			return c.cachedDiskPct
		}
		if !foundFallback {
			fallbackPct = usage.UsedPercent
			foundFallback = true
		}
	}

	if foundFallback {
		c.cachedDiskPct = fallbackPct
		c.lastDiskCheck = time.Now()
	}
	return c.cachedDiskPct
}

func (c *SnapshotCollector) normalizeContainerRates(now time.Time, ts int64, containers []docker.RealtimeContainerInfo) []docker.RealtimeContainerInfo {
	// Number of logical CPUs on the host — used for cgroup CPU% calculation.
	numCPUs := float64(runtime.NumCPU())

	active := make(map[string]struct{}, len(containers))

	for i := range containers {
		id := containers[i].ContainerID
		active[id] = struct{}{}

		prev, ok := c.prevContainers[id]

		currNetRx := containers[i].NetRx
		currNetTx := containers[i].NetTx
		currDisk := containers[i].DiskUsed
		currCPUUsec := containers[i].CPURawUsec

		if ok {
			elapsed := now.Sub(prev.Ts).Seconds()

			// CPU% from cgroup delta.
			// Formula: (delta_cpu_usec / (elapsed_usec * num_cpus)) * 100
			if elapsed > 0 && currCPUUsec >= prev.CPUUsec {
				deltaCPU := float64(currCPUUsec - prev.CPUUsec)
				elapsedUsec := elapsed * 1_000_000
				containers[i].CPU = deltaCPU / (elapsedUsec * numCPUs) * 100.0
			}

			containers[i].NetRx = toRate(currNetRx, prev.NetRx, elapsed)
			containers[i].NetTx = toRate(currNetTx, prev.NetTx, elapsed)
			containers[i].DiskUsed = toRate(currDisk, prev.DiskIO, elapsed)
		} else {
			containers[i].CPU = 0
			containers[i].NetRx = 0
			containers[i].NetTx = 0
			containers[i].DiskUsed = 0
		}

		// Zero out the raw counter before sending over the wire.
		containers[i].CPURawUsec = 0
		containers[i].Timestamp = ts

		c.prevContainers[id] = containerCounters{
			Ts:      now,
			NetRx:   currNetRx,
			NetTx:   currNetTx,
			DiskIO:  currDisk,
			CPUUsec: currCPUUsec,
		}
	}

	for id := range c.prevContainers {
		if _, ok := active[id]; !ok {
			delete(c.prevContainers, id)
			delete(c.cgroupCaches, id)
		}
	}

	return containers
}

func toRate(current, previous uint64, elapsedSeconds float64) uint64 {
	if elapsedSeconds <= 0 {
		return 0
	}
	if current < previous {
		return 0
	}
	return uint64(float64(current-previous) / elapsedSeconds)
}

func collectNetworkTotals(ctx context.Context) (uint64, uint64) {
	// Build ignored interfaces map for fast lookup
	ignoredMap := make(map[string]bool)
	// Common ignored interfaces
	ignoredMap["lo"] = true

	// Try fast direct read first (2x faster than gopsutil)
	rx, tx, err := FastNetworkStats(ignoredMap)
	if err == nil {
		return rx, tx
	}

	// Fallback to gopsutil if direct read fails
	netStats, err := gonet.IOCountersWithContext(ctx, true)
	if err != nil {
		return 0, 0
	}

	rx, tx = 0, 0
	for _, stat := range netStats {
		if ignoredInterface(stat.Name) {
			continue
		}
		rx += stat.BytesRecv
		tx += stat.BytesSent
	}
	return rx, tx
}

func collectDiskIOTotals(ctx context.Context) (uint64, uint64) {
	// Try fast direct read first (2x faster than gopsutil)
	read, write, err := FastDiskIOStats()
	if err == nil {
		return read, write
	}

	// Fallback to gopsutil if direct read fails
	stats, err := godisk.IOCountersWithContext(ctx)
	if err != nil {
		return 0, 0
	}

	read, write = 0, 0
	for _, s := range stats {
		read += s.ReadBytes
		write += s.WriteBytes
	}
	return read, write
}

func ignoredInterface(iface string) bool {
	return config.IsIgnoredNetworkInterface(iface)
}
