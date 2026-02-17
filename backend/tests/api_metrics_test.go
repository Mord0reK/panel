package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/api"
	"backend/internal/buffer"
	"backend/internal/models"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsAPI(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bm := buffer.NewBufferManager()
	handler := api.NewMetricsHandler(db, bm)
	r := mux.NewRouter()
	r.HandleFunc("/api/metrics/history/servers/{uuid}", handler.HandleServerHistory).Methods("GET")
	r.HandleFunc("/api/metrics/history/servers/{uuid}/containers/{id}", handler.HandleContainerHistory).Methods("GET")

	// 1. Prepare Data
	_, err := db.Exec("INSERT INTO servers (uuid, hostname) VALUES (?, ?)", "s1", "host1")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO containers (agent_uuid, container_id, name) VALUES (?, ?, ?)", "s1", "c1", "cont1")
	require.NoError(t, err)
	ts := time.Now().Unix()

	bm.AddMetric("s1", "c1", buffer.MetricPoint{
		Timestamp: ts,
		CPU:       10.0,
		MemUsed:   100,
		DiskUsed:  200,
		NetRx:     1000,
		NetTx:     2000,
	})
	bm.AddHostMetric("s1", buffer.HostMetricPoint{
		Timestamp:            ts,
		CPU:                  22.0,
		MemUsed:              2048,
		MemPercent:           25.0,
		DiskReadBytesPerSec:  300,
		DiskWriteBytesPerSec: 150,
		NetRxBytesPerSec:     1200,
		NetTxBytesPerSec:     900,
	})

	// 2. Test Container History
	req := httptest.NewRequest("GET", "/api/metrics/history/servers/s1/containers/c1?range=1m", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Points []struct {
			Timestamp  int64   `json:"timestamp"`
			CPU        float64 `json:"cpu"`
			MemUsed    uint64  `json:"mem_used"`
			DiskUsed   uint64  `json:"disk_used"`
			NetRxBytes uint64  `json:"net_rx_bytes"`
			NetTxBytes uint64  `json:"net_tx_bytes"`
		} `json:"points"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp.Points, 1)
	assert.Equal(t, 10.0, resp.Points[0].CPU)

	// 3. Test Server History (Host + Containers)
	req = httptest.NewRequest("GET", "/api/metrics/history/servers/s1?range=1m", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var serverResp struct {
		Host struct {
			Points []struct {
				Timestamp int64   `json:"timestamp"`
				CPU       float64 `json:"cpu"`
			} `json:"points"`
		} `json:"host"`
		Containers []struct {
			ContainerID string `json:"container_id"`
			Points      []struct {
				Timestamp int64   `json:"timestamp"`
				CPU       float64 `json:"cpu"`
			} `json:"points"`
		} `json:"containers"`
	}
	json.Unmarshal(w.Body.Bytes(), &serverResp)
	assert.Len(t, serverResp.Host.Points, 1)
	assert.Equal(t, 22.0, serverResp.Host.Points[0].CPU)
	assert.Len(t, serverResp.Containers, 1)
	assert.Equal(t, "c1", serverResp.Containers[0].ContainerID)
	assert.Len(t, serverResp.Containers[0].Points, 1)
	assert.Equal(t, 10.0, serverResp.Containers[0].Points[0].CPU)
}

func TestServerHistoryRangeOver1mIncludesHostFromDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bm := buffer.NewBufferManager()
	handler := api.NewMetricsHandler(db, bm)
	r := mux.NewRouter()
	r.HandleFunc("/api/metrics/history/servers/{uuid}", handler.HandleServerHistory).Methods("GET")

	_, err := db.Exec("INSERT INTO servers (uuid, hostname) VALUES (?, ?)", "s2", "host2")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO containers (agent_uuid, container_id, name) VALUES (?, ?, ?)", "s2", "c2", "cont2")
	require.NoError(t, err)

	now := time.Now().Unix()
	_, err = db.Exec(`INSERT INTO metrics_1m
		(agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"s2", models.HostMainContainerID, now-300,
		35.0, 30.0, 40.0,
		4096.0, 4000.0, 4200.0,
		200.0,
		1000.0, 900.0, 1100.0,
		800.0, 700.0, 900.0,
	)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO metrics_1m
		(agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"s2", models.HostDiskWriteContainerID, now-300,
		0.0, 0.0, 0.0,
		0.0, 0.0, 0.0,
		0.0,
		120.0, 100.0, 140.0,
		0.0, 0.0, 0.0,
	)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO metrics_1m
		(agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"s2", "c2", now-300,
		15.0, 10.0, 20.0,
		1024.0, 1000.0, 1100.0,
		300.0,
		500.0, 450.0, 550.0,
		450.0, 400.0, 500.0,
	)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/metrics/history/servers/s2?range=1h", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Host struct {
			Points []struct {
				CPUAvg float64 `json:"cpu_avg"`
			} `json:"points"`
		} `json:"host"`
		Containers []struct {
			ContainerID string `json:"container_id"`
			Points      []struct {
				CPUAvg float64 `json:"cpu_avg"`
			} `json:"points"`
		} `json:"containers"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Len(t, resp.Host.Points, 1)
	assert.Equal(t, 35.0, resp.Host.Points[0].CPUAvg)
	assert.Len(t, resp.Containers, 1)
	assert.Equal(t, "c2", resp.Containers[0].ContainerID)
	assert.Len(t, resp.Containers[0].Points, 1)
	assert.Equal(t, 15.0, resp.Containers[0].Points[0].CPUAvg)
}
