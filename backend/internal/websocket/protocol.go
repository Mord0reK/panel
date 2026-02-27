package websocket

import (
	"encoding/json"
	"errors"
)

// Message types
const (
	MsgTypeAuth         = "auth"
	MsgTypeAuthResponse = "auth_response"
	MsgTypeMetrics      = "metrics"
	MsgTypeCommand      = "command"
	MsgTypeResponse     = "response"
)

type BaseMessage struct {
	Type string `json:"type"`
}

type AgentInfo struct {
	Hostname     string `json:"hostname"`
	CPUModel     string `json:"cpu_model"`
	CPUCores     int    `json:"cpu_cores"`
	CPUThreads   int    `json:"cpu_threads"`
	MemoryTotal  uint64 `json:"memory_total"`
	Platform     string `json:"platform"`
	Kernel       string `json:"kernel"`
	Architecture string `json:"architecture"`
}

type AgentAuthMessage struct {
	Type string    `json:"type"`
	UUID string    `json:"uuid"`
	Info AgentInfo `json:"info"`
}

type AuthResponseMessage struct {
	Type     string `json:"type"`
	Approved bool   `json:"approved"`
}

type ContainerMetrics struct {
	ContainerID string  `json:"container_id"`
	Name        string  `json:"name"`
	Image       string  `json:"image"`
	Project     string  `json:"project"`
	Service     string  `json:"service"`
	State       string  `json:"state"` // running, stopped, etc.
	Timestamp   int64   `json:"timestamp"`
	CPU         float64 `json:"cpu_percent"`
	MemUsed     uint64  `json:"mem_used"`
	MemPercent  float64 `json:"mem_percent"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskPercent float64 `json:"disk_percent"`
	NetRx       uint64  `json:"net_rx_bytes"`
	NetTx       uint64  `json:"net_tx_bytes"`
}

type HostMetrics struct {
	Timestamp int64   `json:"timestamp"`
	CPU       float64 `json:"cpu_percent"`

	MemUsed     uint64  `json:"mem_used"`
	MemPercent  float64 `json:"mem_percent"`
	MemoryTotal uint64  `json:"memory_total"`

	DiskReadBytesPerSec  uint64  `json:"disk_read_bytes_per_sec"`
	DiskWriteBytesPerSec uint64  `json:"disk_write_bytes_per_sec"`
	NetRxBytesPerSec     uint64  `json:"net_rx_bytes_per_sec"`
	NetTxBytesPerSec     uint64  `json:"net_tx_bytes_per_sec"`
	DiskUsedPercent      float64 `json:"disk_used_percent"`
}

type LegacySystemMetrics struct {
	CPU struct {
		Percent float64 `json:"percent"`
	} `json:"cpu"`
	Memory struct {
		Used    uint64  `json:"used"`
		Percent float64 `json:"percent"`
	} `json:"memory"`
	Network []struct {
		BytesSent uint64 `json:"bytes_sent"`
		BytesRecv uint64 `json:"bytes_recv"`
	} `json:"network"`
}

type AgentMetricsMessage struct {
	Type       string               `json:"type"`
	Timestamp  int64                `json:"timestamp"`
	Host       *HostMetrics         `json:"host,omitempty"`
	System     *LegacySystemMetrics `json:"system,omitempty"`
	Containers []ContainerMetrics   `json:"containers"`
}

type CommandMessage struct {
	Type      string `json:"type"`
	CommandID string `json:"command_id"`
	Action    string `json:"action"`
	Target    string `json:"target"` // container_id or name
}

type CommandResponse struct {
	Type      string          `json:"type"`
	CommandID string          `json:"command_id"`
	Payload   json.RawMessage `json:"payload"`
}

// ParseMessage attempts to decode the raw JSON into a specific message struct based on "type" field
func ParseMessage(data []byte) (interface{}, error) {
	var base BaseMessage
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	switch base.Type {
	case MsgTypeAuth:
		var msg AgentAuthMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case MsgTypeAuthResponse:
		var msg AuthResponseMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case MsgTypeMetrics:
		var msg AgentMetricsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case MsgTypeCommand:
		var msg CommandMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case MsgTypeResponse:
		var msg CommandResponse
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	default:
		return nil, errors.New("unknown message type")
	}
}
