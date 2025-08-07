package entities_test

import (
	"testing"

	"suno-wallets/src/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWallet_Validate(t *testing.T) {
	tests := []struct {
		name        string
		wallet      *entities.Wallet
		expectError bool
		errorMsg    string
	}{
        {
            name: "Valid wallet",
			wallet: &entities.Wallet{
				TenantID:  uuid.New(),
				Name:      "Minha Carteira",
				Type:      entities.WalletTypePersonal,
				Status:    entities.WalletStatusActive,
				Currency:  "BRL",
				OwnerID:   uuid.New(),
				OwnerType: "user",
			},
			expectError: false,
		},
        {
            name: "TenantID required",
			wallet: &entities.Wallet{
				TenantID: uuid.Nil,
				Name:     "Carteira Teste",
			},
			expectError: true,
			errorMsg:    "tenant_id é obrigatório",
		},
        {
            name: "Name required",
			wallet: &entities.Wallet{
				TenantID: uuid.New(),
				Name:     "",
				OwnerID:  uuid.New(),
			},
			expectError: true,
			errorMsg:    "nome da carteira é obrigatório",
		},
        {
            name: "OwnerID required",
			wallet: &entities.Wallet{
				TenantID: uuid.New(),
				Name:     "Carteira Teste",
				OwnerID:  uuid.Nil,
			},
			expectError: true,
			errorMsg:    "owner_id é obrigatório",
		},
        {
            name: "Invalid wallet type",
			wallet: &entities.Wallet{
				TenantID: uuid.New(),
				Name:     "Carteira Teste",
				Type:     "invalid_type",
				OwnerID:  uuid.New(),
			},
			expectError: true,
			errorMsg:    "tipo de carteira inválido",
		},
        {
            name: "Invalid status",
			wallet: &entities.Wallet{
				TenantID: uuid.New(),
				Name:     "Carteira Teste",
				Type:     entities.WalletTypePersonal,
				Status:   "invalid_status",
				OwnerID:  uuid.New(),
			},
			expectError: true,
			errorMsg:    "status da carteira inválido",
		},
        {
            name: "Invalid currency format",
			wallet: &entities.Wallet{
				TenantID: uuid.New(),
				Name:     "Carteira Teste",
				Type:     entities.WalletTypePersonal,
				Status:   entities.WalletStatusActive,
				Currency: "BRLL", // 4 caracteres
				OwnerID:  uuid.New(),
			},
			expectError: true,
			errorMsg:    "moeda deve ter exatamente 3 caracteres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.wallet.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				// Verificar se o campo Currency foi definido como padrão
				if tt.wallet.Currency == "" {
					assert.Equal(t, "BRL", tt.wallet.Currency)
				}
			}
		})
	}
}

func TestWallet_IsActive(t *testing.T) {
    tests := []struct {
		name     string
		status   entities.WalletStatus
		expected bool
	}{
        {"Active wallet", entities.WalletStatusActive, true},
        {"Inactive wallet", entities.WalletStatusInactive, false},
        {"Blocked wallet", entities.WalletStatusBlocked, false},
        {"Suspended wallet", entities.WalletStatusSuspended, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wallet := &entities.Wallet{Status: tt.status}
			assert.Equal(t, tt.expected, wallet.IsActive())
		})
	}
}

func TestWallet_CanDebit(t *testing.T) {
    tests := []struct {
		name       string
		status     entities.WalletStatus
		allowDebit bool
		expected   bool
	}{
        {"Active wallet with debit allowed", entities.WalletStatusActive, true, true},
        {"Active wallet with debit not allowed", entities.WalletStatusActive, false, false},
        {"Inactive wallet with debit allowed", entities.WalletStatusInactive, true, false},
        {"Blocked wallet with debit allowed", entities.WalletStatusBlocked, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wallet := &entities.Wallet{
				Status:     tt.status,
				AllowDebit: tt.allowDebit,
			}
			assert.Equal(t, tt.expected, wallet.CanDebit())
		})
	}
}

func TestWallet_HasSufficientBalance(t *testing.T) {
	wallet := &entities.Wallet{Balance: 10000} // R$ 100,00 em centavos

    tests := []struct {
		name     string
		amount   int64
		expected bool
	}{
        {"Sufficient balance", 5000, true},
        {"Exact balance", 10000, true},
        {"Insufficient balance", 15000, false},
        {"Zero amount", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, wallet.HasSufficientBalance(tt.amount))
		})
	}
}

func TestWallet_GetBalanceInReais(t *testing.T) {
    tests := []struct {
		name            string
		balanceInCents  int64
		expectedInReais float64
	}{
        {"BRL 100.00", 10000, 100.00},
        {"BRL 50.50", 5050, 50.50},
        {"BRL 0.01", 1, 0.01},
        {"BRL 0.00", 0, 0.00},
        {"BRL 1,234.56", 123456, 1234.56},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wallet := &entities.Wallet{Balance: tt.balanceInCents}
			assert.Equal(t, tt.expectedInReais, wallet.GetBalanceInReais())
		})
	}
}

func TestWallet_GetLimitInReais(t *testing.T) {
    t.Run("Daily limit defined", func(t *testing.T) {
		limit := int64(50000) // R$ 500,00
		wallet := &entities.Wallet{DailyLimit: &limit}

		result := wallet.GetDailyLimitInReais()
		require.NotNil(t, result)
		assert.Equal(t, 500.00, *result)
	})

    t.Run("Daily limit not defined", func(t *testing.T) {
		wallet := &entities.Wallet{DailyLimit: nil}

		result := wallet.GetDailyLimitInReais()
		assert.Nil(t, result)
	})

    t.Run("Monthly limit defined", func(t *testing.T) {
		limit := int64(200000) // R$ 2.000,00
		wallet := &entities.Wallet{MonthlyLimit: &limit}

		result := wallet.GetMonthlyLimitInReais()
		require.NotNil(t, result)
		assert.Equal(t, 2000.00, *result)
	})

    t.Run("Monthly limit not defined", func(t *testing.T) {
		wallet := &entities.Wallet{MonthlyLimit: nil}

		result := wallet.GetMonthlyLimitInReais()
		assert.Nil(t, result)
	})
}

// BenchmarkWallet_Validate benchmark para validação de carteira
func BenchmarkWallet_Validate(b *testing.B) {
	wallet := &entities.Wallet{
		TenantID:  uuid.New(),
		Name:      "Carteira Benchmark",
		Type:      entities.WalletTypePersonal,
		Status:    entities.WalletStatusActive,
		Currency:  "BRL",
		OwnerID:   uuid.New(),
		OwnerType: "user",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wallet.Validate()
	}
}
