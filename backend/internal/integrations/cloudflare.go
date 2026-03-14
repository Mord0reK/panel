package integrations

import (
	"context"
	"fmt"
	"strings"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
)

type CloudflareZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CloudflareDNSRecord struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	TTL        int    `json:"ttl"`
	Proxied    bool   `json:"proxied"`
	ModifiedOn string `json:"modified_on"`
}

type cloudflareService struct{}

func newCloudflareClient(apiToken string) *cloudflare.Client {
	return cloudflare.NewClient(option.WithAPIToken(apiToken))
}

func (s cloudflareService) Definition() Definition {
	return Definition{
		Key:             "cloudflare",
		DisplayName:     "Cloudflare",
		Icon:            "/cloudflare.svg",
		RequiresBaseURL: false,
		AuthType:        AuthTypeBasicAuth,
		FixedBaseURL:    "https://api.cloudflare.com/client/v4",
		Endpoints: []string{
			"/services/cloudflare/dns",
		},
	}
}

func (s cloudflareService) TestConnection(ctx context.Context, cfg TestConfig) (int, error) {
	if strings.TrimSpace(cfg.Username) == "" {
		return 0, fmt.Errorf("zone ID (username) is required")
	}
	if strings.TrimSpace(cfg.Password) == "" {
		return 0, fmt.Errorf("API token (password) is required")
	}

	baseURL := s.Definition().FixedBaseURL
	headers := map[string]string{
		"Authorization": "Bearer " + cfg.Password,
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

// FetchCloudflareZones returns all zones accessible with the given API token.
func FetchCloudflareZones(ctx context.Context, apiToken string) ([]CloudflareZone, error) {
	client := newCloudflareClient(apiToken)

	pager := client.Zones.ListAutoPaging(ctx, zones.ZoneListParams{})

	var result []CloudflareZone
	for pager.Next() {
		z := pager.Current()
		result = append(result, CloudflareZone{
			ID:   z.ID,
			Name: z.Name,
		})
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("cloudflare list zones: %w", err)
	}

	return result, nil
}

// FetchCloudflareDNSRecords returns all DNS records for the given zone.
func FetchCloudflareDNSRecords(ctx context.Context, apiToken, zoneID string) ([]CloudflareDNSRecord, error) {
	client := newCloudflareClient(apiToken)

	pager := client.DNS.Records.ListAutoPaging(ctx, dns.RecordListParams{
		ZoneID: cloudflare.F(zoneID),
	})

	var result []CloudflareDNSRecord
	for pager.Next() {
		r := pager.Current()
		ttl := int(r.TTL)
		modifiedOn := ""
		if !r.ModifiedOn.IsZero() {
			modifiedOn = r.ModifiedOn.Format(time.RFC3339)
		}
		result = append(result, CloudflareDNSRecord{
			ID:         r.ID,
			Type:       string(r.Type),
			Name:       r.Name,
			Content:    r.Content,
			TTL:        ttl,
			Proxied:    r.Proxied,
			ModifiedOn: modifiedOn,
		})
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("cloudflare list dns records: %w", err)
	}

	return result, nil
}
