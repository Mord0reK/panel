package models

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	HostMainContainerID = "__host__"
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

type RawHostMetricPoint struct {
	Timestamp            int64   `json:"timestamp"`
	CPU                  float64 `json:"cpu"`
	MemUsed              uint64  `json:"mem_used"`
	MemPercent           float64 `json:"mem_percent"`
	DiskReadBytesPerSec  uint64  `json:"disk_read_bytes_per_sec"`
	DiskWriteBytesPerSec uint64  `json:"disk_write_bytes_per_sec"`
	NetRxBytesPerSec     uint64  `json:"net_rx_bytes_per_sec"`
	NetTxBytesPerSec     uint64  `json:"net_tx_bytes_per_sec"`
}

type HistoricalMetricPoint struct {
	Timestamp    int64   `json:"timestamp"`
	CPUAvg       float64 `json:"cpu_avg"`
	CPUMin       float64 `json:"cpu_min"`
	CPUMax       float64 `json:"cpu_max"`
	MemAvg       float64 `json:"mem_avg"`
	MemMin       float64 `json:"mem_min"`
	MemMax       float64 `json:"mem_max"`
	DiskAvg      float64 `json:"disk_avg"`
	DiskWriteAvg float64 `json:"disk_write_avg"`
	DiskWriteMin float64 `json:"disk_write_min"`
	DiskWriteMax float64 `json:"disk_write_max"`
	NetRxAvg     float64 `json:"net_rx_avg"`
	NetRxMin     float64 `json:"net_rx_min"`
	NetRxMax     float64 `json:"net_rx_max"`
	NetTxAvg     float64 `json:"net_tx_avg"`
	NetTxMin     float64 `json:"net_tx_min"`
	NetTxMax     float64 `json:"net_tx_max"`
}

type HistoricalHostMetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	CPUAvg    float64 `json:"cpu_avg"`
	CPUMin    float64 `json:"cpu_min"`
	CPUMax    float64 `json:"cpu_max"`

	MemUsedAvg float64 `json:"mem_used_avg"`
	MemUsedMin float64 `json:"mem_used_min"`
	MemUsedMax float64 `json:"mem_used_max"`

	DiskReadBytesPerSecAvg float64 `json:"disk_read_bytes_per_sec_avg"`
	DiskReadBytesPerSecMin float64 `json:"disk_read_bytes_per_sec_min"`
	DiskReadBytesPerSecMax float64 `json:"disk_read_bytes_per_sec_max"`

	DiskWriteBytesPerSecAvg float64 `json:"disk_write_bytes_per_sec_avg"`
	DiskWriteBytesPerSecMin float64 `json:"disk_write_bytes_per_sec_min"`
	DiskWriteBytesPerSecMax float64 `json:"disk_write_bytes_per_sec_max"`

	NetRxBytesPerSecAvg float64 `json:"net_rx_bytes_per_sec_avg"`
	NetRxBytesPerSecMin float64 `json:"net_rx_bytes_per_sec_min"`
	NetRxBytesPerSecMax float64 `json:"net_rx_bytes_per_sec_max"`

	NetTxBytesPerSecAvg float64 `json:"net_tx_bytes_per_sec_avg"`
	NetTxBytesPerSecMin float64 `json:"net_tx_bytes_per_sec_min"`
	NetTxBytesPerSecMax float64 `json:"net_tx_bytes_per_sec_max"`

	DiskUsedPercentAvg float64 `json:"disk_used_percent_avg"`
	DiskUsedPercentMin float64 `json:"disk_used_percent_min"`
	DiskUsedPercentMax float64 `json:"disk_used_percent_max"`
}

type rangeConfig struct {
	TableSuffix string
	Duration    time.Duration
}

var rangeMap = map[string]rangeConfig{
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

	query := fmt.Sprintf(`
		SELECT timestamp,
			cpu_avg, cpu_min, cpu_max,
			mem_avg, mem_min, mem_max,
			disk_avg, disk_write_avg, disk_write_min, disk_write_max,
			net_rx_avg, net_rx_min, net_rx_max,
			net_tx_avg, net_tx_min, net_tx_max
		FROM %s WHERE agent_uuid=? AND container_id=? AND timestamp>=? AND timestamp<=?
		ORDER BY timestamp ASC`, table)

	rows, err := db.Query(query, agentUUID, containerID, fromTs, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []HistoricalMetricPoint
	for rows.Next() {
		var p HistoricalMetricPoint
		err := rows.Scan(
			&p.Timestamp,
			&p.CPUAvg, &p.CPUMin, &p.CPUMax,
			&p.MemAvg, &p.MemMin, &p.MemMax,
			&p.DiskAvg, &p.DiskWriteAvg, &p.DiskWriteMin, &p.DiskWriteMax,
			&p.NetRxAvg, &p.NetRxMin, &p.NetRxMax,
			&p.NetTxAvg, &p.NetTxMin, &p.NetTxMax,
		)
		if err != nil {
			return nil, err
		}
		points = append(points, p)
	}

	return points, rows.Err()
}

func GetHistoricalHostMetrics(db *sql.DB, agentUUID, rangeKey string) ([]HistoricalHostMetricPoint, error) {
	cfg, ok := rangeMap[rangeKey]
	if !ok {
		return nil, fmt.Errorf("invalid range: %s", rangeKey)
	}

	now := time.Now().Unix()
	fromTs := now - int64(cfg.Duration.Seconds())
	table := fmt.Sprintf("metrics_%s", cfg.TableSuffix)

	query := fmt.Sprintf(`
		SELECT timestamp,
			cpu_avg, cpu_min, cpu_max,
			mem_avg, mem_min, mem_max,
			disk_avg, disk_write_avg, disk_write_min, disk_write_max,
			net_rx_avg, net_rx_min, net_rx_max,
			net_tx_avg, net_tx_min, net_tx_max,
			disk_used_percent_avg, disk_used_percent_min, disk_used_percent_max
		FROM %s WHERE agent_uuid=? AND container_id=? AND timestamp>=? AND timestamp<=?
		ORDER BY timestamp ASC`, table)

	rows, err := db.Query(query, agentUUID, HostMainContainerID, fromTs, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []HistoricalHostMetricPoint
	for rows.Next() {
		var p HistoricalHostMetricPoint
		err := rows.Scan(
			&p.Timestamp,
			&p.CPUAvg, &p.CPUMin, &p.CPUMax,
			&p.MemUsedAvg, &p.MemUsedMin, &p.MemUsedMax,
			&p.DiskReadBytesPerSecAvg, &p.DiskWriteBytesPerSecAvg, &p.DiskWriteBytesPerSecMin, &p.DiskWriteBytesPerSecMax,
			&p.NetRxBytesPerSecAvg, &p.NetRxBytesPerSecMin, &p.NetRxBytesPerSecMax,
			&p.NetTxBytesPerSecAvg, &p.NetTxBytesPerSecMin, &p.NetTxBytesPerSecMax,
			&p.DiskUsedPercentAvg, &p.DiskUsedPercentMin, &p.DiskUsedPercentMax,
		)
		if err != nil {
			return nil, err
		}
		// disk_avg stores disk_read; replicate min/max from avg for backward compatibility.
		p.DiskReadBytesPerSecMin = p.DiskReadBytesPerSecAvg
		p.DiskReadBytesPerSecMax = p.DiskReadBytesPerSecAvg
		points = append(points, p)
	}

	return points, rows.Err()
}
