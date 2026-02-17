package tests

import (
	"testing"

	"backend/internal/buffer"
	"backend/internal/models"

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

	// Need server row to satisfy FOREIGN KEY(agent_uuid) for metrics tables.

	_, err := db.Exec("INSERT INTO servers (uuid, approved) VALUES (?, ?)", "agent-bulk", true)
	require.NoError(t, err)

	bm := buffer.NewBufferManager()
	inserter := buffer.StartBulkInserter(db, bm)
	defer inserter.Stop()

	for i := 0; i < 65; i++ {
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
			CPU:                  40.0,
			MemUsed:              8192,
			DiskReadBytesPerSec:  1000,
			DiskWriteBytesPerSec: 600,
			NetRxBytesPerSec:     2000,
			NetTxBytesPerSec:     1500,
		})
	}

	// Manually flush
	inserter.Flush()

	// Verify DB
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_5s").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_5s WHERE container_id=?", models.HostMainContainerID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	err = db.QueryRow("SELECT COUNT(*) FROM metrics_5s WHERE container_id=?", models.HostDiskWriteContainerID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	rb := bm.GetOrCreate("agent-bulk", "c-bulk")
	points := rb.GetAll()
	assert.Len(t, points, 60)
	assert.Equal(t, int64(1005), points[0].Timestamp)
	assert.Equal(t, int64(1064), points[len(points)-1].Timestamp)
}
