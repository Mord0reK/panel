package websocket

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type AgentConnection struct {
	UUID     string
	Conn     *websocket.Conn
	SendCh   chan []byte
	CloseCh  chan struct{}
	approved atomic.Bool
}

func (a *AgentConnection) SetApproved(approved bool) {
	a.approved.Store(approved)
}

func (a *AgentConnection) IsApproved() bool {
	return a.approved.Load()
}

type AgentHub struct {
	connections     map[string]*AgentConnection
	Register        chan *AgentConnection
	Unregister      chan *AgentConnection
	PendingCommands sync.Map // map[string]chan []byte (commandID -> response channel)
	mu              sync.RWMutex
	stopCh          chan struct{}
}

func NewHub() *AgentHub {
	return &AgentHub{
		connections: make(map[string]*AgentConnection),
		Register:    make(chan *AgentConnection),
		Unregister:  make(chan *AgentConnection),
		stopCh:      make(chan struct{}),
	}
}

// Stop signals the Run loop to exit.
func (h *AgentHub) Stop() {
	close(h.stopCh)
}

// GetConnection returns the active AgentConnection for the given UUID, or nil.
func (h *AgentHub) GetConnection(uuid string) *AgentConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connections[uuid]
}

func (h *AgentHub) Run() {
	for {
		select {
		case <-h.stopCh:
			return

		case agent := <-h.Register:
			h.mu.Lock()
			if old, ok := h.connections[agent.UUID]; ok {
				close(old.CloseCh)
				delete(h.connections, agent.UUID)
			}
			h.connections[agent.UUID] = agent
			h.mu.Unlock()

		case agent := <-h.Unregister:
			h.mu.Lock()
			// Only act if this is still the current connection for this UUID.
			// A reconnecting agent replaces the entry; the old goroutines must
			// not close/delete the new connection.
			if current, ok := h.connections[agent.UUID]; ok && current == agent {
				delete(h.connections, agent.UUID)
				close(agent.CloseCh)
			}
			h.mu.Unlock()
		}
	}
}

func (h *AgentHub) SendToAgent(uuid string, message []byte) error {
	h.mu.RLock()
	agent, ok := h.connections[uuid]
	h.mu.RUnlock()

	if !ok {
		return errors.New("agent not connected")
	}

	select {
	case agent.SendCh <- message:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("timeout sending message to agent")
	}
}

func (h *AgentHub) SetApproved(uuid string, approved bool) bool {
	h.mu.RLock()
	agent, ok := h.connections[uuid]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	agent.SetApproved(approved)
	return true
}

func (h *AgentHub) RequestAgent(agentUUID string, action, target string) (json.RawMessage, error) {
	commandID := uuid.New().String()
	respCh := make(chan []byte, 1)
	h.PendingCommands.Store(commandID, respCh)
	defer h.PendingCommands.Delete(commandID)

	msg := CommandMessage{
		Type:      MsgTypeCommand,
		CommandID: commandID,
		Action:    action,
		Target:    target,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	if err := h.SendToAgent(agentUUID, data); err != nil {
		return nil, err
	}

	select {
	case respData := <-respCh:
		return respData, nil
	case <-time.After(30 * time.Second):
		return nil, errors.New("command timeout")
	}
}
