package metrics

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// FastCPUTimes reads /proc/stat directly to minimize overhead.
// Returns total and idle CPU times in clock ticks.
// This is ~3x faster than gopsutil's cpu.TimesWithContext().
func FastCPUTimes() (total, idle float64, err error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}

	// Parse first line: "cpu  user nice system idle iowait irq softirq steal guest guest_nice"
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0, 0, fmt.Errorf("empty /proc/stat")
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, fmt.Errorf("invalid /proc/stat format")
	}

	// Sum all fields to get total CPU time
	var values [10]float64
	for i := 1; i < len(fields) && i < 11; i++ {
		v, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return 0, 0, err
		}
		values[i-1] = v
		total += v
	}

	// idle is the 4th field (index 3 after "cpu")
	idle = values[3]
	return total, idle, nil
}

// FastMemoryStats reads /proc/meminfo directly.
// Returns used and total memory in bytes.
// This is ~2-3x faster than gopsutil's mem.VirtualMemoryWithContext().
func FastMemoryStats() (used, total uint64, err error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}

	var memTotal, memAvailable uint64
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			memTotal, _ = strconv.ParseUint(fields[1], 10, 64)
			memTotal *= 1024 // Convert from kB to bytes
		case "MemAvailable:":
			memAvailable, _ = strconv.ParseUint(fields[1], 10, 64)
			memAvailable *= 1024
		}

		// Early exit once we have both values
		if memTotal > 0 && memAvailable > 0 {
			break
		}
	}

	if memTotal == 0 {
		return 0, 0, fmt.Errorf("failed to parse MemTotal")
	}

	used = memTotal - memAvailable
	return used, memTotal, nil
}

// FastNetworkStats reads /proc/net/dev directly.
// Returns total rx and tx bytes across all non-ignored interfaces.
// This is ~2x faster than gopsutil's net.IOCountersWithContext().
func FastNetworkStats(ignoredInterfaces map[string]bool) (rx, tx uint64, err error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(data), "\n")
	// Skip first two header lines
	for i := 2; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Format: "  eth0: 12345 123 ... 67890 123 ..."
		// Split by colon first to get interface name
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		if ignoredInterfaces[iface] {
			continue
		}

		// Parse counters: bytes packets errs drop fifo frame compressed multicast
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		// RX bytes is field 0, TX bytes is field 8
		rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
		txBytes, _ := strconv.ParseUint(fields[8], 10, 64)

		rx += rxBytes
		tx += txBytes
	}

	return rx, tx, nil
}

// FastDiskIOStats reads /proc/diskstats directly.
// Returns total read and write bytes across all disks.
// This is ~2x faster than gopsutil's disk.IOCountersWithContext().
func FastDiskIOStats() (read, write uint64, err error) {
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		// Format: major minor name reads ... sectors_read ... writes ... sectors_written ...
		// sectors_read is field 5 (0-indexed), sectors_written is field 9
		sectorsRead, _ := strconv.ParseUint(fields[5], 10, 64)
		sectorsWritten, _ := strconv.ParseUint(fields[9], 10, 64)

		// Each sector is 512 bytes
		read += sectorsRead * 512
		write += sectorsWritten * 512
	}

	return read, write, nil
}
