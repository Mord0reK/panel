package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"backend/internal/api"
	"backend/internal/buffer"
	ws "backend/internal/websocket"

	"github.com/gorilla/websocket"
)

func TestStressMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	bm := buffer.NewBufferManager()
	hub := ws.NewHub()
	go hub.Run()

	wsHandler := api.NewWebSocketHandler(hub, db, bm)
	ts := httptest.NewServer(http.HandlerFunc(wsHandler.HandleAgent))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	numAgents := 5
	numContainers := 10
	duration := 5 * time.Second

	var wg sync.WaitGroup
	for a := 0; a < numAgents; a++ {
		wg.Add(1)
		go func(agentID int) {
			defer wg.Done()
			agentUUID := fmt.Sprintf("agent-%d", agentID)
			
			// Pre-approve
			db.Exec("INSERT INTO servers (uuid, hostname, approved) VALUES (?, ?, ?)", agentUUID, agentUUID, true)

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			// Auth
			auth := ws.AgentAuthMessage{Type: ws.MsgTypeAuth, UUID: agentUUID}
			data, _ := json.Marshal(auth)
			conn.WriteMessage(websocket.TextMessage, data)
			time.Sleep(100 * time.Millisecond)

			start := time.Now()
			for time.Since(start) < duration {
				metrics := ws.AgentMetricsMessage{
					Type:      ws.MsgTypeMetrics,
					Timestamp: time.Now().Unix(),
				}
				for c := 0; c < numContainers; c++ {
					metrics.Containers = append(metrics.Containers, ws.ContainerMetrics{
						ContainerID: fmt.Sprintf("c-%d", c),
						CPU:         1.0,
					})
				}
				data, _ := json.Marshal(metrics)
				conn.WriteMessage(websocket.TextMessage, data)
				time.Sleep(1 * time.Second)
			}
		}(a)
	}

	wg.Wait()
	// Success if no panic/hang
}
