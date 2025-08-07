package interfaces

import (
	"context"

	"suno-wallets/src/domain/entities"
	"suno-wallets/src/domain/enums"
	"suno-wallets/src/domain/valueobjects"

	"github.com/google/uuid"
)

// UserRepository define a interface para operações com usuários
type UserRepository interface {
	// Create cria um novo usuário
	Create(ctx context.Context, user *entities.User) error

	// GetByID busca um usuário por ID e tenant
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*entities.User, error)

	// GetByEmail busca um usuário por email e tenant
	GetByEmail(ctx context.Context, email *valueobjects.Email, tenantID uuid.UUID) (*entities.User, error)

	// GetByCPF busca um usuário por CPF e tenant
	GetByCPF(ctx context.Context, cpf *valueobjects.CPF, tenantID uuid.UUID) (*entities.User, error)

	// Update atualiza um usuário existente
	Update(ctx context.Context, user *entities.User) error

	// Delete remove um usuário (soft delete)
	Delete(ctx context.Context, id, tenantID uuid.UUID) error

	// List lista usuários com paginação e filtros
	List(ctx context.Context, params ListUserParams) ([]*entities.User, int64, error)

	// UpdateStatus atualiza o status de um usuário
	UpdateStatus(ctx context.Context, id, tenantID uuid.UUID, status enums.UserStatus) error

	// UpdatePassword atualiza a senha do usuário
	UpdatePassword(ctx context.Context, id, tenantID uuid.UUID, passwordHash string) error

	// UpdateLastLogin atualiza informações do último login
	UpdateLastLogin(ctx context.Context, id, tenantID uuid.UUID, ipAddress string) error

	// IncrementFailedLogin incrementa contador de tentativas falhadas
	IncrementFailedLogin(ctx context.Context, id, tenantID uuid.UUID) error

	// ResetFailedLogin reseta contador de tentativas falhadas
	ResetFailedLogin(ctx context.Context, id, tenantID uuid.UUID) error

	// ExistsByEmail verifica se já existe usuário com o email
	ExistsByEmail(ctx context.Context, email *valueobjects.Email, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error)

	// ExistsByCPF verifica se já existe usuário com o CPF
	ExistsByCPF(ctx context.Context, cpf *valueobjects.CPF, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error)

	// GetUsersWithExpiredLocks busca usuários com bloqueios expirados
	GetUsersWithExpiredLocks(ctx context.Context, tenantID uuid.UUID) ([]*entities.User, error)

	// GetUsersByRiskLevel busca usuários por nível de risco
	GetUsersByRiskLevel(ctx context.Context, tenantID uuid.UUID, riskLevel string) ([]*entities.User, error)

	// GetKYCPendingUsers busca usuários com KYC pendente
	GetKYCPendingUsers(ctx context.Context, tenantID uuid.UUID) ([]*entities.User, error)

	// UpdateTransactionLimits atualiza limites de transação do usuário
	UpdateTransactionLimits(ctx context.Context, id, tenantID uuid.UUID, daily, monthly, yearly *int64) error
}

// ListUserParams parâmetros para listagem de usuários
type ListUserParams struct {
	TenantID         uuid.UUID
	Status           *enums.UserStatus
	IsEmailVerified  *bool
	IsKYCCompleted   *bool
	RiskLevel        *string
	CreatedDateRange *valueobjects.DateRange
	LastLoginRange   *valueobjects.DateRange
	Search           string // Busca por nome, email
	Page             int
	Limit            int
	SortBy           string
	SortDesc         bool
	IncludeDeleted   bool
}
