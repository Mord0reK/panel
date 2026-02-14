package collector

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

type SystemMetrics struct {
	Timestamp time.Time      `json:"timestamp"`
	CPU       CPUStats       `json:"cpu"`
	Memory    MemoryStats    `json:"memory"`
	Disk      []DiskStats    `json:"disk"`
	Network   []NetworkStats `json:"network"`
}

type CPUStats struct {
	Percent       float64   `json:"percent"`
	Count         int       `json:"count"`
	PerCPUPercent []float64 `json:"per_cpu_percent"`
}

type CPUInfo struct {
	ModelName     string  `json:"model_name"`
	VendorID      string  `json:"vendor_id"`
	PhysicalCores int     `json:"physical_cores"`
	LogicalCores  int     `json:"logical_cores"`
	Mhz           float64 `json:"mhz"`
	CacheSize     int32   `json:"cache_size"`
}

type MemoryStats struct {
	Total     uint64  `json:"total"`
	Available uint64  `json:"available"`
	Used      uint64  `json:"used"`
	Percent   float64 `json:"percent"`
}

type MemoryInfo struct {
	Total     uint64 `json:"total"`
	SwapTotal uint64 `json:"swap_total"`
}

type DiskStats struct {
	Device     string  `json:"device"`
	Mountpoint string  `json:"mountpoint"`
	Total      uint64  `json:"total"`
	Free       uint64  `json:"free"`
	Used       uint64  `json:"used"`
	Percent    float64 `json:"percent"`
}

type NetworkStats struct {
	Interface   string `json:"interface"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
}

type SystemInfo struct {
	Hostname     string     `json:"hostname"`
	Platform     string     `json:"platform"`
	OS           string     `json:"os"`
	Kernel       string     `json:"kernel"`
	Architecture string     `json:"architecture"`
	CPU          CPUInfo    `json:"cpu"`
	Memory       MemoryInfo `json:"memory"`
	Uptime       uint64     `json:"uptime"`
	BootTime     uint64     `json:"boot_time"`
	NumProcs     int        `json:"num_procs"`
	HostID       string     `json:"host_id"`
}

func CollectSystemInfo(ctx context.Context) (*SystemInfo, error) {
	info := &SystemInfo{}

	hostInfo, err := host.InfoWithContext(ctx)
	if err == nil {
		info.Hostname = hostInfo.Hostname
		platform := capitalize(hostInfo.Platform)
		if hostInfo.PlatformVersion != "" {
			info.Platform = platform + " " + hostInfo.PlatformVersion
		} else {
			info.Platform = platform
		}
		info.OS = hostInfo.OS
		info.Kernel = hostInfo.KernelVersion
		info.Architecture = hostInfo.KernelArch
		info.BootTime = hostInfo.BootTime
		info.Uptime = hostInfo.Uptime
		info.HostID = hostInfo.HostID
		info.NumProcs = int(hostInfo.Procs)
	}

	kernelArch, err := host.KernelArch()
	if err == nil {
		info.Architecture = kernelArch
	}

	cpuInfo, err := cpu.InfoWithContext(ctx)
	if err == nil && len(cpuInfo) > 0 {
		info.CPU = CPUInfo{
			ModelName: cpuInfo[0].ModelName,
			VendorID:  cpuInfo[0].VendorID,
			Mhz:       cpuInfo[0].Mhz,
			CacheSize: cpuInfo[0].CacheSize,
		}
	}

	cpuCount, err := cpu.CountsWithContext(ctx, true)
	if err == nil {
		info.CPU.LogicalCores = cpuCount
	}
	cpuCountPhys, err := cpu.CountsWithContext(ctx, false)
	if err == nil {
		info.CPU.PhysicalCores = cpuCountPhys
	}

	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		info.Memory.Total = memInfo.Total
	}

	swapInfo, err := mem.SwapMemoryWithContext(ctx)
	if err == nil {
		info.Memory.SwapTotal = swapInfo.Total
	}

	return info, nil
}

func CollectSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return nil, err
	}
	metrics.CPU.Percent = 0
	if len(cpuPercent) > 0 {
		metrics.CPU.Percent = cpuPercent[0]
	}
	metrics.CPU.PerCPUPercent = cpuPercent

	cpuInfo, err := cpu.InfoWithContext(ctx)
	if err == nil {
		metrics.CPU.Count = len(cpuInfo)
	}

	memStats, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}
	metrics.Memory = MemoryStats{
		Total:     memStats.Total,
		Available: memStats.Available,
		Used:      memStats.Used,
		Percent:   memStats.UsedPercent,
	}

	diskParts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, err
	}
	ignoredMounts := map[string]bool{
		"/boot/efi": true,
		"/boot":     true,
		"/run":      true,
		"/run/lock": true,
		"/snap":     true,
		"/sys":      true,
		"/proc":     true,
		"/dev":      true,
		"/dev/shm":  true,
	}
	for _, part := range diskParts {
		if ignoredMounts[part.Mountpoint] {
			continue
		}
		usage, err := disk.UsageWithContext(ctx, part.Mountpoint)
		if err != nil {
			continue
		}
		metrics.Disk = append(metrics.Disk, DiskStats{
			Device:     part.Device,
			Mountpoint: part.Mountpoint,
			Total:      usage.Total,
			Free:       usage.Free,
			Used:       usage.Used,
			Percent:    usage.UsedPercent,
		})
	}

	netStats, err := net.IOCountersWithContext(ctx, true)
	if err != nil {
		return nil, err
	}
	ignoredIfaces := map[string]bool{
		"lo":         true,
		"virbr":      true,
		"docker0":    true,
		"br-":        true,
		"veth":       true,
		"tailscale0": true,
	}
	for _, stat := range netStats {
		if ignoredIfaces[stat.Name] || strings.HasPrefix(stat.Name, "br-") || strings.HasPrefix(stat.Name, "veth") || strings.HasPrefix(stat.Name, "virbr") {
			continue
		}
		metrics.Network = append(metrics.Network, NetworkStats{
			Interface:   stat.Name,
			BytesSent:   stat.BytesSent,
			BytesRecv:   stat.BytesRecv,
			PacketsSent: stat.PacketsSent,
			PacketsRecv: stat.PacketsRecv,
		})
	}

	return metrics, nil
}
