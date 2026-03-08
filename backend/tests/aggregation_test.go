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

	// Insert data directly into metrics_15s
	// Use timestamps that are older than threshold (30s for 15s->30s level)
	now := time.Now().Unix()
	tsBase := ((now - 30) / 30) * 30 // align to 30s bucket, 30s old

	tx, _ := db.Begin()
	// Container metrics - 2 points at timestamps that fall in the SAME 30s bucket
	_, err = tx.Exec(`INSERT INTO metrics_15s
		(agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, disk_write_avg, disk_write_min, disk_write_max, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"agent-agg", "c-agg", tsBase,
		10.0, 10.0, 10.0,
		100.0, 100.0, 100.0,
		200.0, 0.0, 0.0, 0.0,
		1000.0, 1000.0, 1000.0,
		2000.0, 2000.0, 2000.0)
	require.NoError(t, err)

	_, err = tx.Exec(`INSERT INTO metrics_15s
		(agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, disk_write_avg, disk_write_min, disk_write_max, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"agent-agg", "c-agg", tsBase+10,
		30.0, 30.0, 30.0,
		100.0, 100.0, 100.0,
		200.0, 0.0, 0.0, 0.0,
		1000.0, 1000.0, 1000.0,
		2000.0, 2000.0, 2000.0)
	require.NoError(t, err)

	// Host metrics - same bucket
	_, err = tx.Exec(`INSERT INTO metrics_15s
		(agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, disk_write_avg, disk_write_min, disk_write_max, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max, disk_used_percent_avg, disk_used_percent_min, disk_used_percent_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"agent-agg", models.HostMainContainerID, tsBase,
		20.0, 20.0, 20.0,
		2048.0, 2048.0, 2048.0,
		300.0, 150.0, 150.0, 150.0,
		1200.0, 1200.0, 1200.0,
		900.0, 900.0, 900.0,
		50.0, 50.0, 50.0)
	require.NoError(t, err)

	_, err = tx.Exec(`INSERT INTO metrics_15s
		(agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, disk_write_avg, disk_write_min, disk_write_max, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max, disk_used_percent_avg, disk_used_percent_min, disk_used_percent_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"agent-agg", models.HostMainContainerID, tsBase+10,
		40.0, 40.0, 40.0,
		2048.0, 2048.0, 2048.0,
		300.0, 150.0, 150.0, 150.0,
		1200.0, 1200.0, 1200.0,
		900.0, 900.0, 900.0,
		60.0, 60.0, 60.0)
	require.NoError(t, err)
	tx.Commit()

	// Verify data exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_15s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 4, count)

	// Run Aggregation to next level (15s -> 30s)
	// Threshold for 15s->30s is 30s, our data is older than that (30s old)
	agg.ProcessAggregation()

	// Check metrics_30s - should have 2 rows (container + host)
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_30s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Check aggregated values: avg of (10+30)/2 = 20 for container
	var cpuAvg float64
	err = db.QueryRow("SELECT cpu_avg FROM metrics_30s WHERE container_id = 'c-agg'").Scan(&cpuAvg)
	require.NoError(t, err)
	assert.Equal(t, 20.0, cpuAvg, "container cpu_avg should be 20")

	// avg of (20+40)/2 = 30 for host
	err = db.QueryRow("SELECT cpu_avg FROM metrics_30s WHERE container_id = ?", models.HostMainContainerID).Scan(&cpuAvg)
	require.NoError(t, err)
	assert.Equal(t, 30.0, cpuAvg, "host cpu_avg should be 30")
}
