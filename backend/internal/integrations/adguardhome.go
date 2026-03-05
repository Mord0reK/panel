package integrations

import (
	"context"
	"fmt"
	"strings"
)

type adGuardHomeService struct{}

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
