package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	b3cfg "suno-wallets/src/infrastructure/b3/config"
)

// Comentários em pt-BR: implementação simples de Client Credentials com cache/refresh
type clientCredentialsImpl struct {
	cfg    *b3cfg.B3Config
	client *http.Client

	mu          sync.Mutex
	cached      *TokenResponse
	refreshSkew time.Duration
}

func NewClientCredentials(cfg *b3cfg.B3Config, httpClient *http.Client) ClientCredentialsProvider {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}
	return &clientCredentialsImpl{cfg: cfg, client: httpClient, refreshSkew: 30 * time.Second}
}

func (c *clientCredentialsImpl) GetToken(ctx context.Context) (*TokenResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached != nil {
		if time.Since(c.cached.ObtainedAt) < time.Duration(c.cached.ExpiresIn)*time.Second-c.refreshSkew {
			return c.cached, nil
		}
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.OAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.cfg.ClientID, c.cfg.ClientSecret)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to acquire token")
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	c.cached = &TokenResponse{AccessToken: payload.AccessToken, ExpiresIn: payload.ExpiresIn, ObtainedAt: time.Now()}
	return c.cached, nil
}
