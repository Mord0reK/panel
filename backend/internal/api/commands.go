package api

import (
	"encoding/json"
	"net/http"

	ws "backend/internal/websocket"

	"github.com/gorilla/mux"
)

type CommandsHandler struct {
	hub *ws.AgentHub
}

func NewCommandsHandler(hub *ws.AgentHub) *CommandsHandler {
	return &CommandsHandler{hub: hub}
}

type commandRequest struct {
	Action string `json:"action"`
	Target string `json:"target"`
}

func (h *CommandsHandler) HandleCommand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.hub.RequestAgent(uuid, req.Action, req.Target)
	if err != nil {
		if err.Error() == "agent not connected" {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (h *CommandsHandler) HandleContainerCommand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	containerID := vars["id"]

	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use containerID as target
	resp, err := h.hub.RequestAgent(uuid, req.Action, containerID)
	if err != nil {
		if err.Error() == "agent not connected" {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (h *CommandsHandler) HandleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	containerID := vars["id"]

	resp, err := h.hub.RequestAgent(uuid, "check-updates", containerID)
	if err != nil {
		if err.Error() == "agent not connected" {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (h *CommandsHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	containerID := vars["id"]

	resp, err := h.hub.RequestAgent(uuid, "update", containerID)
	if err != nil {
		if err.Error() == "agent not connected" {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}
