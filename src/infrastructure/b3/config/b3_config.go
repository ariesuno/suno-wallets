package config

import (
	"net/http"
	"time"
)

// B3Config representa configurações para o cliente oficial da B3
// Comentários em pt-BR conforme diretrizes.
type B3Config struct {
	URLData         string
	OAuthTokenURL   string
	ClientID        string
	ClientSecret    string
	Scope           string // OAuth2 scope para B3
	CertP12Path     string
	CertPassphrase  string
	LegacyCertPath  string // opcional: .cer legado (CA ou cliente)
	Timeout         time.Duration
	MaxRetries      int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	CustomTransport http.RoundTripper // opcional: para testes/mocks
}
