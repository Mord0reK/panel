package buffer

import (
	"database/sql"
	"log"
	"strings"
	"time"
)

type BulkInserter struct {
	db              *sql.DB
	manager         *BufferManager
	stopCh          chan struct{}
	lastFlushed     map[string]map[string]int64 // Track last flushed timestamp
	lastHostFlushed map[string]int64
}

func StartBulkInserter(db *sql.DB, manager *BufferManager) *BulkInserter {
	bi := &BulkInserter{
		db:              db,
		manager:         manager,
		stopCh:          make(chan struct{}),
		lastFlushed:     make(map[string]map[string]int64),
		lastHostFlushed: make(map[string]int64),
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
	buffers := bi.manager.GetAllBuffers()
	hostBuffers := bi.manager.GetAllHostBuffers()

	for agentID, containers := range buffers {
		if _, ok := bi.lastFlushed[agentID]; !ok {
			bi.lastFlushed[agentID] = make(map[string]int64)
		}

		for containerID, rb := range containers {
			lastTs := bi.lastFlushed[agentID][containerID]
			points := rb.GetPointsSince(lastTs)

			if len(points) >= 10 { // Plan says >= 10
				if err := bi.bulkInsert(agentID, containerID, points); err != nil {
					log.Printf("Failed to bulk insert metrics for %s/%s: %v", agentID, containerID, err)
				} else {
					// Update last flushed timestamp
					// Assuming points are sorted by timestamp, take the last one
					if len(points) > 0 {
						bi.lastFlushed[agentID][containerID] = points[len(points)-1].Timestamp
					}
				}
			}
		}
	}

	for agentID, rb := range hostBuffers {
		lastTs := bi.lastHostFlushed[agentID]
		points := rb.GetPointsSince(lastTs)

		if len(points) >= 10 {
			if err := bi.bulkInsertHost(agentID, points); err != nil {
				log.Printf("Failed to bulk insert host metrics for %s: %v", agentID, err)
			} else if len(points) > 0 {
				bi.lastHostFlushed[agentID] = points[len(points)-1].Timestamp
			}
		}
	}
}

func (bi *BulkInserter) bulkInsert(agentID, containerID string, points []MetricPoint) error {
	if len(points) == 0 {
		return nil
	}

	query := "INSERT INTO metrics_1s (agent_uuid, container_id, timestamp, cpu_percent, mem_used, mem_percent, disk_used, disk_percent, net_rx_bytes, net_tx_bytes) VALUES "
	vals := []interface{}{}
	placeholders := []string{}

	for _, p := range points {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		vals = append(vals, agentID, containerID, p.Timestamp, p.CPU, p.MemUsed, p.MemPercent, p.DiskUsed, p.DiskPercent, p.NetRx, p.NetTx)
	}

	query += strings.Join(placeholders, ",")

	// Retry logic for DB lock? Already implemented in execWithRetry if we used it,
	// but here we are using sql.DB directly.
	// We should probably expose execWithRetry or implement simple retry here.
	// For simplicity, just one try or simple loop.

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

	query := "INSERT INTO host_metrics_1s (agent_uuid, timestamp, cpu_percent, mem_used, mem_percent, disk_read_bytes_per_sec, disk_write_bytes_per_sec, net_rx_bytes_per_sec, net_tx_bytes_per_sec) VALUES "
	vals := []interface{}{}
	placeholders := []string{}

	for _, p := range points {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		vals = append(vals,
			agentID, p.Timestamp, p.CPU, p.MemUsed, p.MemPercent,
			p.DiskReadBytesPerSec, p.DiskWriteBytesPerSec, p.NetRxBytesPerSec, p.NetTxBytesPerSec,
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
