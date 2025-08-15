package interfaces

import (
	"context"
	"time"

	"suno-wallets/src/domain/entities"

	"github.com/google/uuid"
)

// RawDataRecord representa um registro RAW (interface para infraestrutura)
type RawDataRecord interface {
	GetID() uuid.UUID
	GetTenantID() string
	GetCPF() string
	GetDataType() string
	GetAssetType() string
	GetPayload() []byte
	GetPeriodStart() time.Time
	GetPeriodEnd() time.Time
}

// NormalizationRepository define contratos para persistência de dados normalizados (DDD)
type NormalizationRepository interface {
	// Buscar dados RAW para normalização
	FindPendingRawData(ctx context.Context, tenantID string, cpf, dataType, assetType string, start, end time.Time, force bool) ([]RawDataRecord, error)

	// Persistir dados normalizados
	SaveNormalizedTransactions(ctx context.Context, transactions []*entities.NormalizedTransaction) error
	SaveNormalizedPositions(ctx context.Context, positions []*entities.NormalizedPosition) error

	// Marcar RAW como processado
	MarkRawAsProcessed(ctx context.Context, rawID uuid.UUID, count int) error
}

// NormalizationService define contratos para serviços de domínio
type NormalizationService interface {
	// Validar parâmetros de normalização
	ValidateNormalizationParams(ctx context.Context, params *NormalizationParams) error

	// Executar normalização completa
	ExecuteNormalization(ctx context.Context, params *NormalizationParams) (*NormalizationResult, error)
}

// NormalizationParams parâmetros para normalização (Domain)
type NormalizationParams struct {
	TenantID  string
	CPF       string
	DataType  string
	AssetType string
	Start     time.Time
	End       time.Time
	Force     bool
	DryRun    bool
}

// NormalizationResult resultado da normalização (Domain)
type NormalizationResult struct {
	TotalProcessed    int                               `json:"total_processed"`
	TransactionsCount int                               `json:"transactions_count"`
	PositionsCount    int                               `json:"positions_count"`
	ErrorsCount       int                               `json:"errors_count"`
	StartedAt         time.Time                         `json:"started_at"`
	CompletedAt       time.Time                         `json:"completed_at"`
	Transactions      []*entities.NormalizedTransaction `json:"transactions,omitempty"`
	Positions         []*entities.NormalizedPosition    `json:"positions,omitempty"`
	Errors            []error                           `json:"errors,omitempty"`
}
