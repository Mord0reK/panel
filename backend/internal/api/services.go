package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"backend/internal/integrations"
	"backend/internal/models"
	"backend/internal/security"

	"github.com/gorilla/mux"
)

type ServicesHandler struct {
	db     *sql.DB
	cipher *security.ServiceSecretsCipher
}

func NewServicesHandler(db *sql.DB, jwtSecret string) (*ServicesHandler, error) {
	cipher, err := security.NewServiceSecretsCipher(jwtSecret)
	if err != nil {
		return nil, err
	}

	return &ServicesHandler{db: db, cipher: cipher}, nil
}

type ServiceListItem struct {
	Key             string                `json:"key"`
	DisplayName     string                `json:"display_name"`
	Icon            string                `json:"icon"`
	Enabled         bool                  `json:"enabled"`
	RequiresBaseURL bool                  `json:"requires_base_url"`
	AuthType        integrations.AuthType `json:"auth_type"`
	FixedBaseURL    string                `json:"fixed_base_url,omitempty"`
	Endpoints       []string              `json:"endpoints"`
}

type ServiceConfigRequest struct {
	Enabled  *bool   `json:"enabled"`
	BaseURL  *string `json:"base_url"`
	Token    *string `json:"token"`
	Username *string `json:"username"`
	Password *string `json:"password"`
}

type ServiceTestResponse struct {
	Success    bool   `json:"success"`
	ServiceKey string `json:"service_key"`
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (h *ServicesHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	var integrationModel models.ServiceIntegration
	states, err := integrationModel.EnabledMap(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defs := integrations.ListDefinitions()
	response := make([]ServiceListItem, 0, len(defs))
	for _, def := range defs {
		response = append(response, ServiceListItem{
			Key:             def.Key,
			DisplayName:     def.DisplayName,
			Icon:            def.Icon,
			Enabled:         states[def.Key],
			RequiresBaseURL: def.RequiresBaseURL,
			AuthType:        def.AuthType,
			FixedBaseURL:    def.FixedBaseURL,
			Endpoints:       def.Endpoints,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ServicesHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	serviceKey := mux.Vars(r)["service"]
	_, ok := integrations.GetDefinition(serviceKey)
	if !ok {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	var integrationModel models.ServiceIntegration
	existing, err := integrationModel.GetByKey(h.db, serviceKey)
	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service_key":  serviceKey,
			"enabled":      false,
			"base_url":     "",
			"username":     "",
			"has_token":    false,
			"has_password": false,
		})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	username := ""
	if existing.EncryptedUsername != "" {
		decrypted, decryptErr := h.cipher.DecryptString(existing.EncryptedUsername)
		if decryptErr != nil {
			http.Error(w, decryptErr.Error(), http.StatusInternalServerError)
			return
		}
		username = decrypted
	}

	passwordPlaceholder := ""
	if existing.EncryptedPassword != "" {
		passwordPlaceholder = "••••••••••••••••••••••••••••••••"
	}

	tokenPlaceholder := ""
	if existing.EncryptedToken != "" {
		tokenPlaceholder = "••••••••••••••••••••••••••••••••"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_key":  serviceKey,
		"enabled":      existing.Enabled,
		"base_url":     existing.BaseURL,
		"username":     username,
		"password":     passwordPlaceholder,
		"token":        tokenPlaceholder,
		"has_token":    existing.EncryptedToken != "",
		"has_password": existing.EncryptedPassword != "",
	})
}

func (h *ServicesHandler) HandleConfigUpsert(w http.ResponseWriter, r *http.Request) {
	serviceKey := mux.Vars(r)["service"]
	def, ok := integrations.GetDefinition(serviceKey)
	if !ok {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	var req ServiceConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Enabled == nil {
		http.Error(w, "enabled is required", http.StatusBadRequest)
		return
	}

	var integrationModel models.ServiceIntegration
	existing, err := integrationModel.GetByKey(h.db, serviceKey)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if existing == nil {
		existing = &models.ServiceIntegration{ServiceKey: serviceKey}
	}

	next := *existing
	next.ServiceKey = serviceKey
	next.Enabled = *req.Enabled

	if req.BaseURL != nil {
		next.BaseURL = strings.TrimSpace(*req.BaseURL)
	}

	if req.Token != nil {
		token := strings.TrimSpace(*req.Token)
		if token == "" {
			next.EncryptedToken = ""
		} else {
			encryptedToken, encryptErr := h.cipher.EncryptString(token)
			if encryptErr != nil {
				http.Error(w, encryptErr.Error(), http.StatusInternalServerError)
				return
			}
			next.EncryptedToken = encryptedToken
		}
	}

	if req.Username != nil {
		username := strings.TrimSpace(*req.Username)
		if username == "" {
			next.EncryptedUsername = ""
		} else {
			encryptedUsername, encryptErr := h.cipher.EncryptString(username)
			if encryptErr != nil {
				http.Error(w, encryptErr.Error(), http.StatusInternalServerError)
				return
			}
			next.EncryptedUsername = encryptedUsername
		}
	}

	if req.Password != nil {
		password := *req.Password
		if password == "" {
			next.EncryptedPassword = ""
		} else {
			encryptedPassword, encryptErr := h.cipher.EncryptString(password)
			if encryptErr != nil {
				http.Error(w, encryptErr.Error(), http.StatusInternalServerError)
				return
			}
			next.EncryptedPassword = encryptedPassword
		}
	}

	if validateErr := validateConfig(def, &next); validateErr != nil {
		http.Error(w, validateErr.Error(), http.StatusBadRequest)
		return
	}

	if err := next.Upsert(h.db); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success":     true,
		"service_key": serviceKey,
		"enabled":     next.Enabled,
	})
}

func (h *ServicesHandler) HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	serviceKey := mux.Vars(r)["service"]
	def, ok := integrations.GetDefinition(serviceKey)
	if !ok {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	service, ok := integrations.GetService(serviceKey)
	if !ok {
		http.Error(w, "service handler not found", http.StatusNotFound)
		return
	}

	var req ServiceConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var integrationModel models.ServiceIntegration
	existing, err := integrationModel.GetByKey(h.db, serviceKey)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	testCfg, err := h.buildTestConfig(def, req, existing)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	statusCode, err := service.TestConnection(ctx, testCfg)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ServiceTestResponse{
			Success:    false,
			ServiceKey: serviceKey,
			StatusCode: statusCode,
			Message:    err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ServiceTestResponse{
		Success:    true,
		ServiceKey: serviceKey,
		StatusCode: statusCode,
		Message:    "connection test succeeded",
	})
}

func validateConfig(def integrations.Definition, cfg *models.ServiceIntegration) error {
	if !cfg.Enabled {
		return nil
	}

	if def.RequiresBaseURL && strings.TrimSpace(cfg.BaseURL) == "" {
		return errors.New("base_url is required for this service")
	}

	if def.AuthType == integrations.AuthTypeToken && strings.TrimSpace(cfg.EncryptedToken) == "" {
		return errors.New("token is required for this service")
	}

	if def.AuthType == integrations.AuthTypeBasicAuth {
		if strings.TrimSpace(cfg.EncryptedUsername) == "" || strings.TrimSpace(cfg.EncryptedPassword) == "" {
			return errors.New("username and password are required for this service")
		}
	}

	return nil
}

func (h *ServicesHandler) buildTestConfig(def integrations.Definition, req ServiceConfigRequest, existing *models.ServiceIntegration) (integrations.TestConfig, error) {
	cfg := integrations.TestConfig{}

	if def.RequiresBaseURL {
		if req.BaseURL != nil {
			cfg.BaseURL = strings.TrimSpace(*req.BaseURL)
		} else if existing != nil {
			cfg.BaseURL = strings.TrimSpace(existing.BaseURL)
		}
		if cfg.BaseURL == "" {
			return integrations.TestConfig{}, fmt.Errorf("base_url is required")
		}
	} else {
		cfg.BaseURL = def.FixedBaseURL
	}

	if def.AuthType == integrations.AuthTypeToken {
		if req.Token != nil {
			cfg.Token = strings.TrimSpace(*req.Token)
		} else if existing != nil && existing.EncryptedToken != "" {
			decrypted, err := h.cipher.DecryptString(existing.EncryptedToken)
			if err != nil {
				return integrations.TestConfig{}, fmt.Errorf("decrypt token: %w", err)
			}
			cfg.Token = strings.TrimSpace(decrypted)
		}
		if cfg.Token == "" {
			return integrations.TestConfig{}, fmt.Errorf("token is required")
		}
	}

	if def.AuthType == integrations.AuthTypeBasicAuth {
		if req.Username != nil {
			cfg.Username = strings.TrimSpace(*req.Username)
		} else if existing != nil && existing.EncryptedUsername != "" {
			decrypted, err := h.cipher.DecryptString(existing.EncryptedUsername)
			if err != nil {
				return integrations.TestConfig{}, fmt.Errorf("decrypt username: %w", err)
			}
			cfg.Username = strings.TrimSpace(decrypted)
		}

		if req.Password != nil {
			cfg.Password = *req.Password
		} else if existing != nil && existing.EncryptedPassword != "" {
			decrypted, err := h.cipher.DecryptString(existing.EncryptedPassword)
			if err != nil {
				return integrations.TestConfig{}, fmt.Errorf("decrypt password: %w", err)
			}
			cfg.Password = decrypted
		}

		if cfg.Username == "" || cfg.Password == "" {
			return integrations.TestConfig{}, fmt.Errorf("username and password are required")
		}
	}

	return cfg, nil
}

func (h *ServicesHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	serviceKey := mux.Vars(r)["service"]
	def, ok := integrations.GetDefinition(serviceKey)
	if !ok {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	service, ok := integrations.GetService(serviceKey)
	if !ok {
		http.Error(w, "service handler not found", http.StatusNotFound)
		return
	}

	dashboardProvider, ok := service.(integrations.DashboardProvider)
	if !ok {
		http.Error(w, "service stats not supported", http.StatusNotFound)
		return
	}

	var integrationModel models.ServiceIntegration
	existing, err := integrationModel.GetByKey(h.db, serviceKey)
	if err == sql.ErrNoRows {
		http.Error(w, "service configuration not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !existing.Enabled {
		http.Error(w, "service is disabled", http.StatusBadRequest)
		return
	}

	testCfg, err := h.buildTestConfig(def, ServiceConfigRequest{}, existing)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	payload, err := dashboardProvider.FetchDashboard(ctx, testCfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}
