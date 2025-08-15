package externalapis

import (
	"context"
	"errors"
	"net/http"
	"time"

	"suno-wallets/src/infrastructure/observability"
)

// HTTPClient define interface para cliente HTTP (facilita mocks em testes)
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// PriceProvider serviço simples para obter preços externos (placeholder)
type PriceProvider struct {
	client  HTTPClient
	baseURL string
	timeout time.Duration
}

// NewPriceProvider constrói um PriceProvider com timeout
func NewPriceProvider(client HTTPClient, baseURL string, timeout time.Duration) *PriceProvider {
	return &PriceProvider{client: client, baseURL: baseURL, timeout: timeout}
}

// GetHealth verifica disponibilidade do serviço externo
func (p *PriceProvider) GetHealth(ctx context.Context) error {
	cctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	endpoint := "/health"
	start := time.Now()
	statusLabel := "success"
	defer func() {
		observability.ObserveExternalAPI("price_provider", endpoint, statusLabel, start)
	}()

	req, err := http.NewRequestWithContext(cctx, http.MethodGet, p.baseURL+endpoint, nil)
	if err != nil {
		statusLabel = "client_error"
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		statusLabel = "network_error"
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		statusLabel = "http_" + http.StatusText(resp.StatusCode)
		return errors.New("external service unhealthy")
	}
	return nil
}
