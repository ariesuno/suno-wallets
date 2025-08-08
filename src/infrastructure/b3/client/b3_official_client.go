package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	b3cfg "suno-wallets/src/infrastructure/b3/config"
	b3err "suno-wallets/src/infrastructure/b3/errors"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"

	"golang.org/x/crypto/pkcs12"
)

// B3OfficialClient é um cliente HTTP com mTLS e OAuth2
// Comentários em pt-BR conforme diretrizes.
type B3OfficialClient struct {
	httpClient *http.Client
	cfg        *b3cfg.B3Config

	// tokenProvider opcional conforme fluxo selecionado (client credentials ou PKCE)
	getBearer func(ctx context.Context) (string, error)
}

// NewB3OfficialClient cria cliente com mTLS (p12) e timeout
func NewB3OfficialClient(cfg *b3cfg.B3Config, getBearer func(context.Context) (string, error)) (*B3OfficialClient, error) {
	tlsConfig, err := buildTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", b3err.ErrTLSConfig, err)
	}

	baseTransport := &http.Transport{TLSClientConfig: tlsConfig}
	var transport http.RoundTripper = baseTransport
	if cfg.CustomTransport != nil {
		transport = cfg.CustomTransport
	}
	c := &http.Client{Transport: transport, Timeout: cfg.Timeout}

	return &B3OfficialClient{httpClient: c, cfg: cfg, getBearer: getBearer}, nil
}

// buildTLSConfig carrega .p12 e monta tls.Config (TLS1.2+)
func buildTLSConfig(cfg *b3cfg.B3Config) (*tls.Config, error) {
	p12Bytes, err := os.ReadFile(cfg.CertP12Path)
	if err != nil {
		return nil, err
	}
	// converte p12 para chave e certificado
	privateKey, certificate, err := pkcs12.Decode(p12Bytes, cfg.CertPassphrase)
	if err != nil {
		return nil, err
	}

	// monta certificado x509 (sem cadeia adicional)
	cert := tls.Certificate{PrivateKey: privateKey, Certificate: [][]byte{certificate.Raw}}

	// opcional: legacy CA extra (.cer)
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if cfg.LegacyCertPath != "" {
		if b, e := os.ReadFile(cfg.LegacyCertPath); e == nil {
			if block, _ := pem.Decode(b); block != nil {
				if certParsed, e2 := x509.ParseCertificate(block.Bytes); e2 == nil {
					rootCAs.AddCert(certParsed)
				}
			} else {
				rootCAs.AppendCertsFromPEM(b)
			}
		}
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		RootCAs:      rootCAs,
	}, nil
}

// makeRequest aplica headers, auth, retry/backoff e métricas
func (c *B3OfficialClient) MakeRequest(ctx context.Context, method, path string, query map[string]string, cpf string, needsAuth bool) (*http.Response, error) {
	// validar CPF (11 dígitos)
	if cpf == "" || len(cpf) != 11 {
		return nil, b3err.ErrInvalidCPF
	}

	// montar URL
	url := strings.TrimRight(c.cfg.URLData, "/") + path
	if len(query) > 0 {
		parts := make([]string, 0, len(query))
		for k, v := range query {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
		url = url + "?" + strings.Join(parts, "&")
	}

	retries := 0
	backoff := c.cfg.InitialBackoff
	if backoff <= 0 {
		backoff = 200 * time.Millisecond
	}
	maxRetries := c.cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	start := time.Now()
	var resp *http.Response
	var err error
	statusLabel := "success"

	for {
		// montar request
		req, e := http.NewRequestWithContext(ctx, method, url, nil)
		if e != nil {
			statusLabel = "client_error"
			err = fmt.Errorf("%w: %v", b3err.ErrHTTPBuildRequest, e)
			break
		}

		// headers obrigatórios
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Investor-CPF", cpf)
		if needsAuth && c.getBearer != nil {
			token, e2 := c.getBearer(ctx)
			if e2 != nil {
				statusLabel = "auth_error"
				err = fmt.Errorf("%w: %v", b3err.ErrTokenAcquisition, e2)
				break
			}
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			statusLabel = "network_error"
		} else if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			// retry/backoff exponencial simples
			retries++
			observability.IncB3Retries(path)
			if retries <= maxRetries {
				// jitter simples: soma até 50% do backoff atual
				jitter := time.Duration(float64(backoff) * 0.5)
				time.Sleep(backoff + time.Duration(time.Now().UnixNano()%int64(jitter+1)))
				backoff *= 2
				if c.cfg.MaxBackoff > 0 && backoff > c.cfg.MaxBackoff {
					backoff = c.cfg.MaxBackoff
				}
				continue
			}
		} else if resp.StatusCode >= 400 {
			// erros não-retriáveis (400/401/403 etc.)
			statusLabel = fmt.Sprintf("http_%d", resp.StatusCode)
		}
		break
	}

	// métricas e logs
	observability.ObserveExternalAPI("b3", path, statusLabel, start)
	tenantID := ""
	// opcional: recuperar tenant do contexto
	tenantID = helpersTenant(ctx)
	helpers.LogInfo("B3 request", map[string]interface{}{
		"endpoint": path,
		"httpStatus": func() int {
			if resp != nil {
				return resp.StatusCode
			}
			return 0
		}(),
		"durationMs": time.Since(start).Milliseconds(),
		"retries":    retries,
		"tenant_id":  tenantID,
		"cpf_masked": maskCPF(cpf),
	})

	if err != nil {
		return nil, err
	}
	if resp != nil && resp.StatusCode >= 400 {
		return nil, b3err.NewB3Error(resp.StatusCode, "b3 http error")
	}
	return resp, nil
}

// maskCPF mascara CPF mantendo apenas últimos 2 dígitos
func maskCPF(cpf string) string {
	if len(cpf) != 11 {
		return "invalid"
	}
	return "*********" + cpf[9:]
}

// helpersTenant tenta extrair tenant_id do contexto (middleware pode setar)
func helpersTenant(ctx context.Context) string {
	// evitar dependência direta cruzada: replicar lógica mínima
	if v := ctx.Value("tenant_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
