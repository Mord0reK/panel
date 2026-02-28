package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/api"
	"backend/internal/models"
	ws "backend/internal/websocket"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServersAPI(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hub := ws.NewHub()
	go hub.Run()
	handler := api.NewServersHandler(db, hub)
	r := mux.NewRouter()
	r.HandleFunc("/api/servers", handler.HandleList).Methods("GET")
	r.HandleFunc("/api/servers/{uuid}", handler.HandleGet).Methods("GET")
	r.HandleFunc("/api/servers/{uuid}/approve", handler.HandleApprove).Methods("PUT")
	r.HandleFunc("/api/servers/{uuid}", handler.HandleDelete).Methods("DELETE")

	// 1. Prepare Data
	_, err := db.Exec("INSERT INTO servers (uuid, hostname, approved) VALUES (?, ?, ?)", "s1", "host1", false)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO containers (agent_uuid, container_id, name) VALUES (?, ?, ?)", "s1", "c1", "cont1")
	require.NoError(t, err)

	// 2. Test List
	req := httptest.NewRequest("GET", "/api/servers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var servers []models.Server
	json.Unmarshal(w.Body.Bytes(), &servers)
	assert.Len(t, servers, 1)
	assert.Equal(t, "s1", servers[0].UUID)

	// 3. Test Get
	req = httptest.NewRequest("GET", "/api/servers/s1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var getResp struct {
		Server     models.Server      `json:"server"`
		Containers []models.Container `json:"containers"`
	}
	json.Unmarshal(w.Body.Bytes(), &getResp)
	assert.Equal(t, "s1", getResp.Server.UUID)
	assert.Len(t, getResp.Containers, 1)

	// 4. Test Approve
	req = httptest.NewRequest("PUT", "/api/servers/s1/approve", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var approved bool
	db.QueryRow("SELECT approved FROM servers WHERE uuid='s1'").Scan(&approved)
	assert.True(t, approved)

	// 5. Test Delete
	req = httptest.NewRequest("DELETE", "/api/servers/s1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM servers").Scan(&count)
	assert.Equal(t, 0, count)
	// Cascade check
	db.QueryRow("SELECT COUNT(*) FROM containers").Scan(&count)
	assert.Equal(t, 0, count)
}

func TestDeleteContainersBulk(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Setup: wstaw 3 kontenery
	_, err := db.Exec("INSERT INTO servers (uuid, hostname, approved) VALUES (?, ?, ?)", "agent1", "host1", true)
	require.NoError(t, err)
	for _, id := range []string{"c1", "c2", "c3"} {
		_, err = db.Exec("INSERT INTO containers (agent_uuid, container_id, name) VALUES (?, ?, ?)", "agent1", id, id)
		require.NoError(t, err)
	}

	var cont models.Container
	deleted, failed := cont.DeleteBulk(db, "agent1", []string{"c1", "c2"})
	assert.Len(t, deleted, 2)
	assert.Len(t, failed, 0)
	assert.ElementsMatch(t, []string{"c1", "c2"}, deleted)

	// c3 powinien nadal istnieć
	var count int
	db.QueryRow("SELECT COUNT(*) FROM containers WHERE agent_uuid='agent1'").Scan(&count)
	assert.Equal(t, 1, count)

	// Pusta lista — brak błędu
	deleted, failed = cont.DeleteBulk(db, "agent1", []string{})
	assert.Len(t, deleted, 0)
	assert.Len(t, failed, 0)
}
