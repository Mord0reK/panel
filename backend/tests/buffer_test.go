package tests

import (
	"testing"

	"backend/internal/buffer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBuffer(t *testing.T) {
	rb := buffer.NewRingBuffer(5)

	// Add 3 points
	rb.Add(buffer.MetricPoint{Timestamp: 1})
	rb.Add(buffer.MetricPoint{Timestamp: 2})
	rb.Add(buffer.MetricPoint{Timestamp: 3})

	points := rb.GetAll()
	assert.Len(t, points, 3)
	assert.Equal(t, int64(1), points[0].Timestamp)
	assert.Equal(t, int64(3), points[2].Timestamp)

	// Add more to overflow
	rb.Add(buffer.MetricPoint{Timestamp: 4})
	rb.Add(buffer.MetricPoint{Timestamp: 5}) // Full now
	rb.Add(buffer.MetricPoint{Timestamp: 6}) // Overwrite 1
	rb.Add(buffer.MetricPoint{Timestamp: 7}) // Overwrite 2

	points = rb.GetAll()
	assert.Len(t, points, 5)
	assert.Equal(t, int64(3), points[0].Timestamp) // Oldest is 3
	assert.Equal(t, int64(7), points[4].Timestamp) // Newest is 7

	// GetPointsSince
	newPoints := rb.GetPointsSince(4)
	assert.Len(t, newPoints, 3) // 5, 6, 7
	assert.Equal(t, int64(5), newPoints[0].Timestamp)
	assert.Equal(t, int64(7), newPoints[2].Timestamp)
}

func TestBufferManager(t *testing.T) {
	bm := buffer.NewBufferManager()

	bm.AddMetric("agent1", "c1", buffer.MetricPoint{Timestamp: 100})
	bm.AddMetric("agent1", "c2", buffer.MetricPoint{Timestamp: 200})

	buffers := bm.GetAllBuffers()
	assert.Len(t, buffers, 1)
	assert.Len(t, buffers["agent1"], 2)

	rb1 := bm.GetOrCreate("agent1", "c1")
	assert.Equal(t, int64(100), rb1.GetAll()[0].Timestamp)
}

func TestBulkInserter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Need to manually insert server and container to satisfy FOREIGN KEYS if any
	// Actually schema has FOREIGN KEYS. So we must insert server.
	// But `metrics_1s` references `servers(uuid)` and `containers(agent_uuid, container_id)`?
	// Schema check:
	// FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ON DELETE CASCADE
	// Does it reference containers?
	// Schema says: `CREATE TABLE metrics_1s (..., PRIMARY KEY(agent_uuid, container_id, timestamp), FOREIGN KEY(agent_uuid) REFERENCES servers(uuid) ...)`
	// It does NOT enforce container existence in `containers` table, only server existence.
	// Good.

	_, err := db.Exec("INSERT INTO servers (uuid, approved) VALUES (?, ?)", "agent-bulk", true)
	require.NoError(t, err)

	bm := buffer.NewBufferManager()
	inserter := buffer.StartBulkInserter(db, bm)
	defer inserter.Stop()

	// Add 12 metrics (threshold is 10)
	for i := 0; i < 12; i++ {
		ts := int64(1000 + i)
		bm.AddMetric("agent-bulk", "c-bulk", buffer.MetricPoint{
			Timestamp: ts,
			CPU:       10.5,
			MemUsed:   1024,
			DiskUsed:  2048,
			NetRx:     100,
			NetTx:     200,
		})
		bm.AddHostMetric("agent-bulk", buffer.HostMetricPoint{
			Timestamp:            ts,
			CPU:                  30.0,
			MemUsed:              2048,
			MemPercent:           15.0,
			DiskReadBytesPerSec:  100,
			DiskWriteBytesPerSec: 200,
			NetRxBytesPerSec:     300,
			NetTxBytesPerSec:     400,
		})
	}

	// Manually flush
	inserter.Flush()

	// Verify DB
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_1s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 12, count)

	err = db.QueryRow("SELECT COUNT(*) FROM host_metrics_1s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 12, count)
}
