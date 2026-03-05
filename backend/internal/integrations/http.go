package integrations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

var serviceHTTPClient = &http.Client{Timeout: 10 * time.Second}

func doGET(ctx context.Context, url string, headers map[string]string, basicUser, basicPassword string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if basicUser != "" {
		req.SetBasicAuth(basicUser, basicPassword)
	}

	resp, err := serviceHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))

	return resp.StatusCode, nil
}
