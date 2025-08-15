package normalize

import (
	"context"
	"time"

	"suno-wallets/src/domain/interfaces"
	"suno-wallets/src/domain/services"
	normrepo "suno-wallets/src/infrastructure/b3/normalize"
	"suno-wallets/src/infrastructure/b3/persistence"

	"github.com/google/uuid"
)

// Tipos compartilhados

type Repository interface {
	UpsertTransactions(ctx context.Context, items []normrepo.NormalizedTransaction) error
	UpsertPositions(ctx context.Context, items []normrepo.NormalizedPosition) error
	SelectPendingRaw(ctx context.Context, tenantID string, cpf, dataType, assetType string, start, end time.Time, force bool) ([]persistence.RawRecord, error)
	MarkRawNormalized(ctx context.Context, rawID uuid.UUID, count int) error
}

type RunParams struct {
	TenantID  string
	CPF       string
	DataType  string // transactions | positions
	AssetType string // equity
	Start     time.Time
	End       time.Time
	Force     bool
	DryRun    bool
}

type Summary struct{ Inserted, Updated, Skipped, Errors, RawProcessed int }

// NormalizationService implementa Application Service seguindo DDD + SOLID
type NormalizationService struct {
	domainService interfaces.NormalizationService
}

// NewNormalizationService cria serviço DDD
func NewNormalizationService(repo Repository) *NormalizationService {
	// Converter Repository (infra) para interface de domínio
	normalizedRepo := repo.(normrepo.NormalizedRepository)
	domainRepo := normrepo.NewNormalizationRepositoryAdapter(normalizedRepo)

	// Criar serviço de domínio
	domainService := services.NewNormalizationService(domainRepo)

	return &NormalizationService{
		domainService: domainService,
	}
}

// Run mantém compatibilidade com interface existente (Application Service)
func (s *NormalizationService) Run(ctx context.Context, p RunParams) (*Summary, error) {
	// Converter parâmetros de Application para Domain
	domainParams := &interfaces.NormalizationParams{
		TenantID:  p.TenantID,
		CPF:       p.CPF,
		DataType:  p.DataType,
		AssetType: p.AssetType,
		Start:     p.Start,
		End:       p.End,
		Force:     p.Force,
		DryRun:    p.DryRun,
	}

	// Executar através do Domain Service
	domainResult, err := s.domainService.ExecuteNormalization(ctx, domainParams)
	if err != nil {
		return nil, err
	}

	// Converter resultado de Domain para Application
	appResult := &Summary{
		Inserted:     domainResult.TransactionsCount + domainResult.PositionsCount,
		Updated:      0, // DDD não distingue insert/update, sempre upsert
		Skipped:      0,
		Errors:       domainResult.ErrorsCount,
		RawProcessed: domainResult.TotalProcessed,
	}

	return appResult, nil
}
