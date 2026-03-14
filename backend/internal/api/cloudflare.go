package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"backend/internal/integrations"
	"backend/internal/models"
	"backend/internal/security"
)

type CloudflareHandler struct {
	db     *sql.DB
	cipher *security.ServiceSecretsCipher
}

func NewCloudflareHandler(db *sql.DB, jwtSecret string) (*CloudflareHandler, error) {
	cipher, err := security.NewServiceSecretsCipher(jwtSecret)
	if err != nil {
		return nil, err
	}
	return &CloudflareHandler{db: db, cipher: cipher}, nil
}

type cloudflareZonesRequest struct {
	APIToken string `json:"api_token"`
}

// HandleZones handles POST /api/services/cloudflare/zones
// Accepts an optional {"api_token": "..."} in body.
// If no token is provided, reads the token from the saved (encrypted) integration config.
func (h *CloudflareHandler) HandleZones(w http.ResponseWriter, r *http.Request) {
	var req cloudflareZonesRequest
	// Ignore decode error — empty or missing body is valid (falls back to saved config)
	_ = json.NewDecoder(r.Body).Decode(&req)

	apiToken := strings.TrimSpace(req.APIToken)

	if apiToken == "" {
		// Fallback: read token from saved config
		var integrationModel models.ServiceIntegration
		existing, err := integrationModel.GetByKey(h.db, "cloudflare")
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "api_token is required or cloudflare integration must be configured", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		decrypted, err := h.cipher.DecryptString(existing.EncryptedPassword)
		if err != nil {
			http.Error(w, "failed to decrypt API token", http.StatusInternalServerError)
			return
		}
		apiToken = strings.TrimSpace(decrypted)
		if apiToken == "" {
			http.Error(w, "api_token is required", http.StatusBadRequest)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	zones, err := integrations.FetchCloudflareZones(ctx, apiToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(zones)
}

// HandleDNSRecords handles GET /api/services/cloudflare/dns
// Reads Zone ID and API token from the saved (encrypted) integration config.
func (h *CloudflareHandler) HandleDNSRecords(w http.ResponseWriter, r *http.Request) {
	var integrationModel models.ServiceIntegration
	existing, err := integrationModel.GetByKey(h.db, "cloudflare")
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "cloudflare integration not configured", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !existing.Enabled {
		http.Error(w, "cloudflare integration is disabled", http.StatusBadRequest)
		return
	}

	zoneID, err := h.cipher.DecryptString(existing.EncryptedUsername)
	if err != nil {
		http.Error(w, "failed to decrypt zone ID", http.StatusInternalServerError)
		return
	}
	apiToken, err := h.cipher.DecryptString(existing.EncryptedPassword)
	if err != nil {
		http.Error(w, "failed to decrypt API token", http.StatusInternalServerError)
		return
	}

	zoneID = strings.TrimSpace(zoneID)
	apiToken = strings.TrimSpace(apiToken)

	if zoneID == "" || apiToken == "" {
		http.Error(w, "cloudflare zone ID and API token must be configured", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	records, err := integrations.FetchCloudflareDNSRecords(ctx, apiToken, zoneID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}
