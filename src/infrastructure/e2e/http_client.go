package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Comentários em pt-BR: HTTP client centralizado com timeouts, retries básicos e headers padrão

type HTTPClient struct {
	baseURL     string
	tenantID    string
	bearerToken string
	httpClient  *http.Client
}

func NewHTTPClient(baseURL, tenantID, bearer string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL:     baseURL,
		tenantID:    tenantID,
		bearerToken: bearer,
		httpClient:  &http.Client{Timeout: timeout},
	}
}

func (c *HTTPClient) withHeaders(req *http.Request, correlationID string) {
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-Id", c.tenantID)
	}
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
	if correlationID != "" {
		req.Header.Set("X-Correlation-Id", correlationID)
	}
	req.Header.Set("Content-Type", "application/json")
}

func (c *HTTPClient) do(ctx context.Context, method, path string, payload any, correlationID string) (int, []byte, error) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return 0, nil, err
	}
	c.withHeaders(req, correlationID)

	// retries simples para 429/5xx
	backoff := 200 * time.Millisecond
	for attempt := 0; attempt < 4; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt == 3 {
				return 0, nil, err
			}
		} else {
			defer func() { _ = resp.Body.Close() }()
			data, _ := io.ReadAll(resp.Body)
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
				time.Sleep(backoff)
				backoff *= 2
			} else {
				return resp.StatusCode, data, nil
			}
		}
	}
	return 0, nil, fmt.Errorf("unreachable")
}

func (c *HTTPClient) Get(ctx context.Context, path string, correlationID string) (int, []byte, error) {
	return c.do(ctx, http.MethodGet, path, nil, correlationID)
}
func (c *HTTPClient) PostJSON(ctx context.Context, path string, body any, correlationID string) (int, []byte, error) {
	return c.do(ctx, http.MethodPost, path, body, correlationID)
}
