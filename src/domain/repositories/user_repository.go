package repositories

import (
	"context"

	"suno-wallets/src/domain/entities"

	"github.com/google/uuid"
)

// UserRepository define a interface para operações com usuários
type UserRepository interface {
	// Create cria um novo usuário
	Create(ctx context.Context, user *entities.User) error

	// FindByID busca um usuário por ID e tenant
	FindByID(ctx context.Context, id, tenantID uuid.UUID) (*entities.User, error)

	// FindByEmail busca um usuário por email e tenant
	FindByEmail(ctx context.Context, email string, tenantID uuid.UUID) (*entities.User, error)

	// Update atualiza um usuário existente
	Update(ctx context.Context, user *entities.User) error

	// Delete remove um usuário (soft delete)
	Delete(ctx context.Context, id, tenantID uuid.UUID) error

	// List lista usuários com paginação e filtros
	List(ctx context.Context, params ListUserParams) ([]*entities.User, int64, error)

	// ExistsByEmail verifica se já existe usuário com o email
	ExistsByEmail(ctx context.Context, email string, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error)
}

// ListUserParams parâmetros para listagem de usuários
type ListUserParams struct {
	TenantID uuid.UUID
	Status   *entities.UserStatus
	Page     int
	Limit    int
	Search   string
	SortBy   string
	SortDesc bool
}
