package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Client struct {
	url       string
	conn      *websocket.Conn
	writeMu   sync.Mutex
	connected bool
	closed    bool
	closeMu   sync.Mutex
}

func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

func (c *Client) Connect(ctx context.Context) error {
	c.closeMu.Lock()
	if c.connected {
		c.closeMu.Unlock()
		return nil
	}
	c.closeMu.Unlock()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}

	c.closeMu.Lock()
	c.conn = conn
	c.connected = true
	c.closeMu.Unlock()

	return nil
}

func (c *Client) SendMessage(msgType string, data interface{}) error {
	return c.send(data)
}

func (c *Client) SendAuth(uuid string, info AuthInfo) error {
	msg := AuthMessage{
		Type: "auth",
		UUID: uuid,
		Info: info,
	}
	return c.send(msg)
}

func (c *Client) send(data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed || !c.connected || c.conn == nil {
		return fmt.Errorf("not connected")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	err = c.conn.WriteMessage(websocket.TextMessage, dataBytes)
	if err != nil {
		c.connected = false
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (c *Client) WaitForAuthResponse(ctx context.Context) (bool, error) {
	conn := c.conn
	if conn == nil {
		return false, fmt.Errorf("not connected")
	}

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return false, fmt.Errorf("failed to read message: %w", err)
		}

		var resp AuthResponseMessage
		if err := json.Unmarshal(message, &resp); err != nil {
			continue
		}

		if resp.Type == "auth_response" {
			return resp.Approved, nil
		}
	}
}

func (c *Client) WaitForApproval(ctx context.Context) (bool, error) {
	conn := c.conn
	if conn == nil {
		return false, fmt.Errorf("not connected")
	}

	conn.SetReadDeadline(time.Time{})

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return false, fmt.Errorf("failed to read approval message: %w", err)
		}

		var resp AuthResponseMessage
		if err := json.Unmarshal(message, &resp); err != nil {
			continue
		}

		if resp.Type == "auth_response" && resp.Approved {
			return true, nil
		}
	}
}

func (c *Client) Listen(ctx context.Context, handler func(Command) error) error {
	c.closeMu.Lock()
	if c.conn == nil {
		c.closeMu.Unlock()
		return fmt.Errorf("no connection")
	}
	conn := c.conn
	c.closeMu.Unlock()

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// Ping goroutine
	pingCtx, cancelPing := context.WithCancel(ctx)
	defer cancelPing()

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-pingCtx.Done():
				return
			case <-ticker.C:
				c.writeMu.Lock()
				err := conn.WriteMessage(websocket.PingMessage, nil)
				c.writeMu.Unlock()
				if err != nil {
					log.Printf("Ping error: %v", err)
					cancelPing()
					return
				}
			}
		}
	}()

	// Main read loop
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		c.closeMu.Lock()
		isClosed := c.closed
		c.closeMu.Unlock()

		if isClosed {
			return nil
		}

		conn.SetReadDeadline(time.Now().Add(pongWait))
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}

			c.closeMu.Lock()
			c.connected = false
			c.closeMu.Unlock()

			return fmt.Errorf("connection lost: %w", err)
		}

		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(message, &envelope); err != nil {
			log.Printf("Failed to unmarshal message envelope: %v", err)
			continue
		}

		if envelope.Type != "command" {
			continue
		}

		var cmd Command
		if err := json.Unmarshal(message, &cmd); err != nil {
			log.Printf("Failed to unmarshal command: %v", err)
			continue
		}

		if err := handler(cmd); err != nil {
			log.Printf("Command handler error: %v", err)
		}
	}
}

func (c *Client) Close() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return
	}

	c.closed = true

	if c.conn != nil {
		c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
	}
}

func (c *Client) IsConnected() bool {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	return c.connected && c.conn != nil
}

func (c *Client) Reconnect(ctx context.Context) error {
	c.closeMu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.closeMu.Unlock()

	if err := c.Connect(ctx); err != nil {
		return err
	}

	log.Printf("WebSocket reconnected to %s", c.url)
	return nil
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Command struct {
	Type      string          `json:"type"`
	CommandID string          `json:"command_id"`
	Action    string          `json:"action"`
	Target    string          `json:"target,omitempty"`
	Args      json.RawMessage `json:"args,omitempty"`
}

type MetricsMessage struct {
	Type       string      `json:"type"`
	Timestamp  int64       `json:"timestamp"`
	Host       interface{} `json:"host,omitempty"`
	Containers interface{} `json:"containers,omitempty"`
}

type AuthMessage struct {
	Type string   `json:"type"`
	UUID string   `json:"uuid"`
	Info AuthInfo `json:"info"`
}

type AuthInfo struct {
	Hostname     string `json:"hostname"`
	CPUModel     string `json:"cpu_model"`
	CPUCores     int    `json:"cpu_cores"`
	MemoryTotal  uint64 `json:"memory_total"`
	Platform     string `json:"platform"`
	Kernel       string `json:"kernel"`
	Architecture string `json:"architecture"`
}

type AuthResponseMessage struct {
	Type     string `json:"type"`
	Approved bool   `json:"approved"`
}
