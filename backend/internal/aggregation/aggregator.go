package aggregation

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

type Aggregator struct {
	db     *sql.DB
	stopCh chan struct{}
}

func NewAggregator(db *sql.DB) *Aggregator {
	a := &Aggregator{
		db:     db,
		stopCh: make(chan struct{}),
	}
	go a.Run()
	return a
}

func (a *Aggregator) Stop() {
	close(a.stopCh)
}

func (a *Aggregator) Run() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.ProcessAggregation()
		}
	}
}

func (a *Aggregator) ProcessAggregation() {
	now := time.Now().Unix()

	for _, level := range ContainerAggregationLevels {
		threshold := now - int64(level.SourceThreshold.Seconds())

		// 1. Fetch data to aggregate
		rows, err := a.fetchData(level.SourceTable, threshold)
		if err != nil {
			log.Printf("Failed to fetch data from %s: %v", level.SourceTable, err)
			continue
		}

		if len(rows) == 0 {
			continue
		}

		// 2. Aggregate
		aggregated := a.aggregateData(rows, level.AggregationInterval)

		// 3. Insert
		if len(aggregated) > 0 {
			if err := a.insertAggregated(level.TargetTable, aggregated); err != nil {
				log.Printf("Failed to insert aggregated data to %s: %v", level.TargetTable, err)
				continue
			}
		}

		// 4. Delete old data
		if err := a.deleteOldData(level.SourceTable, threshold); err != nil {
			log.Printf("Failed to delete old data from %s: %v", level.SourceTable, err)
		}
	}

	// Cleanup last level (>30 days)
	cleanupThreshold := now - int64((30 * 24 * time.Hour).Seconds())
	if err := a.deleteOldData("metrics_12h", cleanupThreshold); err != nil {
		log.Printf("Failed to cleanup %s: %v", "metrics_12h", err)
	}
}

type MetricRow struct {
	AgentUUID   string
	ContainerID string
	Timestamp   int64
	// Values to aggregate
	CPUAvg, CPUMin, CPUMax       float64
	MemAvg, MemMin, MemMax       float64
	DiskAvg                      float64
	NetRxAvg, NetRxMin, NetRxMax float64
	NetTxAvg, NetTxMin, NetTxMax float64
}

func (a *Aggregator) fetchData(table string, threshold int64) ([]MetricRow, error) {
	query := fmt.Sprintf(`SELECT agent_uuid, container_id, timestamp,
		cpu_avg, cpu_min, cpu_max,
		mem_avg, mem_min, mem_max,
		disk_avg,
		net_rx_avg, net_rx_min, net_rx_max,
		net_tx_avg, net_tx_min, net_tx_max
		FROM %s WHERE timestamp < ? ORDER BY timestamp`, table)

	rows, err := a.db.Query(query, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MetricRow
	for rows.Next() {
		var r MetricRow
		err := rows.Scan(
			&r.AgentUUID, &r.ContainerID, &r.Timestamp,
			&r.CPUAvg, &r.CPUMin, &r.CPUMax,
			&r.MemAvg, &r.MemMin, &r.MemMax,
			&r.DiskAvg,
			&r.NetRxAvg, &r.NetRxMin, &r.NetRxMax,
			&r.NetTxAvg, &r.NetTxMin, &r.NetTxMax,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

func (a *Aggregator) aggregateData(rows []MetricRow, interval time.Duration) []MetricRow {
	intervalSec := int64(interval.Seconds())
	groups := make(map[string]map[string]map[int64][]MetricRow) // agent -> container -> timestamp_bucket -> rows

	for _, r := range rows {
		bucket := (r.Timestamp / intervalSec) * intervalSec
		if _, ok := groups[r.AgentUUID]; !ok {
			groups[r.AgentUUID] = make(map[string]map[int64][]MetricRow)
		}
		if _, ok := groups[r.AgentUUID][r.ContainerID]; !ok {
			groups[r.AgentUUID][r.ContainerID] = make(map[int64][]MetricRow)
		}
		groups[r.AgentUUID][r.ContainerID][bucket] = append(groups[r.AgentUUID][r.ContainerID][bucket], r)
	}

	var result []MetricRow
	for agentID, containers := range groups {
		for containerID, buckets := range containers {
			for bucket, items := range buckets {
				agg := MetricRow{
					AgentUUID:   agentID,
					ContainerID: containerID,
					Timestamp:   bucket,
					CPUMin:      math.MaxFloat64, CPUMax: -math.MaxFloat64,
					MemMin: math.MaxFloat64, MemMax: -math.MaxFloat64,
					NetRxMin: math.MaxFloat64, NetRxMax: -math.MaxFloat64,
					NetTxMin: math.MaxFloat64, NetTxMax: -math.MaxFloat64,
				}

				var cpuSum, memSum, diskSum, netRxSum, netTxSum float64
				count := float64(len(items))

				for _, item := range items {
					cpuSum += item.CPUAvg
					if item.CPUMin < agg.CPUMin {
						agg.CPUMin = item.CPUMin
					}
					if item.CPUMax > agg.CPUMax {
						agg.CPUMax = item.CPUMax
					}

					memSum += item.MemAvg
					if item.MemMin < agg.MemMin {
						agg.MemMin = item.MemMin
					}
					if item.MemMax > agg.MemMax {
						agg.MemMax = item.MemMax
					}

					diskSum += item.DiskAvg

					netRxSum += item.NetRxAvg
					if item.NetRxMin < agg.NetRxMin {
						agg.NetRxMin = item.NetRxMin
					}
					if item.NetRxMax > agg.NetRxMax {
						agg.NetRxMax = item.NetRxMax
					}

					netTxSum += item.NetTxAvg
					if item.NetTxMin < agg.NetTxMin {
						agg.NetTxMin = item.NetTxMin
					}
					if item.NetTxMax > agg.NetTxMax {
						agg.NetTxMax = item.NetTxMax
					}
				}

				agg.CPUAvg = cpuSum / count
				agg.MemAvg = memSum / count
				agg.DiskAvg = diskSum / count
				agg.NetRxAvg = netRxSum / count
				agg.NetTxAvg = netTxSum / count

				result = append(result, agg)
			}
		}
	}
	return result
}

func (a *Aggregator) insertAggregated(table string, rows []MetricRow) error {
	if len(rows) == 0 {
		return nil
	}

	query := fmt.Sprintf(`INSERT OR REPLACE INTO %s (
		agent_uuid, container_id, timestamp,
		cpu_avg, cpu_min, cpu_max,
		mem_avg, mem_min, mem_max,
		disk_avg,
		net_rx_avg, net_rx_min, net_rx_max,
		net_tx_avg, net_tx_min, net_tx_max
	) VALUES `, table)

	vals := []interface{}{}
	placeholders := []string{}

	for _, r := range rows {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		vals = append(vals,
			r.AgentUUID, r.ContainerID, r.Timestamp,
			r.CPUAvg, r.CPUMin, r.CPUMax,
			r.MemAvg, r.MemMin, r.MemMax,
			r.DiskAvg,
			r.NetRxAvg, r.NetRxMin, r.NetRxMax,
			r.NetTxAvg, r.NetTxMin, r.NetTxMax,
		)
	}

	query += strings.Join(placeholders, ",")

	// Transaction
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(query, vals...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (a *Aggregator) deleteOldData(table string, threshold int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE timestamp < ?", table)
	_, err := a.db.Exec(query, threshold)
	return err
}
