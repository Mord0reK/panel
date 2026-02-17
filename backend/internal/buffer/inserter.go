package buffer

import (
	"database/sql"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"backend/internal/models"
)

type aggregatedMetricRow struct {
	Timestamp int64

	CPUAvg float64
	CPUMin float64
	CPUMax float64

	MemAvg float64
	MemMin float64
	MemMax float64

	DiskAvg float64

	NetRxAvg float64
	NetRxMin float64
	NetRxMax float64

	NetTxAvg float64
	NetTxMin float64
	NetTxMax float64
}

type aggregatedHostMetricRow struct {
	Timestamp int64

	CPUAvg float64
	CPUMin float64
	CPUMax float64

	MemUsedAvg float64
	MemUsedMin float64
	MemUsedMax float64

	DiskReadBytesPerSecAvg float64
	DiskReadBytesPerSecMin float64
	DiskReadBytesPerSecMax float64

	DiskWriteBytesPerSecAvg float64
	DiskWriteBytesPerSecMin float64
	DiskWriteBytesPerSecMax float64

	NetRxBytesPerSecAvg float64
	NetRxBytesPerSecMin float64
	NetRxBytesPerSecMax float64

	NetTxBytesPerSecAvg float64
	NetTxBytesPerSecMin float64
	NetTxBytesPerSecMax float64
}

type BulkInserter struct {
	db      *sql.DB
	manager *BufferManager
	stopCh  chan struct{}
}

func StartBulkInserter(db *sql.DB, manager *BufferManager) *BulkInserter {
	bi := &BulkInserter{
		db:      db,
		manager: manager,
		stopCh:  make(chan struct{}),
	}
	go bi.Run()
	return bi
}

func (bi *BulkInserter) Run() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bi.stopCh:
			// Final flush?
			bi.Flush()
			return
		case <-ticker.C:
			bi.Flush()
		}
	}
}

func (bi *BulkInserter) Stop() {
	close(bi.stopCh)
}

func (bi *BulkInserter) Flush() {
	pending := bi.manager.DrainPendingMetrics()
	pendingHost := bi.manager.DrainPendingHostMetrics()
	if len(pending) == 0 && len(pendingHost) == 0 {
		return
	}

	failed := make(map[string]map[string][]MetricPoint)
	failedHost := make(map[string][]HostMetricPoint)

	for agentID, containers := range pending {
		for containerID, points := range containers {
			if len(points) == 0 {
				continue
			}

			if err := bi.bulkInsert(agentID, containerID, points); err != nil {
				log.Printf("Failed to bulk insert metrics for %s/%s: %v", agentID, containerID, err)
				if _, ok := failed[agentID]; !ok {
					failed[agentID] = make(map[string][]MetricPoint)
				}
				failed[agentID][containerID] = points
			}
		}
	}

	for agentID, points := range pendingHost {
		if len(points) == 0 {
			continue
		}

		if err := bi.bulkInsertHost(agentID, points); err != nil {
			log.Printf("Failed to bulk insert host metrics for %s: %v", agentID, err)
			failedHost[agentID] = points
		}
	}

	if len(failed) > 0 {
		bi.manager.RequeuePendingMetrics(failed)
	}

	if len(failedHost) > 0 {
		bi.manager.RequeuePendingHostMetrics(failedHost)
	}
}

func (bi *BulkInserter) bulkInsert(agentID, containerID string, points []MetricPoint) error {
	if len(points) == 0 {
		return nil
	}

	rows := aggregateTo5s(points)
	if len(rows) == 0 {
		return nil
	}

	query := `INSERT OR REPLACE INTO metrics_5s (
		agent_uuid, container_id, timestamp,
		cpu_avg, cpu_min, cpu_max,
		mem_avg, mem_min, mem_max,
		disk_avg,
		net_rx_avg, net_rx_min, net_rx_max,
		net_tx_avg, net_tx_min, net_tx_max
	) VALUES `
	vals := []interface{}{}
	placeholders := []string{}

	for _, row := range rows {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		vals = append(vals,
			agentID,
			containerID,
			row.Timestamp,
			row.CPUAvg,
			row.CPUMin,
			row.CPUMax,
			row.MemAvg,
			row.MemMin,
			row.MemMax,
			row.DiskAvg,
			row.NetRxAvg,
			row.NetRxMin,
			row.NetRxMax,
			row.NetTxAvg,
			row.NetTxMin,
			row.NetTxMax,
		)
	}

	query += strings.Join(placeholders, ",")

	tx, err := bi.db.Begin()
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

func (bi *BulkInserter) bulkInsertHost(agentID string, points []HostMetricPoint) error {
	if len(points) == 0 {
		return nil
	}

	rows := aggregateHostTo5s(points)
	if len(rows) == 0 {
		return nil
	}

	query := `INSERT OR REPLACE INTO metrics_5s (
		agent_uuid, container_id, timestamp,
		cpu_avg, cpu_min, cpu_max,
		mem_avg, mem_min, mem_max,
		disk_avg,
		net_rx_avg, net_rx_min, net_rx_max,
		net_tx_avg, net_tx_min, net_tx_max
	) VALUES `
	vals := []interface{}{}
	placeholders := []string{}

	for _, row := range rows {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		vals = append(vals,
			agentID,
			models.HostMainContainerID,
			row.Timestamp,
			row.CPUAvg,
			row.CPUMin,
			row.CPUMax,
			row.MemUsedAvg,
			row.MemUsedMin,
			row.MemUsedMax,
			row.DiskReadBytesPerSecAvg,
			row.NetRxBytesPerSecAvg,
			row.NetRxBytesPerSecMin,
			row.NetRxBytesPerSecMax,
			row.NetTxBytesPerSecAvg,
			row.NetTxBytesPerSecMin,
			row.NetTxBytesPerSecMax,
		)

		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		vals = append(vals,
			agentID,
			models.HostDiskWriteContainerID,
			row.Timestamp,
			0.0,
			0.0,
			0.0,
			0.0,
			0.0,
			0.0,
			0.0,
			row.DiskWriteBytesPerSecAvg,
			row.DiskWriteBytesPerSecMin,
			row.DiskWriteBytesPerSecMax,
			0.0,
			0.0,
			0.0,
		)
	}

	query += strings.Join(placeholders, ",")

	tx, err := bi.db.Begin()
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

func aggregateTo5s(points []MetricPoint) []aggregatedMetricRow {
	const intervalSec int64 = 5

	type accumulator struct {
		count int

		cpuSum float64
		cpuMin float64
		cpuMax float64

		memSum float64
		memMin float64
		memMax float64

		diskSum float64

		netRxSum float64
		netRxMin float64
		netRxMax float64

		netTxSum float64
		netTxMin float64
		netTxMax float64
	}

	buckets := make(map[int64]*accumulator)
	for _, point := range points {
		bucket := (point.Timestamp / intervalSec) * intervalSec
		acc, ok := buckets[bucket]
		if !ok {
			acc = &accumulator{
				cpuMin:   math.MaxFloat64,
				cpuMax:   -math.MaxFloat64,
				memMin:   math.MaxFloat64,
				memMax:   -math.MaxFloat64,
				netRxMin: math.MaxFloat64,
				netRxMax: -math.MaxFloat64,
				netTxMin: math.MaxFloat64,
				netTxMax: -math.MaxFloat64,
			}
			buckets[bucket] = acc
		}

		cpu := point.CPU
		mem := float64(point.MemUsed)
		disk := float64(point.DiskUsed)
		netRx := float64(point.NetRx)
		netTx := float64(point.NetTx)

		acc.count++
		acc.cpuSum += cpu
		if cpu < acc.cpuMin {
			acc.cpuMin = cpu
		}
		if cpu > acc.cpuMax {
			acc.cpuMax = cpu
		}

		acc.memSum += mem
		if mem < acc.memMin {
			acc.memMin = mem
		}
		if mem > acc.memMax {
			acc.memMax = mem
		}

		acc.diskSum += disk

		acc.netRxSum += netRx
		if netRx < acc.netRxMin {
			acc.netRxMin = netRx
		}
		if netRx > acc.netRxMax {
			acc.netRxMax = netRx
		}

		acc.netTxSum += netTx
		if netTx < acc.netTxMin {
			acc.netTxMin = netTx
		}
		if netTx > acc.netTxMax {
			acc.netTxMax = netTx
		}
	}

	bucketKeys := make([]int64, 0, len(buckets))
	for bucket := range buckets {
		bucketKeys = append(bucketKeys, bucket)
	}
	sort.Slice(bucketKeys, func(i, j int) bool {
		return bucketKeys[i] < bucketKeys[j]
	})

	result := make([]aggregatedMetricRow, 0, len(bucketKeys))
	for _, bucket := range bucketKeys {
		acc := buckets[bucket]
		count := float64(acc.count)

		result = append(result, aggregatedMetricRow{
			Timestamp: bucket,
			CPUAvg:    acc.cpuSum / count,
			CPUMin:    acc.cpuMin,
			CPUMax:    acc.cpuMax,
			MemAvg:    acc.memSum / count,
			MemMin:    acc.memMin,
			MemMax:    acc.memMax,
			DiskAvg:   acc.diskSum / count,
			NetRxAvg:  acc.netRxSum / count,
			NetRxMin:  acc.netRxMin,
			NetRxMax:  acc.netRxMax,
			NetTxAvg:  acc.netTxSum / count,
			NetTxMin:  acc.netTxMin,
			NetTxMax:  acc.netTxMax,
		})
	}

	return result
}

func aggregateHostTo5s(points []HostMetricPoint) []aggregatedHostMetricRow {
	const intervalSec int64 = 5

	type accumulator struct {
		count int

		cpuSum float64
		cpuMin float64
		cpuMax float64

		memUsedSum float64
		memUsedMin float64
		memUsedMax float64

		diskReadSum float64
		diskReadMin float64
		diskReadMax float64

		diskWriteSum float64
		diskWriteMin float64
		diskWriteMax float64

		netRxSum float64
		netRxMin float64
		netRxMax float64

		netTxSum float64
		netTxMin float64
		netTxMax float64
	}

	buckets := make(map[int64]*accumulator)
	for _, point := range points {
		bucket := (point.Timestamp / intervalSec) * intervalSec
		acc, ok := buckets[bucket]
		if !ok {
			acc = &accumulator{
				cpuMin:       math.MaxFloat64,
				cpuMax:       -math.MaxFloat64,
				memUsedMin:   math.MaxFloat64,
				memUsedMax:   -math.MaxFloat64,
				diskReadMin:  math.MaxFloat64,
				diskReadMax:  -math.MaxFloat64,
				diskWriteMin: math.MaxFloat64,
				diskWriteMax: -math.MaxFloat64,
				netRxMin:     math.MaxFloat64,
				netRxMax:     -math.MaxFloat64,
				netTxMin:     math.MaxFloat64,
				netTxMax:     -math.MaxFloat64,
			}
			buckets[bucket] = acc
		}

		cpu := point.CPU
		memUsed := float64(point.MemUsed)
		diskRead := float64(point.DiskReadBytesPerSec)
		diskWrite := float64(point.DiskWriteBytesPerSec)
		netRx := float64(point.NetRxBytesPerSec)
		netTx := float64(point.NetTxBytesPerSec)

		acc.count++

		acc.cpuSum += cpu
		if cpu < acc.cpuMin {
			acc.cpuMin = cpu
		}
		if cpu > acc.cpuMax {
			acc.cpuMax = cpu
		}

		acc.memUsedSum += memUsed
		if memUsed < acc.memUsedMin {
			acc.memUsedMin = memUsed
		}
		if memUsed > acc.memUsedMax {
			acc.memUsedMax = memUsed
		}

		acc.diskReadSum += diskRead
		if diskRead < acc.diskReadMin {
			acc.diskReadMin = diskRead
		}
		if diskRead > acc.diskReadMax {
			acc.diskReadMax = diskRead
		}

		acc.diskWriteSum += diskWrite
		if diskWrite < acc.diskWriteMin {
			acc.diskWriteMin = diskWrite
		}
		if diskWrite > acc.diskWriteMax {
			acc.diskWriteMax = diskWrite
		}

		acc.netRxSum += netRx
		if netRx < acc.netRxMin {
			acc.netRxMin = netRx
		}
		if netRx > acc.netRxMax {
			acc.netRxMax = netRx
		}

		acc.netTxSum += netTx
		if netTx < acc.netTxMin {
			acc.netTxMin = netTx
		}
		if netTx > acc.netTxMax {
			acc.netTxMax = netTx
		}
	}

	bucketKeys := make([]int64, 0, len(buckets))
	for bucket := range buckets {
		bucketKeys = append(bucketKeys, bucket)
	}
	sort.Slice(bucketKeys, func(i, j int) bool {
		return bucketKeys[i] < bucketKeys[j]
	})

	result := make([]aggregatedHostMetricRow, 0, len(bucketKeys))
	for _, bucket := range bucketKeys {
		acc := buckets[bucket]
		count := float64(acc.count)

		result = append(result, aggregatedHostMetricRow{
			Timestamp:               bucket,
			CPUAvg:                  acc.cpuSum / count,
			CPUMin:                  acc.cpuMin,
			CPUMax:                  acc.cpuMax,
			MemUsedAvg:              acc.memUsedSum / count,
			MemUsedMin:              acc.memUsedMin,
			MemUsedMax:              acc.memUsedMax,
			DiskReadBytesPerSecAvg:  acc.diskReadSum / count,
			DiskReadBytesPerSecMin:  acc.diskReadMin,
			DiskReadBytesPerSecMax:  acc.diskReadMax,
			DiskWriteBytesPerSecAvg: acc.diskWriteSum / count,
			DiskWriteBytesPerSecMin: acc.diskWriteMin,
			DiskWriteBytesPerSecMax: acc.diskWriteMax,
			NetRxBytesPerSecAvg:     acc.netRxSum / count,
			NetRxBytesPerSecMin:     acc.netRxMin,
			NetRxBytesPerSecMax:     acc.netRxMax,
			NetTxBytesPerSecAvg:     acc.netTxSum / count,
			NetTxBytesPerSecMin:     acc.netTxMin,
			NetTxBytesPerSecMax:     acc.netTxMax,
		})
	}

	return result
}
