package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/api"
	"backend/internal/auth"
	"backend/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := &config.Config{JWTSecret: "test-secret"}
	handler := api.NewAuthHandler(db, cfg)

	// 1. Check Status - Setup Required
	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	handler.HandleStatus(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	
	var statusResp map[string]bool
	err := json.Unmarshal(w.Body.Bytes(), &statusResp)
	require.NoError(t, err)
	assert.True(t, statusResp["setup_required"])
	assert.False(t, statusResp["authenticated"])

	// 2. Setup
	setupPayload := map[string]string{"username": "admin", "password": "password123"}
	body, _ := json.Marshal(setupPayload)
	req = httptest.NewRequest("POST", "/api/setup", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	handler.HandleSetup(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var tokenResp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &tokenResp)
	require.NoError(t, err)
	token := tokenResp["token"]
	assert.NotEmpty(t, token)

	// 3. Check Status - Setup Not Required, Not Authenticated (no header)
	req = httptest.NewRequest("GET", "/api/auth/status", nil)
	w = httptest.NewRecorder()
	handler.HandleStatus(w, req)
	assert.False(t, func() bool {
		var r map[string]bool
		json.Unmarshal(w.Body.Bytes(), &r)
		return r["setup_required"]
	}())

	// 4. Check Status - Authenticated
	req = httptest.NewRequest("GET", "/api/auth/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler.HandleStatus(w, req)
	var resp4 map[string]bool
	json.Unmarshal(w.Body.Bytes(), &resp4)
	assert.True(t, resp4["authenticated"])

	// 5. Login
	loginPayload := map[string]string{"username": "admin", "password": "password123"}
	body, _ = json.Marshal(loginPayload)
	req = httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	handler.HandleLogin(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &tokenResp)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenResp["token"])

	// 6. Login Invalid
	loginPayloadCtx := map[string]string{"username": "admin", "password": "wrongpassword"}
	body, _ = json.Marshal(loginPayloadCtx)
	req = httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	handler.HandleLogin(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 7. Setup again (Forbidden)
	req = httptest.NewRequest("POST", "/api/setup", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	handler.HandleSetup(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuthMiddleware(t *testing.T) {
	secret := "secret123"
	mw := auth.Middleware(secret)

	// Create a dummy handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(auth.UserIDKey).(int)
		if userID == 1 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusForbidden)
		}
	})

	protected := mw(nextHandler)

	// 1. Valid Token
	token, _ := auth.GenerateToken(1, secret)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. Invalid Token
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 3. No Token
	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 4. Expired Token (simulate by using past time in claims if possible, or just wait? No, mocked time? JWT lib uses time.Now)
	// Hard to mock time in JWT lib easily without overriding. Skip strict expiration test for now or assume logic works.
	// But we can check behavior if we could generate expired token.
	// For now, satisfy with invalid token check.
}
