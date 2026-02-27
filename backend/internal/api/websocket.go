package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend/internal/buffer"
	"backend/internal/models"
	ws "backend/internal/websocket"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for agents
	},
}

type WebSocketHandler struct {
	hub           *ws.AgentHub
	db            *sql.DB
	bufferManager *buffer.BufferManager
}

func NewWebSocketHandler(hub *ws.AgentHub, db *sql.DB, bufferManager *buffer.BufferManager) *WebSocketHandler {
	return &WebSocketHandler{hub: hub, db: db, bufferManager: bufferManager}
}

func (h *WebSocketHandler) HandleAgent(w http.ResponseWriter, r *http.Request) {
	log.Println("WebSocket: new connection request from", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	log.Println("WebSocket: connection upgraded, waiting for auth message")

	// 1. Wait for Auth
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Println("WebSocket: failed to read auth message:", err)
		conn.Close()
		return
	}

	log.Printf("WebSocket: received message: %s", string(message))

	msg, err := ws.ParseMessage(message)
	if err != nil {
		log.Println("WebSocket: failed to parse message:", err)
		conn.Close()
		return
	}

	authMsg, ok := msg.(ws.AgentAuthMessage)
	if !ok {
		conn.Close() // Expected auth message
		return
	}

	if authMsg.UUID == "" {
		conn.Close()
		return
	}

	// 2. Register/Update Server in DB
	server := models.Server{
		UUID:         authMsg.UUID,
		Hostname:     authMsg.Info.Hostname,
		CPUModel:     authMsg.Info.CPUModel,
		CPUCores:     authMsg.Info.CPUCores,
		MemoryTotal:  authMsg.Info.MemoryTotal,
		Platform:     authMsg.Info.Platform,
		Kernel:       authMsg.Info.Kernel,
		Architecture: authMsg.Info.Architecture,
	}

	if err := server.Upsert(h.db); err != nil {
		log.Println("failed to upsert server:", err)
		conn.Close()
		return
	}
	log.Printf("WebSocket: server %s upserted, approved=%v", authMsg.UUID, server.Approved)

	// 3. Register to Hub
	agentConn := &ws.AgentConnection{
		UUID:    authMsg.UUID,
		Conn:    conn,
		SendCh:  make(chan []byte, 256),
		CloseCh: make(chan struct{}),
	}
	agentConn.SetApproved(server.Approved)

	// Reject connections from servers that have been explicitly rejected by the admin.
	if server.Status == "rejected" {
		log.Printf("WebSocket: refusing connection from rejected server %s", authMsg.UUID)
		respBytes, _ := json.Marshal(ws.AuthResponseMessage{Type: ws.MsgTypeAuthResponse, Approved: false})
		conn.WriteMessage(websocket.TextMessage, respBytes)
		conn.Close()
		return
	}

	h.hub.Register <- agentConn
	log.Printf("WebSocket: agent %s registered to hub", authMsg.UUID)

	// 5. Send auth response with approval status
	authResp := ws.AuthResponseMessage{
		Type:     ws.MsgTypeAuthResponse,
		Approved: server.Approved,
	}
	respBytes, err := json.Marshal(authResp)
	if err != nil {
		log.Printf("WebSocket: failed to marshal auth response: %v", err)
	} else {
		select {
		case agentConn.SendCh <- respBytes:
			log.Printf("WebSocket: sent auth response approved=%v to %s", server.Approved, authMsg.UUID)
		default:
			log.Printf("WebSocket: failed to send auth response - channel full")
		}
	}

	// 6. Start Loops
	go h.writePump(agentConn)
	h.readPump(agentConn)
}

func (h *WebSocketHandler) readPump(agent *ws.AgentConnection) {
	defer func() {
		h.hub.Unregister <- agent
		agent.Conn.Close()
		// Free all in-memory ring-buffers for this agent so stale allocations
		// don't accumulate when containers are rotated or servers go offline.
		h.bufferManager.RemoveAgentBuffers(agent.UUID)
		log.Printf("WebSocket: freed buffer memory for agent %s", agent.UUID)
	}()

	agent.Conn.SetReadLimit(512 * 1024) // 512KB
	agent.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	agent.Conn.SetPongHandler(func(string) error {
		agent.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := agent.Conn.ReadMessage()
		if err != nil {
			break
		}

		msg, err := ws.ParseMessage(message)
		if err != nil {
			continue
		}

		switch m := msg.(type) {
		case ws.AgentMetricsMessage:
			if agent.IsApproved() {
				if m.Host != nil {
					ts := m.Timestamp
					if m.Host.Timestamp > 0 {
						ts = m.Host.Timestamp
					}
					h.bufferManager.AddHostMetric(agent.UUID, buffer.HostMetricPoint{
						Timestamp:            ts,
						CPU:                  m.Host.CPU,
						MemUsed:              m.Host.MemUsed,
						MemPercent:           m.Host.MemPercent,
						MemoryTotal:          m.Host.MemoryTotal,
						DiskReadBytesPerSec:  m.Host.DiskReadBytesPerSec,
						DiskWriteBytesPerSec: m.Host.DiskWriteBytesPerSec,
						NetRxBytesPerSec:     m.Host.NetRxBytesPerSec,
						NetTxBytesPerSec:     m.Host.NetTxBytesPerSec,
						DiskUsedPercent:      m.Host.DiskUsedPercent,
					})
				} else if m.System != nil {
					// Backward compatibility for older agents sending "system" only.
					h.bufferManager.AddHostMetric(agent.UUID, buffer.HostMetricPoint{
						Timestamp:  m.Timestamp,
						CPU:        m.System.CPU.Percent,
						MemUsed:    m.System.Memory.Used,
						MemPercent: m.System.Memory.Percent,
					})
				}

				// Process metrics
				activeIDs := make([]string, 0, len(m.Containers))
				for _, c := range m.Containers {
					activeIDs = append(activeIDs, c.ContainerID)

					// Upsert container info (running or stopped)
					cont := models.Container{
						AgentUUID:   agent.UUID,
						ContainerID: c.ContainerID,
						Name:        c.Name,
						Image:       c.Image,
						Project:     c.Project,
						Service:     c.Service,
						State:       c.State,
					}
					cont.Upsert(h.db)

					// Buffer metrics only for running containers (or legacy agents that don't send state).
					// Stopped/exited containers carry zeroed stats — no point buffering them.
					if c.State == "running" || c.State == "" {
						point := buffer.MetricPoint{
							Timestamp:   m.Timestamp,
							CPU:         c.CPU,
							MemUsed:     c.MemUsed,
							MemPercent:  c.MemPercent,
							DiskUsed:    c.DiskUsed,
							DiskPercent: c.DiskPercent,
							NetRx:       c.NetRx,
							NetTx:       c.NetTx,
						}
						h.bufferManager.AddMetric(agent.UUID, c.ContainerID, point)
					}
				}

				// Remove containers that the agent no longer knows about (docker rm'd)
				if err := models.DeleteNotInList(h.db, agent.UUID, activeIDs); err != nil {
					log.Printf("WebSocket: failed to sync container list for agent %s: %v", agent.UUID, err)
				}

				var s models.Server
				s.UpdateLastSeen(h.db, agent.UUID)
			} else {
				var s models.Server
				s.UpdateLastSeen(h.db, agent.UUID)
			}
		case ws.CommandResponse:
			if ch, ok := h.hub.PendingCommands.Load(m.CommandID); ok {
				respCh := ch.(chan []byte)
				respCh <- m.Payload
			}
		}
	}
}

func (h *WebSocketHandler) writePump(agent *ws.AgentConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		agent.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-agent.SendCh:
			agent.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				agent.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := agent.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			agent.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := agent.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-agent.CloseCh:
			return
		}
	}
}
