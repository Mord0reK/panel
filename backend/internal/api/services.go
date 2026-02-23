package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/services"

	"github.com/gorilla/mux"
)

type ServicesHandler struct {
	db     *sql.DB
	config *config.Config
}

func NewServicesHandler(db *sql.DB, cfg *config.Config) *ServicesHandler {
	return &ServicesHandler{db: db, config: cfg}
}

func (h *ServicesHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	available := services.GetAvailableServices()

	model := &models.ServiceConfig{}
	configs, err := model.GetAll(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	configMap := make(map[string]models.ServiceConfig)
	for _, c := range configs {
		configMap[c.Slug] = c
	}

	type serviceResp struct {
		Name        string                 `json:"name"`
		Slug        string                 `json:"slug"`
		Description string                 `json:"description"`
		Icon        string                 `json:"icon"`
		IsEnabled   bool                   `json:"is_enabled"`
		Schema      []services.ConfigField `json:"schema"`
	}

	resp := make([]serviceResp, 0)
	for _, s := range available {
		info := s.Info()
		isEnabled := false
		if conf, ok := configMap[info.Slug]; ok {
			isEnabled = conf.IsEnabled
		}

		resp = append(resp, serviceResp{
			Name:        info.Name,
			Slug:        info.Slug,
			Description: info.Description,
			Icon:        info.Icon,
			IsEnabled:   isEnabled,
			Schema:      s.ConfigSchema(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ServicesHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	service, ok := services.GetService(slug)
	if !ok {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	model := &models.ServiceConfig{}
	conf, err := model.GetBySlug(h.db, slug)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty config if not set yet
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	decryptedJSON, err := services.Decrypt(conf.ConfigJSONEncrypted, h.config.EncryptionKey)
	if err != nil {
		http.Error(w, "Failed to decrypt config", http.StatusInternalServerError)
		return
	}

	var configMap map[string]string
	if err := json.Unmarshal([]byte(decryptedJSON), &configMap); err != nil {
		http.Error(w, "Failed to unmarshal config", http.StatusInternalServerError)
		return
	}

	// Mask passwords
	schema := service.ConfigSchema()
	for _, field := range schema {
		if field.Type == "password" {
			if val, ok := configMap[field.Name]; ok && val != "" {
				configMap[field.Name] = "********"
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configMap)
}

func (h *ServicesHandler) HandleSaveConfig(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	_, ok := services.GetService(slug)
	if !ok {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	var req struct {
		Config    map[string]string `json:"config"`
		IsEnabled bool              `json:"is_enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If passwords are masked (********), keep original values from DB
	model := &models.ServiceConfig{}
	existing, err := model.GetBySlug(h.db, slug)
	if err == nil {
		decryptedJSON, err := services.Decrypt(existing.ConfigJSONEncrypted, h.config.EncryptionKey)
		if err == nil {
			var existingMap map[string]string
			if err := json.Unmarshal([]byte(decryptedJSON), &existingMap); err == nil {
				for k, v := range req.Config {
					if v == "********" {
						if oldVal, ok := existingMap[k]; ok {
							req.Config[k] = oldVal
						}
					}
				}
			}
		}
	}

	configBytes, _ := json.Marshal(req.Config)
	encryptedJSON, err := services.Encrypt(string(configBytes), h.config.EncryptionKey)
	if err != nil {
		http.Error(w, "Failed to encrypt config", http.StatusInternalServerError)
		return
	}

	if err := model.Save(h.db, slug, encryptedJSON, req.IsEnabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If enabled, initialize service
	if req.IsEnabled {
		if err := services.InitializeService(slug, req.Config); err != nil {
			// We still saved it, but notify about init error
			http.Error(w, "Config saved but service failed to initialize: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *ServicesHandler) HandleProxy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	path := vars["path"]

	services.HandleServiceRequest(slug, path, w, r)
}
