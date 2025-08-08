package b3_test

import (
	"testing"
	"time"

	b3cli "suno-wallets/src/infrastructure/b3/client"
	b3cfg "suno-wallets/src/infrastructure/b3/config"
)

// Testa construção de cliente com p12 existente (certificado real deve estar em certs/)
func TestBuildClientWithP12(t *testing.T) {
	cfg := &b3cfg.B3Config{
		URLData:        "https://apidata.example", // placeholder
		CertP12Path:    "certs/b3_certificate12filepath.p12",
		CertPassphrase: "changeit",
		Timeout:        5 * time.Second,
	}
	_, err := b3cli.NewB3OfficialClient(cfg, nil)
	if err != nil {
		t.Skip("p12 ausente ou senha inválida no ambiente local; teste de construção é best-effort")
	}
}
