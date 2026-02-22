package metrics

import (
	"context"
	"strings"
	"sync"
	"time"

	"agent/internal/collector"
	"agent/internal/docker"

	"github.com/moby/moby/client"
	godisk "github.com/shirou/gopsutil/v4/disk"
	gonet "github.com/shirou/gopsutil/v4/net"
)

type HostSnapshot struct {
	Timestamp int64   `json:"timestamp"`
	CPU       float64 `json:"cpu_percent"`

	MemUsed    uint64  `json:"mem_used"`
	MemPercent float64 `json:"mem_percent"`

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
}

type containerCounters struct {
	Ts     time.Time
	NetRx  uint64
	NetTx  uint64
	DiskIO uint64
}

type SnapshotCollector struct {
	mu             sync.Mutex
	prevHost       *hostCounters
	prevContainers map[string]containerCounters
}

func NewSnapshotCollector() *SnapshotCollector {
	return &SnapshotCollector{
		prevContainers: make(map[string]containerCounters),
	}
}

func (c *SnapshotCollector) Collect(ctx context.Context, dockerCli *client.Client) (*Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	ts := now.Unix()

	sysMetrics, err := collector.CollectSystemMetrics(ctx)
	if err != nil {
		return nil, err
	}

	netRxTotal, netTxTotal := collectNetworkTotals(ctx)
	diskReadTotal, diskWriteTotal := collectDiskIOTotals(ctx)

	host := HostSnapshot{
		Timestamp:           ts,
		CPU:                 sysMetrics.CPU.Percent,
		MemUsed:             sysMetrics.Memory.Used,
		MemPercent:          sysMetrics.Memory.Percent,
		DiskReadBytesTotal:  diskReadTotal,
		DiskWriteBytesTotal: diskWriteTotal,
		NetRxBytesTotal:     netRxTotal,
		NetTxBytesTotal:     netTxTotal,
	}

	if c.prevHost != nil {
		elapsed := now.Sub(c.prevHost.Ts).Seconds()
		host.DiskReadBytesPerSec = toRate(diskReadTotal, c.prevHost.DiskRead, elapsed)
		host.DiskWriteBytesPerSec = toRate(diskWriteTotal, c.prevHost.DiskWrite, elapsed)
		host.NetRxBytesPerSec = toRate(netRxTotal, c.prevHost.NetRx, elapsed)
		host.NetTxBytesPerSec = toRate(netTxTotal, c.prevHost.NetTx, elapsed)
	}

	// Populate disk usage percent from the root partition (or fallback to first available)
	for _, d := range sysMetrics.Disk {
		if d.Mountpoint == "/" {
			host.DiskUsedPercent = d.Percent
			break
		}
	}
	if host.DiskUsedPercent == 0 && len(sysMetrics.Disk) > 0 {
		host.DiskUsedPercent = sysMetrics.Disk[0].Percent
	}

	c.prevHost = &hostCounters{
		Ts:        now,
		NetRx:     netRxTotal,
		NetTx:     netTxTotal,
		DiskRead:  diskReadTotal,
		DiskWrite: diskWriteTotal,
	}

	var containers []docker.RealtimeContainerInfo
	if dockerCli != nil {
		realtime, err := docker.CollectRealtimeContainerMetrics(ctx, dockerCli)
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

func (c *SnapshotCollector) normalizeContainerRates(now time.Time, ts int64, containers []docker.RealtimeContainerInfo) []docker.RealtimeContainerInfo {
	active := make(map[string]struct{}, len(containers))

	for i := range containers {
		id := containers[i].ContainerID
		active[id] = struct{}{}

		prev, ok := c.prevContainers[id]
		currNetRx := containers[i].NetRx
		currNetTx := containers[i].NetTx
		currDisk := containers[i].DiskUsed

		if ok {
			elapsed := now.Sub(prev.Ts).Seconds()
			containers[i].NetRx = toRate(currNetRx, prev.NetRx, elapsed)
			containers[i].NetTx = toRate(currNetTx, prev.NetTx, elapsed)
			containers[i].DiskUsed = toRate(currDisk, prev.DiskIO, elapsed)
		} else {
			containers[i].NetRx = 0
			containers[i].NetTx = 0
			containers[i].DiskUsed = 0
		}

		containers[i].Timestamp = ts
		c.prevContainers[id] = containerCounters{
			Ts:     now,
			NetRx:  currNetRx,
			NetTx:  currNetTx,
			DiskIO: currDisk,
		}
	}

	for id := range c.prevContainers {
		if _, ok := active[id]; !ok {
			delete(c.prevContainers, id)
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
	delta := current - previous
	return uint64(float64(delta) / elapsedSeconds)
}

func collectNetworkTotals(ctx context.Context) (uint64, uint64) {
	netStats, err := gonet.IOCountersWithContext(ctx, true)
	if err != nil {
		return 0, 0
	}

	var rx, tx uint64
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
	stats, err := godisk.IOCountersWithContext(ctx)
	if err != nil {
		return 0, 0
	}

	var read, write uint64
	for _, s := range stats {
		read += s.ReadBytes
		write += s.WriteBytes
	}
	return read, write
}

func ignoredInterface(iface string) bool {
	ignoredIfaces := map[string]bool{
		"lo":         true,
		"virbr":      true,
		"docker0":    true,
		"br-":        true,
		"veth":       true,
		"tailscale0": true,
	}
	if ignoredIfaces[iface] {
		return true
	}
	return strings.HasPrefix(iface, "br-") || strings.HasPrefix(iface, "veth") || strings.HasPrefix(iface, "virbr")
}
