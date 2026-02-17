package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"backend/internal/buffer"
	"backend/internal/models"

	"github.com/gorilla/mux"
)

type MetricsHandler struct {
	db            *sql.DB
	bufferManager *buffer.BufferManager
}

type rawMetricPoint struct {
	Timestamp  int64   `json:"timestamp"`
	CPU        float64 `json:"cpu"`
	MemUsed    uint64  `json:"mem_used"`
	DiskUsed   uint64  `json:"disk_used"`
	NetRxBytes uint64  `json:"net_rx_bytes"`
	NetTxBytes uint64  `json:"net_tx_bytes"`
}

type rawHostMetricPoint struct {
	Timestamp            int64   `json:"timestamp"`
	CPU                  float64 `json:"cpu"`
	MemUsed              uint64  `json:"mem_used"`
	MemPercent           float64 `json:"mem_percent"`
	DiskReadBytesPerSec  uint64  `json:"disk_read_bytes_per_sec"`
	DiskWriteBytesPerSec uint64  `json:"disk_write_bytes_per_sec"`
	NetRxBytesPerSec     uint64  `json:"net_rx_bytes_per_sec"`
	NetTxBytesPerSec     uint64  `json:"net_tx_bytes_per_sec"`
}

type serverHostHistory struct {
	Points interface{} `json:"points"`
}

type serverContainerHistory struct {
	ContainerID string      `json:"container_id"`
	Name        string      `json:"name"`
	Image       string      `json:"image"`
	Project     string      `json:"project"`
	Service     string      `json:"service"`
	Points      interface{} `json:"points"`
}

func NewMetricsHandler(db *sql.DB, bufferManager *buffer.BufferManager) *MetricsHandler {
	return &MetricsHandler{db: db, bufferManager: bufferManager}
}

func toRawBufferPoints(points []buffer.MetricPoint) []rawMetricPoint {
	rawPoints := make([]rawMetricPoint, len(points))
	for i, p := range points {
		rawPoints[i] = rawMetricPoint{
			Timestamp:  p.Timestamp,
			CPU:        p.CPU,
			MemUsed:    p.MemUsed,
			DiskUsed:   p.DiskUsed,
			NetRxBytes: p.NetRx,
			NetTxBytes: p.NetTx,
		}
	}
	return rawPoints
}

func toRawHostBufferPoints(points []buffer.HostMetricPoint) []rawHostMetricPoint {
	rawPoints := make([]rawHostMetricPoint, len(points))
	for i, p := range points {
		rawPoints[i] = rawHostMetricPoint{
			Timestamp:            p.Timestamp,
			CPU:                  p.CPU,
			MemUsed:              p.MemUsed,
			MemPercent:           p.MemPercent,
			DiskReadBytesPerSec:  p.DiskReadBytesPerSec,
			DiskWriteBytesPerSec: p.DiskWriteBytesPerSec,
			NetRxBytesPerSec:     p.NetRxBytesPerSec,
			NetTxBytesPerSec:     p.NetTxBytesPerSec,
		}
	}
	return rawPoints
}

func (h *MetricsHandler) HandleContainerHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	containerID := vars["id"]
	rangeKey := r.URL.Query().Get("range")
	if rangeKey == "" {
		rangeKey = "1h"
	}

	if !isValidRange(rangeKey) {
		http.Error(w, "Invalid range parameter", http.StatusBadRequest)
		return
	}

	if rangeKey == "1m" {
		points := h.bufferManager.GetContainerPoints(uuid, containerID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"points": toRawBufferPoints(points)})
		return
	}

	points, err := models.GetHistoricalMetrics(h.db, uuid, containerID, rangeKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"points": points})
}

func (h *MetricsHandler) HandleServerHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	rangeKey := r.URL.Query().Get("range")
	if rangeKey == "" {
		rangeKey = "1h"
	}

	if !isValidRange(rangeKey) {
		http.Error(w, "Invalid range parameter", http.StatusBadRequest)
		return
	}

	var hostPayload interface{}
	if rangeKey == "1m" {
		hostPayload = toRawHostBufferPoints(h.bufferManager.GetHostPoints(uuid))
	} else {
		hostPoints, err := models.GetHistoricalHostMetrics(h.db, uuid, rangeKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hostPayload = hostPoints
	}

	var containerModel models.Container
	containers, err := containerModel.GetByAgent(h.db, uuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var containersResp []serverContainerHistory
	for _, c := range containers {
		var payload interface{}
		if rangeKey == "1m" {
			points := h.bufferManager.GetContainerPoints(uuid, c.ContainerID)
			if len(points) == 0 {
				continue
			}
			payload = toRawBufferPoints(points)
		} else {
			points, err := models.GetHistoricalMetrics(h.db, uuid, c.ContainerID, rangeKey)
			if err != nil || len(points) == 0 {
				continue
			}
			payload = points
		}

		containersResp = append(containersResp, serverContainerHistory{
			ContainerID: c.ContainerID,
			Name:        c.Name,
			Image:       c.Image,
			Project:     c.Project,
			Service:     c.Service,
			Points:      payload,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"host":       serverHostHistory{Points: hostPayload},
		"containers": containersResp,
	})
}

func isValidRange(rangeKey string) bool {
	allowed := []string{"1m", "5m", "15m", "30m", "1h", "6h", "12h", "24h", "7d", "15d", "30d"}
	for _, a := range allowed {
		if a == rangeKey {
			return true
		}
	}
	return false
}
