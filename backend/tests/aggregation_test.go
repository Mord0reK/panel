package tests

import (
	"testing"
	"time"

	"backend/internal/aggregation"
	"backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert servers
	_, err := db.Exec("INSERT INTO servers (uuid, approved) VALUES (?, ?)", "agent-agg", true)
	require.NoError(t, err)

	agg := aggregation.NewAggregator(db)
	defer agg.Stop()

	// Insert 36 points to metrics_5s (3 minutes, one point every 5s)
	// Data is intentionally older than 5 minutes to pass SourceThreshold for metrics_5s -> metrics_15s
	now := time.Now().Unix()
	startTs := ((now - int64((10 * time.Minute).Seconds())) / 15) * 15

	tx, _ := db.Begin()
	for i := 0; i < 36; i++ {
		ts := startTs + int64(i*5)
		val := 10.0
		if i%2 == 1 {
			val = 20.0
		}
		_, err := tx.Exec(`INSERT INTO metrics_5s
			(agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"agent-agg", "c-agg", ts,
			val, val, val,
			100.0, 100.0, 100.0,
			200.0,
			1000.0, 1000.0, 1000.0,
			2000.0, 2000.0, 2000.0)
		require.NoError(t, err)
	}
	for i := 0; i < 36; i++ {
		ts := startTs + int64(i*5)
		val := 30.0
		if i%2 == 1 {
			val = 50.0
		}
		_, err := tx.Exec(`INSERT INTO metrics_5s
			(agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"agent-agg", models.HostMainContainerID, ts,
			val, val, val,
			2048.0, 2048.0, 2048.0,
			300.0,
			1200.0, 1200.0, 1200.0,
			900.0, 900.0, 900.0,
		)
		require.NoError(t, err)

		_, err = tx.Exec(`INSERT INTO metrics_5s
			(agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"agent-agg", models.HostDiskWriteContainerID, ts,
			0.0, 0.0, 0.0,
			0.0, 0.0, 0.0,
			0.0,
			150.0, 150.0, 150.0,
			0.0, 0.0, 0.0,
		)
		require.NoError(t, err)
	}
	tx.Commit()

	// Verify data exists (36 container + 36 host + 36 host_disk_write = 108)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_5s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 108, count)

	// Run Aggregation manually
	agg.ProcessAggregation()

	// Check metrics_15s
	// 36 points (5s) -> aggregated to 15s intervals => 12 points expected.
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_15s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 36, count)
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_15s WHERE container_id=?", models.HostMainContainerID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 12, count)
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_15s WHERE container_id=?", models.HostDiskWriteContainerID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 12, count)

	// Check values
	var cpuAvg, cpuMin, cpuMax float64
	err = db.QueryRow("SELECT cpu_avg, cpu_min, cpu_max FROM metrics_15s WHERE timestamp = ?", startTs+15).Scan(&cpuAvg, &cpuMin, &cpuMax)
	require.NoError(t, err)
	assert.Greater(t, cpuAvg, 10.0)
	assert.Less(t, cpuAvg, 20.0)
	assert.Equal(t, 10.0, cpuMin)
	assert.Equal(t, 20.0, cpuMax)

	// Check old metrics_5s deleted
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_5s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
