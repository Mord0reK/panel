package websocket

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type AgentConnection struct {
	UUID    string
	Conn    *websocket.Conn
	SendCh  chan []byte
	CloseCh chan struct{}
}

type AgentHub struct {
	Connections     map[string]*AgentConnection
	Register        chan *AgentConnection
	Unregister      chan *AgentConnection
	PendingCommands sync.Map // map[string]chan []byte (commandID -> response channel)
	mu              sync.RWMutex
}

func NewHub() *AgentHub {
	return &AgentHub{
		Connections: make(map[string]*AgentConnection),
		Register:    make(chan *AgentConnection),
		Unregister:  make(chan *AgentConnection),
	}
}

func (h *AgentHub) Run() {
	for {
		select {
		case agent := <-h.Register:
			h.mu.Lock()
			if old, ok := h.Connections[agent.UUID]; ok {
				close(old.CloseCh)
				delete(h.Connections, agent.UUID)
			}
			h.Connections[agent.UUID] = agent
			h.mu.Unlock()

		case agent := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Connections[agent.UUID]; ok {
				delete(h.Connections, agent.UUID)
				close(agent.CloseCh)
			}
			h.mu.Unlock()
		}
	}
}

func (h *AgentHub) SendToAgent(uuid string, message []byte) error {
	h.mu.RLock()
	agent, ok := h.Connections[uuid]
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
