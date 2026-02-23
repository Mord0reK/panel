package services

import (
	"net/http"
)

// ConfigField defines a single configuration field for a service
type ConfigField struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // text, password, url, number, boolean
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
}

// ServiceInfo contains metadata about the service
type ServiceInfo struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"` // Lucide icon name or URL
}

// Service is the interface that all external services must implement
type Service interface {
	// Info returns metadata about the service
	Info() ServiceInfo

	// ConfigSchema returns the required configuration fields for the service
	ConfigSchema() []ConfigField

	// Initialize is called when the service is registered or configuration changes
	// It should setup internal clients, transports, etc.
	Initialize(config map[string]string) error

	// HandleRequest handles API/Proxy requests for this service
	// path is the sub-path after /api/services/{slug}/
	HandleRequest(w http.ResponseWriter, r *http.Request, path string)
}
