package b3_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	b3cli "suno-wallets/src/infrastructure/b3/client"
	b3cfg "suno-wallets/src/infrastructure/b3/config"
)

// fake client para não efetuar chamadas reais
type fakeRoundTripper struct{}

func (f *fakeRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200}, nil
}

func TestCPFValidation(t *testing.T) {
	cfg := &b3cfg.B3Config{URLData: "https://apidata.example", CertP12Path: "certs/b3_certificate12filepath.p12", CertPassphrase: "changeit", Timeout: 2 * time.Second}
	c, err := b3cli.NewB3OfficialClient(cfg, func(ctx context.Context) (string, error) { return "token", nil })
	if err != nil {
		t.Skip("p12 ausente; ignorando")
	}
	// substitui transport por fake
	_ = &http.Client{Transport: &fakeRoundTripper{}, Timeout: 2 * time.Second}
	// reflect set é indesejável; neste teste validamos apenas erro de CPF sem efetuar request
	_, err = c.MakeRequest(context.Background(), http.MethodGet, "/echo", nil, "123", true)
	if err == nil {
		t.Fatalf("esperava erro de CPF inválido")
	}
}
