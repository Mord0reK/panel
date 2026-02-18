package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"backend/internal/buffer"
	"backend/internal/models"

	"github.com/gorilla/mux"
)

type SSEHandler struct {
	db            *sql.DB
	bufferManager *buffer.BufferManager
	corsOrigin    string
}

func NewSSEHandler(db *sql.DB, bufferManager *buffer.BufferManager, corsOrigin string) *SSEHandler {
	return &SSEHandler{db: db, bufferManager: bufferManager, corsOrigin: corsOrigin}
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
			// 2. Fetch all approved servers
			var serverModel models.Server
			servers, err := serverModel.GetAll(h.db)
			if err != nil {
				continue
			}

			type serverLive struct {
				UUID     string  `json:"uuid"`
				Hostname string  `json:"hostname"`
				CPU      float64 `json:"cpu"`
				Memory   uint64  `json:"memory"`

				DiskReadBytesPerSec  uint64 `json:"disk_read_bytes_per_sec"`
				DiskWriteBytesPerSec uint64 `json:"disk_write_bytes_per_sec"`
				NetRxBytesPerSec     uint64 `json:"net_rx_bytes_per_sec"`
				NetTxBytesPerSec     uint64 `json:"net_tx_bytes_per_sec"`
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
					UUID:     s.UUID,
					Hostname: s.Hostname,
					CPU:      host.CPU,
					Memory:   host.MemUsed,

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
			containers := h.bufferManager.GetLatestForServerAtTimestamp(uuid, host.Timestamp)

			data, _ := json.Marshal(map[string]interface{}{
				"server_uuid": uuid,
				"timestamp":   host.Timestamp,
				"host":        host,
				"containers":  containers,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
