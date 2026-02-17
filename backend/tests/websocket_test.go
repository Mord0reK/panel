package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/internal/api"
	"backend/internal/buffer"
	"backend/internal/models"
	ws "backend/internal/websocket"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketAgentConnect(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hub := ws.NewHub()
	go hub.Run()

	bm := buffer.NewBufferManager()
	handler := api.NewWebSocketHandler(hub, db, bm)
	s := httptest.NewServer(http.HandlerFunc(handler.HandleAgent))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")

	// Connect
	wsConn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	defer wsConn.Close()

	// Prepare Auth Message
	authMsg := ws.AgentAuthMessage{
		Type: ws.MsgTypeAuth,
		UUID: "agent-123",
		Info: ws.AgentInfo{
			Hostname: "test-host",
			Platform: "linux",
		},
	}
	data, _ := json.Marshal(authMsg)

	// Send Auth
	err = wsConn.WriteMessage(websocket.TextMessage, data)
	require.NoError(t, err)

	// Receive initial auth response
	wsConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, _, err = wsConn.ReadMessage()
	require.NoError(t, err)
	wsConn.SetReadDeadline(time.Time{})

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check if server is in DB
	var server models.Server
	srv, err := server.GetByUUID(db, "agent-123")
	require.NoError(t, err)
	assert.Equal(t, "test-host", srv.Hostname)
	assert.False(t, srv.Approved)

	// Check Hub
	err = hub.SendToAgent("agent-123", []byte("test-command"))
	assert.NoError(t, err)

	// Verify agent received command
	wsConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg, err := wsConn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, "test-command", string(msg))
}

func TestWebSocketAgentAuthTimeout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hub := ws.NewHub()
	go hub.Run()

	bm := buffer.NewBufferManager()
	handler := api.NewWebSocketHandler(hub, db, bm)
	s := httptest.NewServer(http.HandlerFunc(handler.HandleAgent))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")

	// Connect but send nothing
	wsConn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	defer wsConn.Close()

	// Wait for timeout
}

func TestWebSocketMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hub := ws.NewHub()
	go hub.Run()

	bm := buffer.NewBufferManager()
	handler := api.NewWebSocketHandler(hub, db, bm)
	s := httptest.NewServer(http.HandlerFunc(handler.HandleAgent))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	wsConn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	defer wsConn.Close()

	// Auth (First time)
	authMsg := ws.AgentAuthMessage{
		Type: ws.MsgTypeAuth,
		UUID: "agent-metrics",
		Info: ws.AgentInfo{Hostname: "metrics-host"},
	}
	data, _ := json.Marshal(authMsg)
	wsConn.WriteMessage(websocket.TextMessage, data)

	// Read initial auth response (approved=false)
	wsConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg, err := wsConn.ReadMessage()
	require.NoError(t, err)
	var authResp ws.AuthResponseMessage
	err = json.Unmarshal(msg, &authResp)
	require.NoError(t, err)
	assert.False(t, authResp.Approved)
	wsConn.SetReadDeadline(time.Time{})

	time.Sleep(50 * time.Millisecond)

	// Approve server manually
	var server models.Server
	err = server.Approve(db, "agent-metrics")
	require.NoError(t, err)

	// Push approval to connected agent
	hub.SetApproved("agent-metrics", true)
	payload, err := json.Marshal(ws.AuthResponseMessage{Type: ws.MsgTypeAuthResponse, Approved: true})
	require.NoError(t, err)
	err = hub.SendToAgent("agent-metrics", payload)
	require.NoError(t, err)

	wsConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg, err = wsConn.ReadMessage()
	require.NoError(t, err)
	err = json.Unmarshal(msg, &authResp)
	require.NoError(t, err)
	assert.True(t, authResp.Approved)
	wsConn.SetReadDeadline(time.Time{})
	time.Sleep(50 * time.Millisecond)

	// Send Metrics
	metricsMsg := ws.AgentMetricsMessage{
		Type:      ws.MsgTypeMetrics,
		Timestamp: time.Now().Unix(),
		Host: &ws.HostMetrics{
			Timestamp:            time.Now().Unix(),
			CPU:                  12.5,
			MemUsed:              1024,
			MemPercent:           10.0,
			DiskReadBytesPerSec:  100,
			DiskWriteBytesPerSec: 200,
			NetRxBytesPerSec:     300,
			NetTxBytesPerSec:     400,
		},
		Containers: []ws.ContainerMetrics{
			{ContainerID: "c1", CPU: 10.0},
		},
	}
	data, _ = json.Marshal(metricsMsg)
	err = wsConn.WriteMessage(websocket.TextMessage, data)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Check LastSeen updated
	srv, err := server.GetByUUID(db, "agent-metrics")
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), srv.LastSeen, 2*time.Second)

	// Verify buffer has metrics
	buf := bm.GetOrCreate("agent-metrics", "c1")
	points := buf.GetAll()
	assert.NotEmpty(t, points)
	assert.Equal(t, 10.0, points[0].CPU)
	host := bm.GetLatestHostForServer("agent-metrics")
	assert.NotNil(t, host)
	assert.Equal(t, 12.5, host.CPU)
}
