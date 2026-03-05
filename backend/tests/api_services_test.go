package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/api"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicesAPIList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO service_integrations (service_key, enabled)
		VALUES (?, ?), (?, ?)
	`, "adguardhome", true, "jellyfin", false)
	require.NoError(t, err)

	handler, err := api.NewServicesHandler(db, "test-secret")
	require.NoError(t, err)

	r := mux.NewRouter()
	r.HandleFunc("/api/services", handler.HandleList).Methods("GET")

	req := httptest.NewRequest("GET", "/api/services", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var services []map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &services)
	require.NoError(t, err)
	assert.NotEmpty(t, services)

	serviceByKey := make(map[string]map[string]interface{})
	for _, service := range services {
		key, ok := service["key"].(string)
		if !ok {
			continue
		}
		serviceByKey[key] = service
	}

	adguard, ok := serviceByKey["adguardhome"]
	require.True(t, ok)
	assert.Equal(t, true, adguard["enabled"])
	assert.Equal(t, "basic_auth", adguard["auth_type"])
	assert.Equal(t, true, adguard["requires_base_url"])
	assert.NotEmpty(t, adguard["icon"])

	jellyfin, ok := serviceByKey["jellyfin"]
	require.True(t, ok)
	assert.Equal(t, false, jellyfin["enabled"])
	assert.Equal(t, "token", jellyfin["auth_type"])
	assert.Equal(t, true, jellyfin["requires_base_url"])

	cloudflare, ok := serviceByKey["cloudflare"]
	require.True(t, ok)
	assert.Equal(t, false, cloudflare["enabled"])
	assert.Equal(t, "token", cloudflare["auth_type"])
	assert.Equal(t, false, cloudflare["requires_base_url"])

	tailscale, ok := serviceByKey["tailscale"]
	require.True(t, ok)
	assert.Equal(t, false, tailscale["enabled"])
	assert.Equal(t, "token", tailscale["auth_type"])
	assert.Equal(t, false, tailscale["requires_base_url"])
}

func TestServicesAPIConfigUpsertEncryptsSecrets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler, err := api.NewServicesHandler(db, "test-secret")
	require.NoError(t, err)

	r := mux.NewRouter()
	r.HandleFunc("/api/services/{service}/config", handler.HandleConfigUpsert).Methods("PUT")

	body := map[string]any{
		"enabled":  true,
		"base_url": "http://adguard.local:3000",
		"username": "admin",
		"password": "super-secret-pass",
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/api/services/adguardhome/config", bytes.NewBuffer(payload))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var (
		enabled           bool
		baseURL           string
		encryptedUsername string
		encryptedPassword string
	)
	err = db.QueryRow(`
		SELECT enabled, base_url, encrypted_username, encrypted_password
		FROM service_integrations
		WHERE service_key = ?
	`, "adguardhome").Scan(&enabled, &baseURL, &encryptedUsername, &encryptedPassword)
	require.NoError(t, err)

	assert.True(t, enabled)
	assert.Equal(t, "http://adguard.local:3000", baseURL)
	assert.NotEmpty(t, encryptedUsername)
	assert.NotEmpty(t, encryptedPassword)
	assert.NotEqual(t, "admin", encryptedUsername)
	assert.NotEqual(t, "super-secret-pass", encryptedPassword)
}

func TestServicesAPITestConnectionValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler, err := api.NewServicesHandler(db, "test-secret")
	require.NoError(t, err)

	r := mux.NewRouter()
	r.HandleFunc("/api/services/{service}/test", handler.HandleTestConnection).Methods("POST")

	req := httptest.NewRequest("POST", "/api/services/adguardhome/test", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
