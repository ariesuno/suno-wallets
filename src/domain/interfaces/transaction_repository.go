package interfaces

import (
	"context"

	"suno-wallets/src/domain/entities"
	"suno-wallets/src/domain/enums"
	"suno-wallets/src/domain/valueobjects"

	"github.com/google/uuid"
)

// TransactionRepository define a interface para operações com transações
type TransactionRepository interface {
	// Create cria uma nova transação
	Create(ctx context.Context, transaction *entities.Transaction) error

	// GetByID busca uma transação por ID e tenant
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*entities.Transaction, error)

	// GetByWallet busca transações de uma carteira
	GetByWallet(ctx context.Context, walletID, tenantID uuid.UUID, params ListTransactionParams) ([]*entities.Transaction, int64, error)

	// GetByUser busca transações de um usuário
	GetByUser(ctx context.Context, userID, tenantID uuid.UUID, params ListTransactionParams) ([]*entities.Transaction, int64, error)

	// Update atualiza uma transação existente
	Update(ctx context.Context, transaction *entities.Transaction) error

	// UpdateStatus atualiza o status de uma transação
	UpdateStatus(ctx context.Context, id, tenantID uuid.UUID, status enums.TransactionStatus, notes string) error

	// List lista transações com paginação e filtros
	List(ctx context.Context, params ListTransactionParams) ([]*entities.Transaction, int64, error)

	// GetPendingTransactions busca transações pendentes
	GetPendingTransactions(ctx context.Context, tenantID uuid.UUID) ([]*entities.Transaction, error)

	// GetProcessingTransactions busca transações em processamento
	GetProcessingTransactions(ctx context.Context, tenantID uuid.UUID) ([]*entities.Transaction, error)

	// GetExpiredTransactions busca transações expiradas
	GetExpiredTransactions(ctx context.Context, tenantID uuid.UUID) ([]*entities.Transaction, error)

	// GetTransactionsByDateRange busca transações por intervalo de datas
	GetTransactionsByDateRange(ctx context.Context, tenantID uuid.UUID, dateRange *valueobjects.DateRange) ([]*entities.Transaction, error)

	// GetDailyTransactionSum calcula soma de transações por dia
	GetDailyTransactionSum(ctx context.Context, userID, tenantID uuid.UUID, date *valueobjects.DateRange) (*valueobjects.Money, error)

	// GetMonthlyTransactionSum calcula soma de transações por mês
	GetMonthlyTransactionSum(ctx context.Context, userID, tenantID uuid.UUID, date *valueobjects.DateRange) (*valueobjects.Money, error)

	// GetTransactionStats busca estatísticas de transações
	GetTransactionStats(ctx context.Context, params TransactionStatsParams) (*TransactionStats, error)

	// BulkUpdateStatus atualiza status de múltiplas transações
	BulkUpdateStatus(ctx context.Context, tenantID uuid.UUID, transactionIDs []uuid.UUID, status enums.TransactionStatus) error

	// GetTransactionsByReference busca transações por referência externa
	GetTransactionsByReference(ctx context.Context, reference string, tenantID uuid.UUID) ([]*entities.Transaction, error)

	// GetLargeTransactions busca transações acima de um valor
	GetLargeTransactions(ctx context.Context, tenantID uuid.UUID, minAmount *valueobjects.Money) ([]*entities.Transaction, error)

	// GetSuspiciousTransactions busca transações suspeitas para compliance
	GetSuspiciousTransactions(ctx context.Context, tenantID uuid.UUID) ([]*entities.Transaction, error)

	// GetUserTransactionHistory busca histórico completo de um usuário
	GetUserTransactionHistory(ctx context.Context, userID, tenantID uuid.UUID, dateRange *valueobjects.DateRange) ([]*entities.Transaction, error)
}

// ListTransactionParams parâmetros para listagem de transações
type ListTransactionParams struct {
	TenantID        uuid.UUID
	WalletID        *uuid.UUID
	UserID          *uuid.UUID
	AssetID         *uuid.UUID
	Type            *enums.TransactionType
	Status          *enums.TransactionStatus
	DateRange       *valueobjects.DateRange
	MinAmount       *valueobjects.Money
	MaxAmount       *valueobjects.Money
	Reference       string
	Search          string // Busca por descrição, referência
	Page            int
	Limit           int
	SortBy          string
	SortDesc        bool
	IncludeReversed bool
}

// TransactionStatsParams parâmetros para estatísticas de transações
type TransactionStatsParams struct {
	TenantID  uuid.UUID
	UserID    *uuid.UUID
	WalletID  *uuid.UUID
	AssetID   *uuid.UUID
	DateRange *valueobjects.DateRange
	GroupBy   string // daily, weekly, monthly, yearly
}

// TransactionStats estatísticas de transações
type TransactionStats struct {
	TotalCount    int64                    `json:"total_count"`
	TotalVolume   *valueobjects.Money      `json:"total_volume"`
	AverageAmount *valueobjects.Money      `json:"average_amount"`
	ByType        map[string]int64         `json:"by_type"`
	ByStatus      map[string]int64         `json:"by_status"`
	DailyVolume   []*DailyVolumeStats      `json:"daily_volume,omitempty"`
	TopUsers      []*UserTransactionStats  `json:"top_users,omitempty"`
	TopAssets     []*AssetTransactionStats `json:"top_assets,omitempty"`
}

// DailyVolumeStats estatísticas de volume diário
type DailyVolumeStats struct {
	Date   string              `json:"date"`
	Count  int64               `json:"count"`
	Volume *valueobjects.Money `json:"volume"`
}

// UserTransactionStats estatísticas de transações por usuário
type UserTransactionStats struct {
	UserID uuid.UUID           `json:"user_id"`
	Count  int64               `json:"count"`
	Volume *valueobjects.Money `json:"volume"`
}

// AssetTransactionStats estatísticas de transações por ativo
type AssetTransactionStats struct {
	AssetID uuid.UUID           `json:"asset_id"`
	Symbol  string              `json:"symbol"`
	Count   int64               `json:"count"`
	Volume  *valueobjects.Money `json:"volume"`
}
