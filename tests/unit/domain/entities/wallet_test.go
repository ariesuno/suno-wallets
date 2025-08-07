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
			name: "Carteira válida",
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
			name: "TenantID obrigatório",
			wallet: &entities.Wallet{
				TenantID: uuid.Nil,
				Name:     "Carteira Teste",
			},
			expectError: true,
			errorMsg:    "tenant_id é obrigatório",
		},
		{
			name: "Nome obrigatório",
			wallet: &entities.Wallet{
				TenantID: uuid.New(),
				Name:     "",
				OwnerID:  uuid.New(),
			},
			expectError: true,
			errorMsg:    "nome da carteira é obrigatório",
		},
		{
			name: "OwnerID obrigatório",
			wallet: &entities.Wallet{
				TenantID: uuid.New(),
				Name:     "Carteira Teste",
				OwnerID:  uuid.Nil,
			},
			expectError: true,
			errorMsg:    "owner_id é obrigatório",
		},
		{
			name: "Tipo de carteira inválido",
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
			name: "Status inválido",
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
			name: "Moeda com formato incorreto",
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
		{"Carteira ativa", entities.WalletStatusActive, true},
		{"Carteira inativa", entities.WalletStatusInactive, false},
		{"Carteira bloqueada", entities.WalletStatusBlocked, false},
		{"Carteira suspensa", entities.WalletStatusSuspended, false},
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
		{"Carteira ativa com débito permitido", entities.WalletStatusActive, true, true},
		{"Carteira ativa com débito não permitido", entities.WalletStatusActive, false, false},
		{"Carteira inativa com débito permitido", entities.WalletStatusInactive, true, false},
		{"Carteira bloqueada com débito permitido", entities.WalletStatusBlocked, true, false},
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
		{"Saldo suficiente", 5000, true},
		{"Saldo exato", 10000, true},
		{"Saldo insuficiente", 15000, false},
		{"Valor zero", 0, true},
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
		{"R$ 100,00", 10000, 100.00},
		{"R$ 50,50", 5050, 50.50},
		{"R$ 0,01", 1, 0.01},
		{"R$ 0,00", 0, 0.00},
		{"R$ 1.234,56", 123456, 1234.56},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wallet := &entities.Wallet{Balance: tt.balanceInCents}
			assert.Equal(t, tt.expectedInReais, wallet.GetBalanceInReais())
		})
	}
}

func TestWallet_GetLimitInReais(t *testing.T) {
	t.Run("Limite diário definido", func(t *testing.T) {
		limit := int64(50000) // R$ 500,00
		wallet := &entities.Wallet{DailyLimit: &limit}

		result := wallet.GetDailyLimitInReais()
		require.NotNil(t, result)
		assert.Equal(t, 500.00, *result)
	})

	t.Run("Limite diário não definido", func(t *testing.T) {
		wallet := &entities.Wallet{DailyLimit: nil}

		result := wallet.GetDailyLimitInReais()
		assert.Nil(t, result)
	})

	t.Run("Limite mensal definido", func(t *testing.T) {
		limit := int64(200000) // R$ 2.000,00
		wallet := &entities.Wallet{MonthlyLimit: &limit}

		result := wallet.GetMonthlyLimitInReais()
		require.NotNil(t, result)
		assert.Equal(t, 2000.00, *result)
	})

	t.Run("Limite mensal não definido", func(t *testing.T) {
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
