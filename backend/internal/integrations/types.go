package integrations

import "context"

type AuthType string

const (
	AuthTypeToken     AuthType = "token"
	AuthTypeBasicAuth AuthType = "basic_auth"
)

type Definition struct {
	Key             string   `json:"key"`
	DisplayName     string   `json:"display_name"`
	Icon            string   `json:"icon"`
	RequiresBaseURL bool     `json:"requires_base_url"`
	AuthType        AuthType `json:"auth_type"`
	FixedBaseURL    string   `json:"fixed_base_url,omitempty"`
	Endpoints       []string `json:"endpoints"`
}

type TestConfig struct {
	BaseURL  string
	Token    string
	Username string
	Password string
}

type Service interface {
	Definition() Definition
	TestConnection(ctx context.Context, cfg TestConfig) (int, error)
}

type DashboardProvider interface {
	FetchDashboard(ctx context.Context, cfg TestConfig) (any, error)
}
