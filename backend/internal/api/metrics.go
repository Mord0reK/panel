package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"backend/internal/models"

	"github.com/gorilla/mux"
)

type MetricsHandler struct {
	db *sql.DB
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
	Timestamp int64   `json:"timestamp"`
	CPU       float64 `json:"cpu"`

	MemUsed uint64 `json:"mem_used"`

	DiskReadBytesPerSec  uint64 `json:"disk_read_bytes_per_sec"`
	DiskWriteBytesPerSec uint64 `json:"disk_write_bytes_per_sec"`
	NetRxBytesPerSec     uint64 `json:"net_rx_bytes_per_sec"`
	NetTxBytesPerSec     uint64 `json:"net_tx_bytes_per_sec"`
}

type serverContainerHistory struct {
	ContainerID string      `json:"container_id"`
	Name        string      `json:"name"`
	Image       string      `json:"image"`
	Project     string      `json:"project"`
	Service     string      `json:"service"`
	Points      interface{} `json:"points"`
}

func NewMetricsHandler(db *sql.DB) *MetricsHandler {
	return &MetricsHandler{db: db}
}

func toRawPoints(points []models.HistoricalMetricPoint) []rawMetricPoint {
	rawPoints := make([]rawMetricPoint, len(points))
	for i, p := range points {
		rawPoints[i] = rawMetricPoint{
			Timestamp:  p.Timestamp,
			CPU:        p.CPUAvg,
			MemUsed:    uint64(p.MemAvg),
			DiskUsed:   uint64(p.DiskAvg),
			NetRxBytes: uint64(p.NetRxAvg),
			NetTxBytes: uint64(p.NetTxAvg),
		}
	}
	return rawPoints
}

func toRawHostPoints(points []models.HostHistoricalMetricPoint) []rawHostMetricPoint {
	rawPoints := make([]rawHostMetricPoint, len(points))
	for i, p := range points {
		rawPoints[i] = rawHostMetricPoint{
			Timestamp:            p.Timestamp,
			CPU:                  p.CPUAvg,
			MemUsed:              uint64(p.MemAvg),
			DiskReadBytesPerSec:  uint64(p.DiskReadAvg),
			DiskWriteBytesPerSec: uint64(p.DiskWriteAvg),
			NetRxBytesPerSec:     uint64(p.NetRxAvg),
			NetTxBytesPerSec:     uint64(p.NetTxAvg),
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

	points, err := models.GetHistoricalMetrics(h.db, uuid, containerID, rangeKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rangeKey == "1m" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"points": toRawPoints(points)})
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

	hostPoints, err := models.GetHostHistoricalMetrics(h.db, uuid, rangeKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var containerModel models.Container
	containers, err := containerModel.GetByAgent(h.db, uuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var containersResp []serverContainerHistory
	for _, c := range containers {
		points, err := models.GetHistoricalMetrics(h.db, uuid, c.ContainerID, rangeKey)
		if err != nil || len(points) == 0 {
			continue
		}

		var payload interface{}
		if rangeKey == "1m" {
			payload = toRawPoints(points)
		} else {
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

	var hostPayload interface{}
	if rangeKey == "1m" {
		hostPayload = toRawHostPoints(hostPoints)
	} else {
		hostPayload = hostPoints
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"points":      hostPayload, // compatibility alias
		"host_points": hostPayload,
		"containers":  containersResp,
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
