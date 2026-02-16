package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/internal/api"
	"backend/internal/buffer"
	"backend/internal/config"
	ws "backend/internal/websocket"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := &config.Config{JWTSecret: "super-secret"}
	bm := buffer.NewBufferManager()
	hub := ws.NewHub()
	go hub.Run()

	authHandler := api.NewAuthHandler(db, cfg)
	wsHandler := api.NewWebSocketHandler(hub, db, bm)
	serversHandler := api.NewServersHandler(db)
	commandsHandler := api.NewCommandsHandler(hub)
	metricsHandler := api.NewMetricsHandler(db)

	r := mux.NewRouter()
	r.HandleFunc("/api/setup", authHandler.HandleSetup).Methods("POST")
	r.HandleFunc("/api/login", authHandler.HandleLogin).Methods("POST")
	r.HandleFunc("/api/servers/{uuid}/approve", serversHandler.HandleApprove).Methods("PUT")
	r.HandleFunc("/api/servers/{uuid}/command", commandsHandler.HandleCommand).Methods("POST")
	r.HandleFunc("/api/metrics/history/servers/{uuid}", metricsHandler.HandleServerHistory).Methods("GET")
	r.HandleFunc("/ws/agent", wsHandler.HandleAgent)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// 1. Setup User
	setupPayload := map[string]string{"username": "admin", "password": "password123"}
	body, _ := json.Marshal(setupPayload)
	resp, _ := http.Post(ts.URL+"/api/setup", "application/json", bytes.NewBuffer(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var tokenResp map[string]string
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	token := tokenResp["token"]

	// 2. Connect Agent
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/agent"
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	authMsg := ws.AgentAuthMessage{
		Type: ws.MsgTypeAuth,
		UUID: "agent-full",
		Info: ws.AgentInfo{Hostname: "full-host"},
	}
	data, _ := json.Marshal(authMsg)
	wsConn.WriteMessage(websocket.TextMessage, data)
	time.Sleep(100 * time.Millisecond)

	// 3. Approve Server
	req, _ := http.NewRequest("PUT", ts.URL+"/api/servers/agent-full/approve", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ = http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 4. Send Metrics (requires reconnect because of my implementation)
	wsConn.Close()
	wsConn, _, _ = websocket.DefaultDialer.Dial(wsURL, nil)
	wsConn.WriteMessage(websocket.TextMessage, data)
	time.Sleep(50 * time.Millisecond)

	metricsMsg := ws.AgentMetricsMessage{
		Type:      ws.MsgTypeMetrics,
		Timestamp: time.Now().Unix(),
		Host: &ws.HostMetrics{
			Timestamp:            time.Now().Unix(),
			CPU:                  30.0,
			MemUsed:              4096,
			MemPercent:           20.0,
			DiskReadBytesPerSec:  100,
			DiskWriteBytesPerSec: 50,
			NetRxBytesPerSec:     1000,
			NetTxBytesPerSec:     800,
		},
		Containers: []ws.ContainerMetrics{
			{ContainerID: "c1", CPU: 25.0},
		},
	}
	data, _ = json.Marshal(metricsMsg)
	wsConn.WriteMessage(websocket.TextMessage, data)
	time.Sleep(50 * time.Millisecond)

	// 5. Check RAM Buffer
	latest := bm.GetLatestForServer("agent-full")
	assert.NotNil(t, latest["c1"])
	assert.Equal(t, 25.0, latest["c1"].CPU)
	latestHost := bm.GetLatestHostForServer("agent-full")
	assert.NotNil(t, latestHost)
	assert.Equal(t, 30.0, latestHost.CPU)

	// 6. Bulk Insert
	inserter := buffer.StartBulkInserter(db, bm)
	// Add 9 more points to reach 10 threshold
	baseTs := time.Now().Unix() - 9
	for i := 1; i < 10; i++ {
		tsPoint := baseTs + int64(i)
		bm.AddMetric("agent-full", "c1", buffer.MetricPoint{Timestamp: tsPoint, CPU: 20.0})
		bm.AddHostMetric("agent-full", buffer.HostMetricPoint{
			Timestamp:            tsPoint,
			CPU:                  30.0,
			MemUsed:              4096,
			DiskReadBytesPerSec:  100,
			DiskWriteBytesPerSec: 50,
			NetRxBytesPerSec:     1000,
			NetTxBytesPerSec:     800,
		})
	}
	inserter.Flush()
	inserter.Stop()

	// 7. Query History
	req, _ = http.NewRequest("GET", ts.URL+"/api/metrics/history/servers/agent-full?range=1m", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ = http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var histResp struct {
		Points     []interface{} `json:"points"`
		HostPoints []interface{} `json:"host_points"`
		Containers []interface{} `json:"containers"`
	}
	json.NewDecoder(resp.Body).Decode(&histResp)
	assert.NotEmpty(t, histResp.Points)
	assert.NotEmpty(t, histResp.HostPoints)
	assert.NotEmpty(t, histResp.Containers)

	// 8. Command
	done := make(chan struct{})
	go func() {
		payload := map[string]string{"action": "stop", "target": "c1"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", ts.URL+"/api/servers/agent-full/command", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		close(done)
	}()

	// Agent receive cmd
	_, msg, _ := wsConn.ReadMessage()
	var cmd ws.CommandMessage
	json.Unmarshal(msg, &cmd)
	respMsg := ws.CommandResponse{
		Type:      ws.MsgTypeResponse,
		CommandID: cmd.CommandID,
		Payload:   json.RawMessage(`{"status": "stopped"}`),
	}
	data, _ = json.Marshal(respMsg)
	wsConn.WriteMessage(websocket.TextMessage, data)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Command timeout")
	}
}
