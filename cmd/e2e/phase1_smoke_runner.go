package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	phase1 "suno-wallets/src/application/e2e/phase1"
	e2e "suno-wallets/src/infrastructure/e2e"
)

// Comentários em pt-BR: CLI do runner de smoke da Fase 1

func main() {
	// Flags
	tenant := flag.String("tenant", os.Getenv("TEST_TENANT_ID"), "Tenant ID")
	cpf := flag.String("cpf", os.Getenv("TEST_CPF"), "CPF (11 dígitos)")
	baseURL := flag.String("base-url", os.Getenv("API_BASE_URL"), "Base URL da API")
	token := flag.String("token", os.Getenv("AUTH_BEARER"), "Bearer token")
	allowReset := flag.Bool("allow-reset", os.Getenv("ALLOW_DESTRUCTIVE_RESET") == "true", "Permite reset real")
	flag.Parse()

	// Pré-flight
	if *tenant == "" || *cpf == "" || *baseURL == "" || *token == "" {
		log.Fatal("missing required env/flags: TEST_TENANT_ID, TEST_CPF, API_BASE_URL, AUTH_BEARER")
	}

	httpc := e2e.NewHTTPClient(*baseURL, *tenant, *token, 15*time.Second)
	orch := phase1.NewOrchestrator(httpc)
	if _, err := orch.Run(context.Background(), *tenant, *cpf, *baseURL, *allowReset); err != nil {
		log.Fatal(err)
	}
}
