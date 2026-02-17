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
	ws "backend/internal/websocket"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsAPI(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hub := ws.NewHub()
	go hub.Run()

	// 1. Mock Agent Connection
	wsHandler := api.NewWebSocketHandler(hub, db, nil)
	s := httptest.NewServer(http.HandlerFunc(wsHandler.HandleAgent))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	wsConn, _, _ := websocket.DefaultDialer.Dial(u, nil)

	// Auth
	authMsg := ws.AgentAuthMessage{Type: ws.MsgTypeAuth, UUID: "agent-cmd"}
	data, _ := json.Marshal(authMsg)
	wsConn.WriteMessage(websocket.TextMessage, data)
	wsConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, _, err := wsConn.ReadMessage()
	require.NoError(t, err)
	wsConn.SetReadDeadline(time.Time{})
	time.Sleep(50 * time.Millisecond)

	// 2. Command API Setup
	cmdHandler := api.NewCommandsHandler(hub)
	r := mux.NewRouter()
	r.HandleFunc("/api/servers/{uuid}/command", cmdHandler.HandleCommand).Methods("POST")

	// 3. Request Command (in goroutine because it blocks)
	var respCode int
	var respBody []byte
	done := make(chan struct{})

	go func() {
		payload := map[string]string{"action": "restart", "target": "nginx"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/servers/agent-cmd/command", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		respCode = w.Code
		respBody = w.Body.Bytes()
		close(done)
	}()

	// 4. Mock Agent receiving and responding
	wsConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg, err := wsConn.ReadMessage()
	require.NoError(t, err)

	var cmd ws.CommandMessage
	json.Unmarshal(msg, &cmd)
	assert.Equal(t, "restart", cmd.Action)

	// Respond
	respMsg := ws.CommandResponse{
		Type:      ws.MsgTypeResponse,
		CommandID: cmd.CommandID,
		Payload:   json.RawMessage(`{"success": true}`),
	}
	data, _ = json.Marshal(respMsg)
	wsConn.WriteMessage(websocket.TextMessage, data)

	// 5. Verify API Response
	select {
	case <-done:
		assert.Equal(t, http.StatusOK, respCode)
		assert.Contains(t, string(respBody), "true")
	case <-time.After(2 * time.Second):
		t.Fatal("API timeout")
	}
}
