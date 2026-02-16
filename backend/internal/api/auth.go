package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/models"
)

type AuthHandler struct {
	db  *sql.DB
	cfg *config.Config
}

func NewAuthHandler(db *sql.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

type setupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type statusResponse struct {
	SetupRequired bool `json:"setup_required"`
	Authenticated bool `json:"authenticated"`
}

func (h *AuthHandler) HandleSetup(w http.ResponseWriter, r *http.Request) {
	var userModel models.User
	exists, err := userModel.Exists(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "Setup already completed", http.StatusForbidden)
		return
	}

	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Username) < 3 || len(req.Password) < 8 {
		http.Error(w, "Invalid username or password length", http.StatusBadRequest)
		return
	}

	if err := userModel.Create(h.db, req.Username, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-login after setup
	user, err := userModel.Authenticate(h.db, req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(user.ID, h.cfg.JWTSecret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResponse{Token: token})
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var userModel models.User
	user, err := userModel.Authenticate(h.db, req.Username, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID, h.cfg.JWTSecret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResponse{Token: token})
}

func (h *AuthHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	var userModel models.User
	exists, err := userModel.Exists(h.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := statusResponse{
		SetupRequired: !exists,
		Authenticated: false,
	}

	// Check authentication manually
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString != authHeader {
			_, err := auth.ValidateToken(tokenString, h.cfg.JWTSecret)
			if err == nil {
				resp.Authenticated = true
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
