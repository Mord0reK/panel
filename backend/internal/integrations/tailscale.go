package integrations

import (
	"context"
	"fmt"
	"strings"
)

type tailscaleService struct{}

func (s tailscaleService) Definition() Definition {
	return Definition{
		Key:             "tailscale",
		DisplayName:     "Tailscale",
		Icon:            "/icons/tailscale.svg",
		RequiresBaseURL: false,
		AuthType:        AuthTypeToken,
		FixedBaseURL:    "https://api.tailscale.com/api/v2",
		Endpoints: []string{
			"/services/tailscale/stats",
		},
	}
}

func (s tailscaleService) TestConnection(ctx context.Context, cfg TestConfig) (int, error) {
	if strings.TrimSpace(cfg.Token) == "" {
		return 0, fmt.Errorf("token is required")
	}

	baseURL := s.Definition().FixedBaseURL
	status, err := doGET(ctx, strings.TrimRight(baseURL, "/")+"/tailnet/-/devices", nil, cfg.Token, "")
	if err != nil {
		return 0, err
	}
	if status < 200 || status >= 300 {
		return status, fmt.Errorf("tailscale test failed with status %d", status)
	}

	return status, nil
}
