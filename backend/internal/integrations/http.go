package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var serviceHTTPClient = &http.Client{Timeout: 10 * time.Second}

func doJSONRequest(
	ctx context.Context,
	method string,
	url string,
	body any,
	headers map[string]string,
	basicUser string,
	basicPassword string,
	target any,
) (int, error) {
	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if basicUser != "" {
		req.SetBasicAuth(basicUser, basicPassword)
	}

	resp, err := serviceHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(responseBody))
	}

	if target == nil || len(responseBody) == 0 {
		return resp.StatusCode, nil
	}

	if err := json.Unmarshal(responseBody, target); err != nil {
		return resp.StatusCode, fmt.Errorf("decode response body: %w", err)
	}

	return resp.StatusCode, nil
}

func doGET(ctx context.Context, url string, headers map[string]string, basicUser, basicPassword string) (int, error) {
	return doJSONRequest(ctx, http.MethodGet, url, nil, headers, basicUser, basicPassword, nil)
}
