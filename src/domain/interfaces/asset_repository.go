package interfaces

import (
	"context"

	"suno-wallets/src/domain/entities"
	"suno-wallets/src/domain/enums"
	"suno-wallets/src/domain/valueobjects"

	"github.com/google/uuid"
)

// AssetRepository define a interface para operações com ativos
type AssetRepository interface {
	// Create cria um novo ativo
	Create(ctx context.Context, asset *entities.Asset) error

	// GetByID busca um ativo por ID e tenant
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*entities.Asset, error)

	// GetBySymbol busca um ativo por símbolo e tenant
	GetBySymbol(ctx context.Context, symbol string, tenantID uuid.UUID) (*entities.Asset, error)

	// Update atualiza um ativo existente
	Update(ctx context.Context, asset *entities.Asset) error

	// Delete remove um ativo (soft delete)
	Delete(ctx context.Context, id, tenantID uuid.UUID) error

	// List lista ativos com paginação e filtros
	List(ctx context.Context, params ListAssetParams) ([]*entities.Asset, int64, error)

	// GetByType busca ativos por tipo
	GetByType(ctx context.Context, tenantID uuid.UUID, assetType enums.AssetType) ([]*entities.Asset, error)

	// GetByCurrency busca ativos por moeda
	GetByCurrency(ctx context.Context, tenantID uuid.UUID, currency enums.CurrencyType) ([]*entities.Asset, error)

	// GetTradeable busca ativos que podem ser negociados
	GetTradeable(ctx context.Context, tenantID uuid.UUID) ([]*entities.Asset, error)

	// UpdatePrice atualiza o preço de um ativo
	UpdatePrice(ctx context.Context, id, tenantID uuid.UUID, price *valueobjects.Money, change24h, changePerc float64) error

	// UpdatePrices atualiza preços de múltiplos ativos em batch
	UpdatePrices(ctx context.Context, updates []AssetPriceUpdate) error

	// GetActiveAssets busca ativos ativos
	GetActiveAssets(ctx context.Context, tenantID uuid.UUID) ([]*entities.Asset, error)

	// ExistsBySymbol verifica se já existe ativo com o símbolo
	ExistsBySymbol(ctx context.Context, symbol string, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error)

	// GetAssetsByTag busca ativos por tag
	GetAssetsByTag(ctx context.Context, tenantID uuid.UUID, tag string) ([]*entities.Asset, error)

	// GetHighRiskAssets busca ativos de alto risco
	GetHighRiskAssets(ctx context.Context, tenantID uuid.UUID) ([]*entities.Asset, error)

	// GetStablecoins busca todas as stablecoins
	GetStablecoins(ctx context.Context, tenantID uuid.UUID) ([]*entities.Asset, error)

	// GetTopVolumeAssets busca ativos com maior volume
	GetTopVolumeAssets(ctx context.Context, tenantID uuid.UUID, limit int) ([]*entities.Asset, error)

	// GetAssetsPriceOutdated busca ativos com preços desatualizados
	GetAssetsPriceOutdated(ctx context.Context, tenantID uuid.UUID, maxAge int) ([]*entities.Asset, error)

	// UpdateTradingStatus atualiza status de negociação de um ativo
	UpdateTradingStatus(ctx context.Context, id, tenantID uuid.UUID, isTradeable bool) error

	// BulkUpdateStatus atualiza status de múltiplos ativos
	BulkUpdateStatus(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID, isActive bool) error
}

// ListAssetParams parâmetros para listagem de ativos
type ListAssetParams struct {
	TenantID       uuid.UUID
	Type           *enums.AssetType
	Currency       *enums.CurrencyType
	IsActive       *bool
	IsTradeable    *bool
	IsWithdrawable *bool
	IsDepositable  *bool
	RequiresKYC    *bool
	RiskRating     *string
	Tags           []string
	Search         string // Busca por símbolo, nome
	Page           int
	Limit          int
	SortBy         string
	SortDesc       bool
	IncludeDeleted bool
}

// AssetPriceUpdate estrutura para atualização de preços em batch
type AssetPriceUpdate struct {
	AssetID    uuid.UUID
	TenantID   uuid.UUID
	Price      *valueobjects.Money
	Change24h  float64
	ChangePerc float64
	MarketCap  *valueobjects.Money
	Volume24h  *valueobjects.Money
}
