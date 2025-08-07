package entities_test

import (
	"testing"

	"suno-wallets/src/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name        string
		user        *entities.User
		expectError bool
		errorMsg    string
	}{
		{
			name: "Usuário válido",
			user: &entities.User{
				TenantID: uuid.New(),
				Name:     "João Silva",
				Email:    "joao@example.com",
				Phone:    "+5511999999999",
				Status:   entities.UserStatusActive,
			},
			expectError: false,
		},
		{
			name: "TenantID obrigatório",
			user: &entities.User{
				TenantID: uuid.Nil,
				Name:     "João Silva",
				Email:    "joao@example.com",
			},
			expectError: true,
			errorMsg:    "tenant_id é obrigatório",
		},
		{
			name: "Nome obrigatório",
			user: &entities.User{
				TenantID: uuid.New(),
				Name:     "",
				Email:    "joao@example.com",
			},
			expectError: true,
			errorMsg:    "nome é obrigatório",
		},
		{
			name: "Email obrigatório",
			user: &entities.User{
				TenantID: uuid.New(),
				Name:     "João Silva",
				Email:    "",
			},
			expectError: true,
			errorMsg:    "email é obrigatório",
		},
		{
			name: "Email com formato inválido",
			user: &entities.User{
				TenantID: uuid.New(),
				Name:     "João Silva",
				Email:    "email-invalido",
			},
			expectError: true,
			errorMsg:    "formato de email inválido",
		},
		{
			name: "Nome muito longo",
			user: &entities.User{
				TenantID: uuid.New(),
				Name:     string(make([]rune, 256)), // 256 caracteres
				Email:    "joao@example.com",
			},
			expectError: true,
			errorMsg:    "nome não pode ter mais de 255 caracteres",
		},
		{
			name: "Email muito longo",
			user: &entities.User{
				TenantID: uuid.New(),
				Name:     "João Silva",
				Email:    string(make([]rune, 250)) + "@example.com", // > 255 caracteres
			},
			expectError: true,
			errorMsg:    "email não pode ter mais de 255 caracteres",
		},
		{
			name: "Telefone muito longo",
			user: &entities.User{
				TenantID: uuid.New(),
				Name:     "João Silva",
				Email:    "joao@example.com",
				Phone:    string(make([]rune, 21)), // 21 caracteres
			},
			expectError: true,
			errorMsg:    "telefone não pode ter mais de 20 caracteres",
		},
		{
			name: "Status inválido",
			user: &entities.User{
				TenantID: uuid.New(),
				Name:     "João Silva",
				Email:    "joao@example.com",
				Status:   "invalid_status",
			},
			expectError: true,
			errorMsg:    "status do usuário inválido",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   entities.UserStatus
		expected bool
	}{
		{"Usuário ativo", entities.UserStatusActive, true},
		{"Usuário inativo", entities.UserStatusInactive, false},
		{"Usuário bloqueado", entities.UserStatusBlocked, false},
		{"Usuário pendente", entities.UserStatusPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &entities.User{Status: tt.status}
			assert.Equal(t, tt.expected, user.IsActive())
		})
	}
}

func TestUser_CanPerformActions(t *testing.T) {
	tests := []struct {
		name     string
		status   entities.UserStatus
		expected bool
	}{
		{"Usuário ativo pode realizar ações", entities.UserStatusActive, true},
		{"Usuário inativo não pode realizar ações", entities.UserStatusInactive, false},
		{"Usuário bloqueado não pode realizar ações", entities.UserStatusBlocked, false},
		{"Usuário pendente não pode realizar ações", entities.UserStatusPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &entities.User{Status: tt.status}
			assert.Equal(t, tt.expected, user.CanPerformActions())
		})
	}
}

func TestUser_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		user     *entities.User
		expected string
	}{
		{
			name: "Retorna nome quando disponível",
			user: &entities.User{
				Name:  "João Silva",
				Email: "joao@example.com",
			},
			expected: "João Silva",
		},
		{
			name: "Retorna email quando nome vazio",
			user: &entities.User{
				Name:  "",
				Email: "joao@example.com",
			},
			expected: "joao@example.com",
		},
		{
			name: "Retorna nome mesmo quando ambos disponíveis",
			user: &entities.User{
				Name:  "João Silva",
				Email: "joao@example.com",
			},
			expected: "João Silva",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.user.GetDisplayName())
		})
	}
}

func TestUser_TableName(t *testing.T) {
	user := &entities.User{}
	assert.Equal(t, "users", user.TableName())
}

// BenchmarkUser_Validate benchmark para validação de usuário
func BenchmarkUser_Validate(b *testing.B) {
	user := &entities.User{
		TenantID: uuid.New(),
		Name:     "João Silva",
		Email:    "joao@example.com",
		Phone:    "+5511999999999",
		Status:   entities.UserStatusActive,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = user.Validate()
	}
}
