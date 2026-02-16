package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"backend/internal/models"

	"github.com/gorilla/mux"
)

type ServersHandler struct {
	db *sql.DB
}

func NewServersHandler(db *sql.DB) *ServersHandler {
	return &ServersHandler{db: db}
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
