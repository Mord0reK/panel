package tests

import (
	"bytes"
	"encoding/json"
	"io"
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

func TestServicesAPIAdGuardHomeStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	adguard := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "super-secret-pass" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/control/status":
			_, _ = w.Write([]byte(`{
				"protection_enabled": true,
				"running": true,
				"version": "v0.107.62"
			}`))
		case "/control/stats":
			_, _ = w.Write([]byte(`{
				"time_units": "hours",
				"num_dns_queries": 52369,
				"num_blocked_filtering": 13966,
				"num_replaced_safebrowsing": 0,
				"num_replaced_safesearch": 735,
				"num_replaced_parental": 0,
				"avg_processing_time": 6.2,
				"dns_queries": [100, 120, 115],
				"blocked_filtering": [20, 30, 25],
				"replaced_safebrowsing": [0, 0, 0],
				"replaced_parental": [0, 0, 0],
				"top_queried_domains": [{"cdn.samsungcloudsolution.com": 2426}],
				"top_blocked_domains": [{"logs.netflix.com": 2192}],
				"top_clients": [{"093105118218.siedlce.vectranet.pl": 45734}]
			}`))
		case "/control/version.json":
			assert.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), "recheck_now")
			_, _ = w.Write([]byte(`{
				"new_version": "v0.107.63",
				"announcement": "AdGuard Home v0.107.63 is now available!",
				"announcement_url": "https://github.com/AdguardTeam/AdGuardHome/releases/tag/v0.107.63",
				"can_autoupdate": true
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer adguard.Close()

	handler, err := api.NewServicesHandler(db, "test-secret")
	require.NoError(t, err)

	r := mux.NewRouter()
	r.HandleFunc("/api/services/{service}/config", handler.HandleConfigUpsert).Methods("PUT")
	r.HandleFunc("/api/services/{service}/stats", handler.HandleStats).Methods("GET")

	configBody := map[string]any{
		"enabled":  true,
		"base_url": adguard.URL,
		"username": "admin",
		"password": "super-secret-pass",
	}
	configPayload, err := json.Marshal(configBody)
	require.NoError(t, err)

	configReq := httptest.NewRequest("PUT", "/api/services/adguardhome/config", bytes.NewBuffer(configPayload))
	configW := httptest.NewRecorder()
	r.ServeHTTP(configW, configReq)
	assert.Equal(t, http.StatusOK, configW.Code)

	req := httptest.NewRequest("GET", "/api/services/adguardhome/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var payload map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &payload)
	require.NoError(t, err)

	assert.Equal(t, "adguardhome", payload["service_key"])

	status, ok := payload["status"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, status["protection_enabled"])
	assert.Equal(t, true, status["running"])

	version, ok := payload["version"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "v0.107.62", version["current_version"])
	assert.Equal(t, "v0.107.63", version["latest_version"])
	assert.Equal(t, true, version["update_available"])

	stats, ok := payload["stats"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(52369), stats["num_dns_queries"])
	assert.Equal(t, float64(13966), stats["num_blocked_filtering"])
	assert.Equal(t, float64(735), stats["num_replaced_safesearch"])

	topClients, ok := stats["top_clients"].([]any)
	require.True(t, ok)
	require.Len(t, topClients, 1)
	firstClient, ok := topClients[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "093105118218.siedlce.vectranet.pl", firstClient["name"])
	assert.Equal(t, float64(45734), firstClient["count"])
}
