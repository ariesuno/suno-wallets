package repositories

import (
	"context"

	"suno-wallets/src/domain/entities"

	"github.com/google/uuid"
)

// WalletRepository define a interface para operações com carteiras
type WalletRepository interface {
	// Create cria uma nova carteira
	Create(ctx context.Context, wallet *entities.Wallet) error

	// GetByID busca uma carteira por ID e tenant
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*entities.Wallet, error)

	// GetByOwner busca carteiras por proprietário e tenant
	GetByOwner(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*entities.Wallet, error)

	// GetDefaultByOwner busca a carteira padrão do proprietário
	GetDefaultByOwner(ctx context.Context, ownerID, tenantID uuid.UUID) (*entities.Wallet, error)

	// Update atualiza uma carteira existente
	Update(ctx context.Context, wallet *entities.Wallet) error

	// Delete remove uma carteira (soft delete)
	Delete(ctx context.Context, id, tenantID uuid.UUID) error

	// List lista carteiras com paginação e filtros
	List(ctx context.Context, params ListWalletParams) ([]*entities.Wallet, int64, error)

	// UpdateBalance atualiza o saldo de uma carteira
	UpdateBalance(ctx context.Context, id, tenantID uuid.UUID, newBalance int64) error

	// ExistsDefaultWallet verifica se já existe carteira padrão para o proprietário
	ExistsDefaultWallet(ctx context.Context, ownerID, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error)
}

// ListWalletParams parâmetros para listagem de carteiras
type ListWalletParams struct {
	TenantID uuid.UUID
	OwnerID  *uuid.UUID
	Status   *entities.WalletStatus
	Type     *entities.WalletType
	Page     int
	Limit    int
	Search   string
	SortBy   string
	SortDesc bool
}
