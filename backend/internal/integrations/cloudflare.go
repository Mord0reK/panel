package integrations

import (
	"context"
	"fmt"
	"strings"
)

type cloudflareService struct{}

func (s cloudflareService) Definition() Definition {
	return Definition{
		Key:             "cloudflare",
		DisplayName:     "Cloudflare",
		Icon:            "/cloudflare.svg",
		RequiresBaseURL: false,
		AuthType:        AuthTypeToken,
		FixedBaseURL:    "https://api.cloudflare.com/client/v4",
		Endpoints: []string{
			"/services/cloudflare/analytics",
		},
	}
}

func (s cloudflareService) TestConnection(ctx context.Context, cfg TestConfig) (int, error) {
	if strings.TrimSpace(cfg.Token) == "" {
		return 0, fmt.Errorf("token is required")
	}

	baseURL := s.Definition().FixedBaseURL
	headers := map[string]string{
		"Authorization": "Bearer " + cfg.Token,
	}

	status, err := doGET(ctx, strings.TrimRight(baseURL, "/")+"/user/tokens/verify", headers, "", "")
	if err != nil {
		return 0, err
	}
	if status < 200 || status >= 300 {
		return status, fmt.Errorf("cloudflare test failed with status %d", status)
	}

	return status, nil
}
