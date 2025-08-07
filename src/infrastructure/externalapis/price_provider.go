package externalapis

import (
	"context"
	"errors"
	"net/http"
	"time"
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

	req, err := http.NewRequestWithContext(cctx, http.MethodGet, p.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("external service unhealthy")
	}
	return nil
}
