package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"backend/internal/buffer"
	"backend/internal/models"

	"github.com/gorilla/mux"
)

type containerDBCache struct {
	mu       sync.RWMutex
	byAgent  map[string]containerCacheEntry
	cacheTTL time.Duration
}

type containerCacheEntry struct {
	timestamp  time.Time
	containers []models.Container
}

func newContainerDBCache() *containerDBCache {
	return &containerDBCache{
		byAgent:  make(map[string]containerCacheEntry),
		cacheTTL: 10 * time.Second, // Refresh every 10s instead of every SSE tick
	}
}

func (c *containerDBCache) get(db *sql.DB, agentUUID string) ([]models.Container, error) {
	c.mu.RLock()
	entry, ok := c.byAgent[agentUUID]
	c.mu.RUnlock()

	if ok && time.Since(entry.timestamp) < c.cacheTTL {
		return entry.containers, nil
	}

	// Cache miss or expired - fetch from DB
	var containerModel models.Container
	containers, err := containerModel.GetByAgent(db, agentUUID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.byAgent[agentUUID] = containerCacheEntry{
		timestamp:  time.Now(),
		containers: containers,
	}
	c.mu.Unlock()

	return containers, nil
}

// serverListCache caches the full server list from SQLite.
// Servers change only when a new agent authenticates — no need to query every SSE tick.
type serverListCache struct {
	mu        sync.RWMutex
	servers   []models.Server
	fetchedAt time.Time
	ttl       time.Duration
}

func newServerListCache() *serverListCache {
	return &serverListCache{ttl: 30 * time.Second}
}

func (s *serverListCache) get(db *sql.DB) ([]models.Server, error) {
	s.mu.RLock()
	if !s.fetchedAt.IsZero() && time.Since(s.fetchedAt) < s.ttl {
		result := s.servers
		s.mu.RUnlock()
		return result, nil
	}
	s.mu.RUnlock()

	var serverModel models.Server
	servers, err := serverModel.GetAll(db)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.servers = servers
	s.fetchedAt = time.Now()
	s.mu.Unlock()

	return servers, nil
}

type SSEHandler struct {
	db               *sql.DB
	bufferManager    *buffer.BufferManager
	corsOrigin       string
	containerDBCache *containerDBCache
	serverListCache  *serverListCache
}

func NewSSEHandler(db *sql.DB, bufferManager *buffer.BufferManager, corsOrigin string) *SSEHandler {
	return &SSEHandler{
		db:               db,
		bufferManager:    bufferManager,
		corsOrigin:       corsOrigin,
		containerDBCache: newContainerDBCache(),
		serverListCache:  newServerListCache(),
	}
}

func (h *SSEHandler) HandleLiveAll(w http.ResponseWriter, r *http.Request) {
	// 1. Setup SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", h.corsOrigin)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			// 2. Fetch all approved servers (cached, TTL 30s)
			servers, err := h.serverListCache.get(h.db)
			if err != nil {
				continue
			}

			type serverLive struct {
				UUID        string  `json:"uuid"`
				Hostname    string  `json:"hostname"`
				CPU         float64 `json:"cpu"`
				Memory      uint64  `json:"memory"`
				MemPercent  float64 `json:"mem_percent"`
				MemoryTotal uint64  `json:"memory_total"`

				DiskUsed             uint64  `json:"disk_used"`
				DiskUsedPercent      float64 `json:"disk_used_percent"`
				DiskReadBytesPerSec  uint64  `json:"disk_read_bytes_per_sec"`
				DiskWriteBytesPerSec uint64  `json:"disk_write_bytes_per_sec"`
				NetRxBytesPerSec     uint64  `json:"net_rx_bytes_per_sec"`
				NetTxBytesPerSec     uint64  `json:"net_tx_bytes_per_sec"`
			}

			var liveData []serverLive
			for _, s := range servers {
				if !s.Approved {
					continue
				}

				host := h.bufferManager.GetLatestHostForServer(s.UUID)
				if host == nil {
					continue
				}

				liveData = append(liveData, serverLive{
					UUID:        s.UUID,
					Hostname:    s.Hostname,
					CPU:         host.CPU,
					Memory:      host.MemUsed,
					MemPercent:  host.MemPercent,
					MemoryTotal: host.MemoryTotal,

					DiskUsed:             host.DiskUsed,
					DiskUsedPercent:      host.DiskUsedPercent,
					DiskReadBytesPerSec:  host.DiskReadBytesPerSec,
					DiskWriteBytesPerSec: host.DiskWriteBytesPerSec,
					NetRxBytesPerSec:     host.NetRxBytesPerSec,
					NetTxBytesPerSec:     host.NetTxBytesPerSec,
				})
			}

			// 3. Send SSE
			data, _ := json.Marshal(map[string]interface{}{"servers": liveData})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (h *SSEHandler) HandleLiveServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", h.corsOrigin)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			host := h.bufferManager.GetLatestHostForServer(uuid)
			if host == nil {
				continue
			}
			containersMap := h.bufferManager.GetLatestForServerAtTimestamp(uuid)

			// Fetch container states from DB with caching (10s TTL).
			// State/health/status change rarely, so we don't need to query DB every SSE tick.
			dbContainers, _ := h.containerDBCache.get(h.db, uuid)
			type containerDBInfo struct {
				State   string
				Health  string
				Status  string
				Project string
			}
			infoByID := make(map[string]containerDBInfo, len(dbContainers))
			for _, c := range dbContainers {
				infoByID[c.ContainerID] = containerDBInfo{
					State:   c.State,
					Health:  c.Health,
					Status:  c.Status,
					Project: c.Project,
				}
			}

			type containerLive struct {
				ContainerID string  `json:"ContainerID"`
				Timestamp   int64   `json:"Timestamp"`
				CPU         float64 `json:"CPU"`
				MemUsed     uint64  `json:"MemUsed"`
				MemPercent  float64 `json:"MemPercent"`
				DiskUsed    uint64  `json:"DiskUsed"`
				DiskPercent float64 `json:"DiskPercent"`
				NetRx       uint64  `json:"NetRx"`
				NetTx       uint64  `json:"NetTx"`
				State       string  `json:"State"`
				Health      string  `json:"Health"`
				Status      string  `json:"Status"`
				Project     string  `json:"Project"`
			}

			containerSlice := make([]containerLive, 0, len(containersMap))
			for id, p := range containersMap {
				if id == models.HostMainContainerID {
					continue
				}
				info := infoByID[id]
				containerSlice = append(containerSlice, containerLive{
					ContainerID: id,
					Timestamp:   p.Timestamp,
					CPU:         p.CPU,
					MemUsed:     p.MemUsed,
					MemPercent:  p.MemPercent,
					DiskUsed:    p.DiskUsed,
					DiskPercent: p.DiskPercent,
					NetRx:       p.NetRx,
					NetTx:       p.NetTx,
					State:       info.State,
					Health:      info.Health,
					Status:      info.Status,
					Project:     info.Project,
				})
			}

			type hostLive struct {
				Timestamp            int64   `json:"Timestamp"`
				CPU                  float64 `json:"CPU"`
				MemUsed              uint64  `json:"MemUsed"`
				MemPercent           float64 `json:"MemPercent"`
				DiskReadBytesPerSec  uint64  `json:"DiskReadBytesPerSec"`
				DiskWriteBytesPerSec uint64  `json:"DiskWriteBytesPerSec"`
				NetRxBytesPerSec     uint64  `json:"NetRxBytesPerSec"`
				NetTxBytesPerSec     uint64  `json:"NetTxBytesPerSec"`
				DiskUsed             uint64  `json:"DiskUsed"`
				DiskUsedPercent      float64 `json:"DiskUsedPercent"`
			}

			data, _ := json.Marshal(map[string]interface{}{
				"server_uuid": uuid,
				"timestamp":   host.Timestamp,
				"host": hostLive{
					Timestamp:            host.Timestamp,
					CPU:                  host.CPU,
					MemUsed:              host.MemUsed,
					MemPercent:           host.MemPercent,
					DiskReadBytesPerSec:  host.DiskReadBytesPerSec,
					DiskWriteBytesPerSec: host.DiskWriteBytesPerSec,
					NetRxBytesPerSec:     host.NetRxBytesPerSec,
					NetTxBytesPerSec:     host.NetTxBytesPerSec,
					DiskUsed:             host.DiskUsed,
					DiskUsedPercent:      host.DiskUsedPercent,
				},
				"containers": containerSlice,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
