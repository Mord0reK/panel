package integrations

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type adGuardHomeService struct{}

type AdGuardTopItem struct {
	Name  string `json:"name"`
	Count uint64 `json:"count"`
}

type AdGuardHomeStatus struct {
	ProtectionEnabled bool `json:"protection_enabled"`
	Running           bool `json:"running"`
}

type AdGuardHomeVersion struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version,omitempty"`
	Announcement    string `json:"announcement,omitempty"`
	AnnouncementURL string `json:"announcement_url,omitempty"`
	CanAutoUpdate   bool   `json:"can_auto_update"`
	UpdateAvailable bool   `json:"update_available"`
	CheckDisabled   bool   `json:"check_disabled"`
	CheckFailed     bool   `json:"check_failed"`
}

type AdGuardHomeStats struct {
	TimeUnits               string           `json:"time_units"`
	NumDNSQueries           uint64           `json:"num_dns_queries"`
	NumBlockedFiltering     uint64           `json:"num_blocked_filtering"`
	NumReplacedSafeBrowsing uint64           `json:"num_replaced_safebrowsing"`
	NumReplacedSafeSearch   uint64           `json:"num_replaced_safesearch"`
	NumReplacedParental     uint64           `json:"num_replaced_parental"`
	AvgProcessingTime       float64          `json:"avg_processing_time"`
	DNSQueries              []uint64         `json:"dns_queries"`
	BlockedFiltering        []uint64         `json:"blocked_filtering"`
	ReplacedSafeBrowsing    []uint64         `json:"replaced_safebrowsing"`
	ReplacedParental        []uint64         `json:"replaced_parental"`
	TopClients              []AdGuardTopItem `json:"top_clients"`
	TopQueriedDomains       []AdGuardTopItem `json:"top_queried_domains"`
	TopBlockedDomains       []AdGuardTopItem `json:"top_blocked_domains"`
}

type AdGuardHomeDashboard struct {
	ServiceKey string             `json:"service_key"`
	Status     AdGuardHomeStatus  `json:"status"`
	Version    AdGuardHomeVersion `json:"version"`
	Stats      AdGuardHomeStats   `json:"stats"`
}

type adGuardStatusResponse struct {
	ProtectionEnabled bool   `json:"protection_enabled"`
	Running           bool   `json:"running"`
	Version           string `json:"version"`
}

type adGuardStatsResponse struct {
	TimeUnits               string              `json:"time_units"`
	NumDNSQueries           uint64              `json:"num_dns_queries"`
	NumBlockedFiltering     uint64              `json:"num_blocked_filtering"`
	NumReplacedSafeBrowsing uint64              `json:"num_replaced_safebrowsing"`
	NumReplacedSafeSearch   uint64              `json:"num_replaced_safesearch"`
	NumReplacedParental     uint64              `json:"num_replaced_parental"`
	AvgProcessingTime       float64             `json:"avg_processing_time"`
	DNSQueries              []uint64            `json:"dns_queries"`
	BlockedFiltering        []uint64            `json:"blocked_filtering"`
	ReplacedSafeBrowsing    []uint64            `json:"replaced_safebrowsing"`
	ReplacedParental        []uint64            `json:"replaced_parental"`
	TopClients              []map[string]uint64 `json:"top_clients"`
	TopQueriedDomains       []map[string]uint64 `json:"top_queried_domains"`
	TopBlockedDomains       []map[string]uint64 `json:"top_blocked_domains"`
}

type adGuardVersionResponse struct {
	Disabled        bool   `json:"disabled"`
	NewVersion      string `json:"new_version"`
	Announcement    string `json:"announcement"`
	AnnouncementURL string `json:"announcement_url"`
	CanAutoUpdate   bool   `json:"can_autoupdate"`
}

func (s adGuardHomeService) Definition() Definition {
	return Definition{
		Key:             "adguardhome",
		DisplayName:     "AdGuard Home",
		Icon:            "/adguard-home.svg",
		RequiresBaseURL: true,
		AuthType:        AuthTypeBasicAuth,
		Endpoints: []string{
			"/services/adguardhome/stats",
		},
	}
}

func (s adGuardHomeService) TestConnection(ctx context.Context, cfg TestConfig) (int, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return 0, fmt.Errorf("base URL is required")
	}
	if strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.Password) == "" {
		return 0, fmt.Errorf("username and password are required")
	}

	status, err := doGET(ctx, strings.TrimRight(cfg.BaseURL, "/")+"/control/status", nil, cfg.Username, cfg.Password)
	if err != nil {
		return 0, err
	}
	if status < 200 || status >= 300 {
		return status, fmt.Errorf("adguard home test failed with status %d", status)
	}

	return status, nil
}

func (s adGuardHomeService) FetchDashboard(ctx context.Context, cfg TestConfig) (any, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.Password) == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")

	var statusResp adGuardStatusResponse
	_, err := doJSONRequest(
		ctx,
		http.MethodGet,
		baseURL+"/control/status",
		nil,
		nil,
		cfg.Username,
		cfg.Password,
		&statusResp,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch status: %w", err)
	}

	var statsResp adGuardStatsResponse
	_, err = doJSONRequest(
		ctx,
		http.MethodGet,
		baseURL+"/control/stats",
		nil,
		nil,
		cfg.Username,
		cfg.Password,
		&statsResp,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch stats: %w", err)
	}

	version := AdGuardHomeVersion{CurrentVersion: statusResp.Version}
	var versionResp adGuardVersionResponse
	_, err = doJSONRequest(
		ctx,
		http.MethodPost,
		baseURL+"/control/version.json",
		map[string]bool{"recheck_now": false},
		nil,
		cfg.Username,
		cfg.Password,
		&versionResp,
	)
	if err != nil {
		version.CheckFailed = true
	} else {
		version.CheckDisabled = versionResp.Disabled
		version.LatestVersion = versionResp.NewVersion
		version.Announcement = versionResp.Announcement
		version.AnnouncementURL = versionResp.AnnouncementURL
		version.CanAutoUpdate = versionResp.CanAutoUpdate
		version.UpdateAvailable = versionResp.NewVersion != "" && versionResp.NewVersion != statusResp.Version
	}

	return AdGuardHomeDashboard{
		ServiceKey: "adguardhome",
		Status: AdGuardHomeStatus{
			ProtectionEnabled: statusResp.ProtectionEnabled,
			Running:           statusResp.Running,
		},
		Version: version,
		Stats: AdGuardHomeStats{
			TimeUnits:               statsResp.TimeUnits,
			NumDNSQueries:           statsResp.NumDNSQueries,
			NumBlockedFiltering:     statsResp.NumBlockedFiltering,
			NumReplacedSafeBrowsing: statsResp.NumReplacedSafeBrowsing,
			NumReplacedSafeSearch:   statsResp.NumReplacedSafeSearch,
			NumReplacedParental:     statsResp.NumReplacedParental,
			AvgProcessingTime:       statsResp.AvgProcessingTime,
			DNSQueries:              statsResp.DNSQueries,
			BlockedFiltering:        statsResp.BlockedFiltering,
			ReplacedSafeBrowsing:    statsResp.ReplacedSafeBrowsing,
			ReplacedParental:        statsResp.ReplacedParental,
			TopClients:              normalizeAdGuardTopItems(statsResp.TopClients),
			TopQueriedDomains:       normalizeAdGuardTopItems(statsResp.TopQueriedDomains),
			TopBlockedDomains:       normalizeAdGuardTopItems(statsResp.TopBlockedDomains),
		},
	}, nil
}

func normalizeAdGuardTopItems(entries []map[string]uint64) []AdGuardTopItem {
	items := make([]AdGuardTopItem, 0, len(entries))
	for _, entry := range entries {
		for name, count := range entry {
			items = append(items, AdGuardTopItem{Name: name, Count: count})
		}
	}

	return items
}
