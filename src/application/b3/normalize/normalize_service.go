package normalize

import (
    "context"
    "encoding/json"
    "time"

    "github.com/google/uuid"
    normrepo "suno-wallets/src/infrastructure/b3/normalize"
)

// Comentários em pt-BR: serviço de normalização (orquestra parsers e persistência)

type Repository interface {
    UpsertTransactions(ctx context.Context, items []normrepo.NormalizedTransaction) error
    UpsertPositions(ctx context.Context, items []normrepo.NormalizedPosition) error
}

type Service struct { repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

type RunParams struct {
    TenantID  uuid.UUID
    CPF       string
    DataType  string // transactions | positions
    AssetType string // equity
    RawID     uuid.UUID
    Payload   json.RawMessage
    Force     bool
    DryRun    bool
}

type Summary struct { Inserted, Updated, Skipped, Errors, RawProcessed int }

func (s *Service) Run(ctx context.Context, p RunParams) (*Summary, error) {
    sum := &Summary{RawProcessed: 1}
    switch p.DataType {
    case "transactions":
        items, _ := normrepo.NormalizeTransactions(p.TenantID, p.CPF, p.AssetType, p.RawID, p.Payload)
        if !p.DryRun { _ = s.repo.UpsertTransactions(ctx, items) }
        sum.Inserted = len(items)
    case "positions":
        items, _ := normrepo.NormalizePositions(p.TenantID, p.CPF, p.AssetType, p.RawID, p.Payload)
        if !p.DryRun { _ = s.repo.UpsertPositions(ctx, items) }
        sum.Inserted = len(items)
    default:
        return sum, nil
    }
    _ = time.Now() // placeholder de métricas/logs futuras
    return sum, nil
}


