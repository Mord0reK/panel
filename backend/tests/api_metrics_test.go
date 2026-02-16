package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/api"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsAPI(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := api.NewMetricsHandler(db)
	r := mux.NewRouter()
	r.HandleFunc("/api/metrics/history/servers/{uuid}", handler.HandleServerHistory).Methods("GET")
	r.HandleFunc("/api/metrics/history/servers/{uuid}/containers/{id}", handler.HandleContainerHistory).Methods("GET")

	// 1. Prepare Data
	_, err := db.Exec("INSERT INTO servers (uuid, hostname) VALUES (?, ?)", "s1", "host1")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO containers (agent_uuid, container_id, name) VALUES (?, ?, ?)", "s1", "c1", "cont1")
	require.NoError(t, err)
	ts := time.Now().Unix()

	// Add metric point
	_, err = db.Exec(`INSERT INTO metrics_1s 
		(agent_uuid, container_id, timestamp, cpu_percent, mem_used, mem_percent, disk_used, disk_percent, net_rx_bytes, net_tx_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"s1", "c1", ts, 10.0, 100, 10.0, 200, 20.0, 1000, 2000)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO host_metrics_1s
		(agent_uuid, timestamp, cpu_percent, mem_used, mem_percent, disk_read_bytes_per_sec, disk_write_bytes_per_sec, net_rx_bytes_per_sec, net_tx_bytes_per_sec)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"s1", ts, 50.0, 2048, 20.0, 100, 50, 1000, 1500)
	require.NoError(t, err)

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
		Points []struct {
			Timestamp int64   `json:"timestamp"`
			CPU       float64 `json:"cpu"`
			MemUsed   uint64  `json:"mem_used"`
		} `json:"points"`
		HostPoints []struct {
			Timestamp int64   `json:"timestamp"`
			CPU       float64 `json:"cpu"`
			MemUsed   uint64  `json:"mem_used"`
		} `json:"host_points"`
		Containers []struct {
			ContainerID string `json:"container_id"`
			Points      []struct {
				Timestamp int64   `json:"timestamp"`
				CPU       float64 `json:"cpu"`
			} `json:"points"`
		} `json:"containers"`
	}
	json.Unmarshal(w.Body.Bytes(), &serverResp)
	assert.Len(t, serverResp.Points, 1)
	assert.Equal(t, 50.0, serverResp.Points[0].CPU)
	assert.Len(t, serverResp.HostPoints, 1)
	assert.Equal(t, 50.0, serverResp.HostPoints[0].CPU)
	assert.Len(t, serverResp.Containers, 1)
	assert.Equal(t, "c1", serverResp.Containers[0].ContainerID)
	assert.Len(t, serverResp.Containers[0].Points, 1)
	assert.Equal(t, 10.0, serverResp.Containers[0].Points[0].CPU)
}
