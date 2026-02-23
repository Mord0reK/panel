package services

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type AdGuardService struct {
	url      string
	username string
	password string
}

func init() {
	Register(&AdGuardService{})
}

func (s *AdGuardService) Info() ServiceInfo {
	return ServiceInfo{
		Name:        "AdGuard Home",
		Slug:        "adguardhome",
		Description: "Sieciowy filtr reklam i śledzenia",
		Icon:        "ShieldCheck",
	}
}

func (s *AdGuardService) ConfigSchema() []ConfigField {
	return []ConfigField{
		{
			Name:        "url",
			Label:       "URL Adresu",
			Type:        "url",
			Required:    true,
			Description: "Np. http://192.168.1.100:80",
		},
		{
			Name:        "username",
			Label:       "Użytkownik",
			Type:        "text",
			Required:    true,
		},
		{
			Name:        "password",
			Label:       "Hasło",
			Type:        "password",
			Required:    true,
		},
	}
}

func (s *AdGuardService) Initialize(config map[string]string) error {
	s.url = strings.TrimSuffix(config["url"], "/")
	s.username = config["username"]
	s.password = config["password"]

	if s.url == "" {
		return fmt.Errorf("URL is required")
	}
	return nil
}

func (s *AdGuardService) HandleRequest(w http.ResponseWriter, r *http.Request, path string) {
	if s.url == "" {
		http.Error(w, "Service not initialized", http.StatusInternalServerError)
		return
	}

	// Example: proxying to AdGuard Home API
	targetURL := fmt.Sprintf("%s/%s", s.url, path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers
	for k, v := range r.Header {
		if k != "Host" && k != "Authorization" {
			proxyReq.Header[k] = v
		}
	}

	// Add AdGuard Authentication
	proxyReq.SetBasicAuth(s.username, s.password)

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers and status
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
