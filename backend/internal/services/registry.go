package services

import (
	"fmt"
	"net/http"
	"sync"
)

var (
	registryMu sync.RWMutex
	availableServices = make(map[string]Service)
	activeInstances   = make(map[string]Service)
)

// Register adds a service implementation to the registry
// This should be called from the service's init() function
func Register(s Service) {
	registryMu.Lock()
	defer registryMu.Unlock()
	slug := s.Info().Slug
	if _, dup := availableServices[slug]; dup {
		panic(fmt.Sprintf("service already registered: %s", slug))
	}
	availableServices[slug] = s
}

// GetAvailableServices returns all registered service implementations
func GetAvailableServices() []Service {
	registryMu.RLock()
	defer registryMu.RUnlock()
	services := make([]Service, 0, len(availableServices))
	for _, s := range availableServices {
		services = append(services, s)
	}
	return services
}

// GetService returns a specific registered service by slug
func GetService(slug string) (Service, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	s, ok := availableServices[slug]
	return s, ok
}

// InitializeService configures and activates a service instance
func InitializeService(slug string, config map[string]string) error {
	registryMu.Lock()
	defer registryMu.Unlock()

	s, ok := availableServices[slug]
	if !ok {
		return fmt.Errorf("service not found: %s", slug)
	}

	if err := s.Initialize(config); err != nil {
		return err
	}

	activeInstances[slug] = s
	return nil
}

// HandleServiceRequest routes a request to the appropriate active service
func HandleServiceRequest(slug string, path string, w http.ResponseWriter, r *http.Request) {
	registryMu.RLock()
	instance, ok := activeInstances[slug]
	registryMu.RUnlock()

	if !ok {
		http.Error(w, "Service not configured or not found", http.StatusNotFound)
		return
	}

	instance.HandleRequest(w, r, path)
}
