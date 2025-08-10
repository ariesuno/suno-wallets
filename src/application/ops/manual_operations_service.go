package ops

import (
    "context"
    "time"

    obs "suno-wallets/src/infrastructure/observability"
    "suno-wallets/src/shared/helpers"
    "github.com/google/uuid"
)

// Comentários em pt-BR: service para CRUD seguro de USER_MANUAL no ledger

type ManualOperation struct {
    ID            uuid.UUID
    TenantID      uuid.UUID
    CPF           string
    Ticker        string
    AssetType     string
    OperationDate time.Time
    OperationType string
    Quantity      float64
    UnitPrice     *float64
    Currency      string
}

type ManualOpsRepository interface {
    CreateManual(ctx context.Context, op ManualOperation) (uuid.UUID, error)
    UpdateManual(ctx context.Context, op ManualOperation) error
    SoftDeleteManual(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
    ListManual(ctx context.Context, tenantID uuid.UUID, cpf string, filters map[string]interface{}, page, pageSize int) ([]ManualOperation, error)
}

type ManualOperationsService struct{ repo ManualOpsRepository }

func NewManualOperationsService(repo ManualOpsRepository) *ManualOperationsService { return &ManualOperationsService{repo: repo} }

func (s *ManualOperationsService) Create(ctx context.Context, op ManualOperation) (uuid.UUID, error) {
    log := helpers.GetLoggerWithFields(map[string]interface{}{"service": "manual_ops"})
    id, err := s.repo.CreateManual(ctx, op)
    if err != nil {
        obs.IncManualWrites("error")
        return uuid.Nil, err
    }
    log.Info("manual_op_created", map[string]interface{}{"id": id})
    obs.IncManualWrites("success")
    return id, nil
}

func (s *ManualOperationsService) Update(ctx context.Context, op ManualOperation) error {
    return s.repo.UpdateManual(ctx, op)
}

func (s *ManualOperationsService) SoftDelete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
    return s.repo.SoftDeleteManual(ctx, tenantID, id)
}


