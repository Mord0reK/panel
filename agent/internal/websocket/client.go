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
	reconnectInitialDelay = 1 * time.Second
	reconnectMaxDelay     = 30 * time.Second
	pongWait              = 60 * time.Second
	pingPeriod            = (pongWait * 9) / 10
)

type Client struct {
	url         string
	conn        *websocket.Conn
	writeMu     sync.Mutex
	connected   bool
	connectedCh chan bool
	closed      bool
	closeMu     sync.Mutex
}

func NewClient(url string) *Client {
	return &Client{
		url:         url,
		connectedCh: make(chan bool, 1),
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

	select {
	case c.connectedCh <- true:
	default:
	}

	return nil
}

func (c *Client) reconnect(ctx context.Context) {
	delay := reconnectInitialDelay
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		log.Printf("Attempting to reconnect to %s...", c.url)

		err := c.Connect(ctx)
		if err != nil {
			log.Printf("Reconnect failed: %v", err)
			delay = delay * 2
			if delay > reconnectMaxDelay {
				delay = reconnectMaxDelay
			}
			continue
		}

		log.Printf("WebSocket reconnected to %s", c.url)
		return
	}
}

func (c *Client) SendMessage(msgType string, data interface{}) error {
	c.closeMu.Lock()
	if c.closed || !c.connected || c.conn == nil {
		c.closeMu.Unlock()
		return fmt.Errorf("not connected")
	}
	c.closeMu.Unlock()

	msg := Message{
		Type: msgType,
		Data: data,
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.closeMu.Lock()
	conn := c.conn
	c.closeMu.Unlock()

	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	dataBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, dataBytes)
	if err != nil {
		c.closeMu.Lock()
		c.connected = false
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.closeMu.Unlock()
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
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

	readCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	for {
		select {
		case <-readCtx.Done():
			return nil
		case <-pingTicker.C:
			c.writeMu.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			c.writeMu.Unlock()
			if err != nil {
				log.Printf("Ping error: %v", err)
				return fmt.Errorf("ping failed: %w", err)
			}
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
	Type   string          `json:"type"`
	Action string          `json:"action"`
	Target string          `json:"target,omitempty"`
	Args   json.RawMessage `json:"args,omitempty"`
}

type MetricsMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	System    interface{} `json:"system,omitempty"`
	Docker    interface{} `json:"docker,omitempty"`
}
