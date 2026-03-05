package integrations

import (
	"context"
	"fmt"
	"strings"
)

type jellyfinService struct{}

func (s jellyfinService) Definition() Definition {
	return Definition{
		Key:             "jellyfin",
		DisplayName:     "Jellyfin",
		Icon:            "/jellyfin.svg",
		RequiresBaseURL: true,
		AuthType:        AuthTypeToken,
		Endpoints: []string{
			"/services/jellyfin/recents",
		},
	}
}

func (s jellyfinService) TestConnection(ctx context.Context, cfg TestConfig) (int, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return 0, fmt.Errorf("base URL is required")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return 0, fmt.Errorf("token is required")
	}

	headers := map[string]string{
		"X-Emby-Token": cfg.Token,
	}

	status, err := doGET(ctx, strings.TrimRight(cfg.BaseURL, "/")+"/System/Info/Public", headers, "", "")
	if err != nil {
		return 0, err
	}
	if status < 200 || status >= 300 {
		return status, fmt.Errorf("jellyfin test failed with status %d", status)
	}

	return status, nil
}
