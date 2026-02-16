package tests

import (
	"testing"
	"time"

	"backend/internal/aggregation"

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

	// Insert 120 points to metrics_1s (2 minutes)
	// Aligned to 60s to have predictable buckets
	now := time.Now().Unix()
	startTs := (now / 60) * 60 - 180 // 3 minutes ago

	tx, _ := db.Begin()
	for i := 0; i < 120; i++ {
		ts := startTs + int64(i)
		// Alternating values: 10, 20, 10, 20...
		val := 10.0
		if i%2 == 1 {
			val = 20.0
		}
		_, err := tx.Exec(`INSERT INTO metrics_1s 
			(agent_uuid, container_id, timestamp, cpu_percent, mem_used, mem_percent, disk_used, disk_percent, net_rx_bytes, net_tx_bytes)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"agent-agg", "c-agg", ts, val, 100, 10.0, 200, 20.0, 1000, 2000)
		require.NoError(t, err)
	}
	tx.Commit()

	// Verify data exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_1s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 120, count)

	// Run Aggregation manually
	agg.ProcessAggregation()

	// Check metrics_5s
	// 120 points (1s) -> aggregated to 5s intervals.
	// 120s / 5s = 24 points expected.
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_5s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 24, count)

	// Check values
	var cpuAvg, cpuMin, cpuMax float64
	// Check a middle bucket to avoid edge effects
	err = db.QueryRow("SELECT cpu_avg, cpu_min, cpu_max FROM metrics_5s WHERE timestamp = ?", startTs+10).Scan(&cpuAvg, &cpuMin, &cpuMax)
	require.NoError(t, err)
	// Bucket starting at startTs+10: contains i=10,11,12,13,14
	// Values: i=10(10), 11(20), 12(10), 13(20), 14(10)
	// Sum: 70, Count: 5 -> Avg: 14.
	assert.Equal(t, 14.0, cpuAvg)
	assert.Equal(t, 10.0, cpuMin)
	assert.Equal(t, 20.0, cpuMax)

	// Check metrics_1s deleted
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_1s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
