package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"backend/internal/models"
	ws "backend/internal/websocket"

	"github.com/gorilla/mux"
)

type ServersHandler struct {
	db  *sql.DB
	hub *ws.AgentHub
}

func NewServersHandler(db *sql.DB, hub *ws.AgentHub) *ServersHandler {
	return &ServersHandler{db: db, hub: hub}
}

func (h *ServersHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	var serverModel models.Server
	servers, err := serverModel.GetAll(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

func (h *ServersHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	var serverModel models.Server
	srv, err := serverModel.GetByUUID(h.db, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Server not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	var containerModel models.Container
	containers, err := containerModel.GetByAgent(h.db, uuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := struct {
		Server     *models.Server     `json:"server"`
		Containers []models.Container `json:"containers"`
	}{
		Server:     srv,
		Containers: containers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ServersHandler) HandleApprove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	var serverModel models.Server
	if err := serverModel.Approve(h.db, uuid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.hub != nil {
		authResp := ws.AuthResponseMessage{
			Type:     ws.MsgTypeAuthResponse,
			Approved: true,
		}

		if payload, err := json.Marshal(authResp); err != nil {
			log.Printf("failed to marshal approval response for %s: %v", uuid, err)
		} else {
			h.hub.SetApproved(uuid, true)
			if err := h.hub.SendToAgent(uuid, payload); err != nil {
				log.Printf("server %s approved but not pushed to websocket: %v", uuid, err)
			} else {
				log.Printf("server %s approved and pushed to connected agent", uuid)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *ServersHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	var serverModel models.Server
	if err := serverModel.Delete(h.db, uuid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// PatchServerRequest defines editable fields for PATCH /api/servers/:uuid.
type PatchServerRequest struct {
	DisplayName *string `json:"display_name"`
	Icon        *string `json:"icon"`
	Status      *string `json:"status"`
}

func (h *ServersHandler) HandlePatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	var req PatchServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var serverModel models.Server
	srv, err := serverModel.GetByUUID(h.db, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "server not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	displayName := srv.DisplayName
	icon := srv.Icon
	status := srv.Status

	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}
	if req.Icon != nil {
		icon = *req.Icon
	}
	if req.Status != nil {
		if *req.Status != "active" && *req.Status != "rejected" {
			http.Error(w, "status must be 'active' or 'rejected'", http.StatusBadRequest)
			return
		}
		status = *req.Status
	}

	if err := serverModel.UpdateMeta(h.db, uuid, displayName, icon, status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Disconnect active agent when rejected so it stops sending data.
	if status == "rejected" && h.hub != nil {
		h.hub.DisconnectAgent(uuid)
	}

	updated, err := serverModel.GetByUUID(h.db, uuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}
