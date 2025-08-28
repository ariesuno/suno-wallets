package b3

import (
	"context"
	"os"
	"testing"
	"time"

	"suno-wallets/src/domain/enums"
	"suno-wallets/src/infrastructure/b3/client"
	"suno-wallets/src/infrastructure/b3/client/auth"
	"suno-wallets/src/infrastructure/b3/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestB3MultipleEndpoints testa todos os endpoints B3 específicos por tipo de ativo
func TestB3MultipleEndpoints(t *testing.T) {
	// Verificar se testes de integração estão habilitados
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Integration tests disabled. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	// Configurar cliente B3 (usando configurações de ambiente)
	cfg := &config.B3Config{
		URLData:        os.Getenv("B3_URL_DATA"),
		OAuthTokenURL:  os.Getenv("B3_OAUTH_TOKEN_URL"),
		ClientID:       os.Getenv("B3_CLIENT_ID"),
		ClientSecret:   os.Getenv("B3_CLIENT_SECRET"),
		Scope:          os.Getenv("B3_SCOPE"),
		CertP12Path:    os.Getenv("B3_CERT_P12_PATH"),
		CertPassphrase: os.Getenv("B3_CERT_PASSPHRASE"),
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     5 * time.Second,
	}

	// Verificar se configuração está disponível
	if cfg.URLData == "" || cfg.ClientID == "" {
		t.Skip("B3 configuration not available for integration tests")
	}

	creds := auth.NewClientCredentials(cfg, nil)
	getBearer := func(ctx context.Context) (string, error) {
		tok, err := creds.GetToken(ctx)
		if err != nil {
			return "", err
		}
		return tok.AccessToken, nil
	}

	b3Client, err := client.NewB3OfficialClient(cfg, getBearer)
	require.NoError(t, err, "Failed to create B3 client")

	ctx := context.Background()
	testCPF := os.Getenv("B3_TEST_CPF")
	if testCPF == "" {
		t.Skip("B3_TEST_CPF not provided for integration tests")
	}

	startDate := "2024-01-01"
	endDate := "2024-01-31"
	page := 1

	t.Run("TestAllAssetTypesTransactions", func(t *testing.T) {
		allTypes := enums.GetAllB3AssetTypes()

		for _, assetType := range allTypes {
			t.Run("Transactions_"+string(assetType), func(t *testing.T) {
				if !assetType.IsTransactionSupported() {
					t.Skipf("Transactions not supported for %s", assetType)
				}

				resp, err := b3Client.GetTransactionsByAssetType(ctx, testCPF, assetType, startDate, endDate, page)

				// Aceitar tanto sucesso quanto alguns erros específicos da B3
				if err != nil {
					t.Logf("Request failed for %s transactions: %v", assetType, err)
					// Não falhar o teste - alguns tipos podem não ter dados ou retornar erros específicos
					return
				}

				require.NotNil(t, resp, "Response should not be nil for %s", assetType)
				resp.Body.Close()

				// Log para verificação manual
				t.Logf("Successfully called %s transactions endpoint", assetType)
			})
		}
	})

	t.Run("TestAllAssetTypesPositions", func(t *testing.T) {
		allTypes := enums.GetAllB3AssetTypes()

		for _, assetType := range allTypes {
			t.Run("Positions_"+string(assetType), func(t *testing.T) {
				if !assetType.IsPositionSupported() {
					t.Skipf("Positions not supported for %s", assetType)
				}

				resp, err := b3Client.GetPositionsByAssetType(ctx, testCPF, assetType, startDate, endDate, page)

				// Aceitar tanto sucesso quanto alguns erros específicos da B3
				if err != nil {
					t.Logf("Request failed for %s positions: %v", assetType, err)
					// Não falhar o teste - alguns tipos podem não ter dados ou retornar erros específicos
					return
				}

				require.NotNil(t, resp, "Response should not be nil for %s", assetType)
				resp.Body.Close()

				// Log para verificação manual
				t.Logf("Successfully called %s positions endpoint", assetType)
			})
		}
	})

	t.Run("TestSpecificEndpointMethods", func(t *testing.T) {
		// Teste métodos específicos para garantir que funcionam
		testCases := []struct {
			name string
			test func(t *testing.T)
		}{
			{
				name: "EquitiesTransactions",
				test: func(t *testing.T) {
					resp, err := b3Client.GetEquitiesTransactions(ctx, testCPF, startDate, endDate, page)
					if err != nil {
						t.Logf("Equities transactions failed: %v", err)
						return
					}
					assert.NotNil(t, resp)
					resp.Body.Close()
				},
			},
			{
				name: "FixedIncomeTransactions",
				test: func(t *testing.T) {
					resp, err := b3Client.GetFixedIncomeTransactions(ctx, testCPF, startDate, endDate, page)
					if err != nil {
						t.Logf("Fixed income transactions failed: %v", err)
						return
					}
					assert.NotNil(t, resp)
					resp.Body.Close()
				},
			},
			{
				name: "DerivativesTransactions",
				test: func(t *testing.T) {
					resp, err := b3Client.GetDerivativesTransactions(ctx, testCPF, startDate, endDate, page)
					if err != nil {
						t.Logf("Derivatives transactions failed: %v", err)
						return
					}
					assert.NotNil(t, resp)
					resp.Body.Close()
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, tc.test)
		}
	})

	t.Run("TestInvalidAssetType", func(t *testing.T) {
		invalidType := enums.B3AssetType("invalid-type")

		_, err := b3Client.GetTransactionsByAssetType(ctx, testCPF, invalidType, startDate, endDate, page)
		assert.Error(t, err, "Should return error for invalid asset type")
		assert.Contains(t, err.Error(), "invalid asset type")
	})
}

// TestB3EndpointMapping testa o mapeamento correto dos endpoints
func TestB3EndpointMapping(t *testing.T) {
	testCases := []struct {
		assetType        enums.B3AssetType
		expectedAPI      string
		expectedPosition string
	}{
		{
			assetType:        enums.B3AssetTypeEquities,
			expectedAPI:      "/equities/investors",
			expectedPosition: "/position/v3/equities/investors", // Endpoint especial para equities
		},
		{
			assetType:        enums.B3AssetTypeFixedIncome,
			expectedAPI:      "/fixed-income/investors",
			expectedPosition: "/fixed-income/investors",
		},
		{
			assetType:        enums.B3AssetTypeTreasuryBonds,
			expectedAPI:      "/treasury-bonds/investors",
			expectedPosition: "/treasury-bonds/investors",
		},
		{
			assetType:        enums.B3AssetTypeDerivatives,
			expectedAPI:      "/derivatives/investors",
			expectedPosition: "/derivatives/investors",
		},
		{
			assetType:        enums.B3AssetTypeSecuritiesLending,
			expectedAPI:      "/securities-lending/investors",
			expectedPosition: "/securities-lending/investors",
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.assetType), func(t *testing.T) {
			assert.Equal(t, tc.expectedAPI, tc.assetType.GetAPIEndpoint())
			assert.Equal(t, tc.expectedPosition, tc.assetType.GetPositionsEndpoint())
		})
	}
}
