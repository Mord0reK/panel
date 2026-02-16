package models

import (
	"database/sql"
	"fmt"
	"time"
)

type RawMetricPoint struct {
	Timestamp  int64   `json:"timestamp"`
	CPU        float64 `json:"cpu"`
	MemUsed    uint64  `json:"mem_used"`
	MemPercent float64 `json:"mem_percent"`
	DiskUsed   uint64  `json:"disk_used"`
	NetRxBytes uint64  `json:"net_rx_bytes"`
	NetTxBytes uint64  `json:"net_tx_bytes"`
}

type HistoricalMetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	CPUAvg    float64 `json:"cpu_avg"`
	CPUMin    float64 `json:"cpu_min"`
	CPUMax    float64 `json:"cpu_max"`
	MemAvg    float64 `json:"mem_avg"`
	MemMin    float64 `json:"mem_min"`
	MemMax    float64 `json:"mem_max"`
	DiskAvg   float64 `json:"disk_avg"`
	NetRxAvg  float64 `json:"net_rx_avg"`
	NetRxMin  float64 `json:"net_rx_min"`
	NetRxMax  float64 `json:"net_rx_max"`
	NetTxAvg  float64 `json:"net_tx_avg"`
	NetTxMin  float64 `json:"net_tx_min"`
	NetTxMax  float64 `json:"net_tx_max"`
}

type HostHistoricalMetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	CPUAvg    float64 `json:"cpu_avg"`
	CPUMin    float64 `json:"cpu_min"`
	CPUMax    float64 `json:"cpu_max"`
	MemAvg    float64 `json:"mem_avg"`
	MemMin    float64 `json:"mem_min"`
	MemMax    float64 `json:"mem_max"`

	DiskReadAvg float64 `json:"disk_read_avg"`
	DiskReadMin float64 `json:"disk_read_min"`
	DiskReadMax float64 `json:"disk_read_max"`

	DiskWriteAvg float64 `json:"disk_write_avg"`
	DiskWriteMin float64 `json:"disk_write_min"`
	DiskWriteMax float64 `json:"disk_write_max"`

	NetRxAvg float64 `json:"net_rx_avg"`
	NetRxMin float64 `json:"net_rx_min"`
	NetRxMax float64 `json:"net_rx_max"`
	NetTxAvg float64 `json:"net_tx_avg"`
	NetTxMin float64 `json:"net_tx_min"`
	NetTxMax float64 `json:"net_tx_max"`
}

type rangeConfig struct {
	TableSuffix string
	Duration    time.Duration
}

var rangeMap = map[string]rangeConfig{
	"1m":  {TableSuffix: "1s", Duration: 1 * time.Minute},
	"5m":  {TableSuffix: "5s", Duration: 5 * time.Minute},
	"15m": {TableSuffix: "15s", Duration: 15 * time.Minute},
	"30m": {TableSuffix: "30s", Duration: 30 * time.Minute},
	"1h":  {TableSuffix: "1m", Duration: 1 * time.Hour},
	"6h":  {TableSuffix: "5m", Duration: 6 * time.Hour},
	"12h": {TableSuffix: "15m", Duration: 12 * time.Hour},
	"24h": {TableSuffix: "30m", Duration: 24 * time.Hour},
	"7d":  {TableSuffix: "1h", Duration: 7 * 24 * time.Hour},
	"15d": {TableSuffix: "6h", Duration: 15 * 24 * time.Hour},
	"30d": {TableSuffix: "12h", Duration: 30 * 24 * time.Hour},
}

func GetHistoricalMetrics(db *sql.DB, agentUUID, containerID, rangeKey string) ([]HistoricalMetricPoint, error) {
	cfg, ok := rangeMap[rangeKey]
	if !ok {
		return nil, fmt.Errorf("invalid range: %s", rangeKey)
	}

	now := time.Now().Unix()
	fromTs := now - int64(cfg.Duration.Seconds())
	table := fmt.Sprintf("metrics_%s", cfg.TableSuffix)
	isRawData := cfg.TableSuffix == "1s"

	var query string
	if isRawData {
		query = fmt.Sprintf(`
			SELECT timestamp, cpu_percent, mem_used, mem_percent, disk_used, net_rx_bytes, net_tx_bytes
			FROM %s WHERE agent_uuid=? AND container_id=? AND timestamp>=? AND timestamp<=?
			ORDER BY timestamp ASC`, table)
	} else {
		query = fmt.Sprintf(`
			SELECT timestamp,
				cpu_avg, cpu_min, cpu_max,
				mem_avg, mem_min, mem_max,
				disk_avg,
				net_rx_avg, net_rx_min, net_rx_max,
				net_tx_avg, net_tx_min, net_tx_max
			FROM %s WHERE agent_uuid=? AND container_id=? AND timestamp>=? AND timestamp<=?
			ORDER BY timestamp ASC`, table)
	}

	rows, err := db.Query(query, agentUUID, containerID, fromTs, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []HistoricalMetricPoint
	for rows.Next() {
		var p HistoricalMetricPoint
		if isRawData {
			var timestamp int64
			var cpu, memPercent float64
			var memUsed, diskUsed, netRxBytes, netTxBytes uint64
			err := rows.Scan(
				&timestamp,
				&cpu,
				&memUsed,
				&memPercent,
				&diskUsed,
				&netRxBytes,
				&netTxBytes,
			)
			if err != nil {
				return nil, err
			}
			p.Timestamp = timestamp
			p.CPUAvg = cpu
			p.CPUMin = cpu
			p.CPUMax = cpu
			p.MemAvg = float64(memUsed)
			p.MemMin = float64(memUsed)
			p.MemMax = float64(memUsed)
			p.DiskAvg = float64(diskUsed)
			p.NetRxAvg = float64(netRxBytes)
			p.NetRxMin = float64(netRxBytes)
			p.NetRxMax = float64(netRxBytes)
			p.NetTxAvg = float64(netTxBytes)
			p.NetTxMin = float64(netTxBytes)
			p.NetTxMax = float64(netTxBytes)
		} else {
			err := rows.Scan(
				&p.Timestamp,
				&p.CPUAvg, &p.CPUMin, &p.CPUMax,
				&p.MemAvg, &p.MemMin, &p.MemMax,
				&p.DiskAvg,
				&p.NetRxAvg, &p.NetRxMin, &p.NetRxMax,
				&p.NetTxAvg, &p.NetTxMin, &p.NetTxMax,
			)
			if err != nil {
				return nil, err
			}
		}
		points = append(points, p)
	}

	return points, rows.Err()
}

func GetHostHistoricalMetrics(db *sql.DB, agentUUID, rangeKey string) ([]HostHistoricalMetricPoint, error) {
	cfg, ok := rangeMap[rangeKey]
	if !ok {
		return nil, fmt.Errorf("invalid range: %s", rangeKey)
	}

	now := time.Now().Unix()
	fromTs := now - int64(cfg.Duration.Seconds())
	table := fmt.Sprintf("host_metrics_%s", cfg.TableSuffix)
	isRawData := cfg.TableSuffix == "1s"

	var query string
	if isRawData {
		query = fmt.Sprintf(`
			SELECT timestamp, cpu_percent, mem_used, mem_percent,
				disk_read_bytes_per_sec, disk_write_bytes_per_sec,
				net_rx_bytes_per_sec, net_tx_bytes_per_sec
			FROM %s WHERE agent_uuid=? AND timestamp>=? AND timestamp<=?
			ORDER BY timestamp ASC`, table)
	} else {
		query = fmt.Sprintf(`
			SELECT timestamp,
				cpu_avg, cpu_min, cpu_max,
				mem_avg, mem_min, mem_max,
				disk_read_avg, disk_read_min, disk_read_max,
				disk_write_avg, disk_write_min, disk_write_max,
				net_rx_avg, net_rx_min, net_rx_max,
				net_tx_avg, net_tx_min, net_tx_max
			FROM %s WHERE agent_uuid=? AND timestamp>=? AND timestamp<=?
			ORDER BY timestamp ASC`, table)
	}

	rows, err := db.Query(query, agentUUID, fromTs, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []HostHistoricalMetricPoint
	for rows.Next() {
		var p HostHistoricalMetricPoint
		if isRawData {
			var timestamp int64
			var cpu, memPercent float64
			var memUsed uint64
			var diskRead, diskWrite, netRx, netTx float64
			err := rows.Scan(
				&timestamp,
				&cpu,
				&memUsed,
				&memPercent,
				&diskRead,
				&diskWrite,
				&netRx,
				&netTx,
			)
			if err != nil {
				return nil, err
			}
			p.Timestamp = timestamp
			p.CPUAvg, p.CPUMin, p.CPUMax = cpu, cpu, cpu
			p.MemAvg, p.MemMin, p.MemMax = float64(memUsed), float64(memUsed), float64(memUsed)
			p.DiskReadAvg, p.DiskReadMin, p.DiskReadMax = diskRead, diskRead, diskRead
			p.DiskWriteAvg, p.DiskWriteMin, p.DiskWriteMax = diskWrite, diskWrite, diskWrite
			p.NetRxAvg, p.NetRxMin, p.NetRxMax = netRx, netRx, netRx
			p.NetTxAvg, p.NetTxMin, p.NetTxMax = netTx, netTx, netTx
		} else {
			err := rows.Scan(
				&p.Timestamp,
				&p.CPUAvg, &p.CPUMin, &p.CPUMax,
				&p.MemAvg, &p.MemMin, &p.MemMax,
				&p.DiskReadAvg, &p.DiskReadMin, &p.DiskReadMax,
				&p.DiskWriteAvg, &p.DiskWriteMin, &p.DiskWriteMax,
				&p.NetRxAvg, &p.NetRxMin, &p.NetRxMax,
				&p.NetTxAvg, &p.NetTxMin, &p.NetTxMax,
			)
			if err != nil {
				return nil, err
			}
		}

		points = append(points, p)
	}

	return points, rows.Err()
}
