package reconciliation

import (
    "context"
    "time"

    "github.com/google/uuid"
)

// Comentários em pt-BR: Service orquestra geração de operações de sistema (auto-fix) a partir das inconsistências

type AutoFixRequest struct {
    CPF         string
    Types       []string
    Tickers     []string
    DryRun      bool
    Force       bool
    Concurrency int
}

type OperationPreview struct {
    Ticker      string
    Operation   string
    Date        string
    Quantity    string
    UnitPrice   string
    ReasonCode  string
    Confidence  string
    InconsID    uuid.UUID
}

type AutoFixResult struct {
    Created       int
    Skipped       int
    Pending       int
    ResolvedIDs   []uuid.UUID
    PendingIDs    []uuid.UUID
    Previews      []OperationPreview
    DurationMs    int64
}

type SystemOpsRepository interface {
    TryAcquireLock(ctx context.Context, tenantID uuid.UUID, cpf string, ttlSeconds int) (bool, error)
    ReleaseLock(ctx context.Context, tenantID uuid.UUID, cpf string) error

    // leitura de inconsistências abertas
    ListOpenInconsistencies(ctx context.Context, tenantID uuid.UUID, cpf string, types []string, tickers []string) ([]Inconsistency, error)

    // geração no ledger (idempotente)
    UpsertSystemOperation(ctx context.Context, tenantID uuid.UUID, op OperationPreview) (bool, error)

    // atualização do status da inconsistência e details (anexar generated_op_ids)
    ResolveInconsistency(ctx context.Context, tenantID uuid.UUID, inconsID uuid.UUID, generatedIDs []uuid.UUID) error
}

type PriceLookupPort interface {
    GetClosingPrice(ctx context.Context, ticker string, date time.Time) (float64, bool, error)
}

type SystemOperationsService struct {
    repo  SystemOpsRepository
    price PriceLookupPort
}

func NewSystemOperationsService(repo SystemOpsRepository, price PriceLookupPort) *SystemOperationsService {
    return &SystemOperationsService{repo: repo, price: price}
}

func (s *SystemOperationsService) AutoFix(ctx context.Context, tenantID uuid.UUID, req AutoFixRequest) (*AutoFixResult, error) {
    started := time.Now()

    locked, err := s.repo.TryAcquireLock(ctx, tenantID, req.CPF, 180)
    if err != nil || !locked {
        return nil, err
    }
    defer func() { _ = s.repo.ReleaseLock(ctx, tenantID, req.CPF) }()

    // buscar inconsistências abertas para os tipos alvo
    incs, err := s.repo.ListOpenInconsistencies(ctx, tenantID, req.CPF, req.Types, req.Tickers)
    if err != nil {
        return nil, err
    }

    res := &AutoFixResult{}

    for _, inc := range incs {
        switch inc.Type {
        case "OPENING_BALANCE_MISSING":
            prev := OperationPreview{
                Ticker:     inc.Ticker,
                Operation:  "OPENING_BALANCE",
                Date:       time.Now().Format("2006-01-02"), // placeholder: ideal usar earliest date
                Quantity:   "0",                               // placeholder até termos qty calculada no repo
                UnitPrice:  "",
                ReasonCode: "OPENING_BALANCE_PRE_API",
                Confidence: "UNKNOWN",
                InconsID:   inc.ID,
            }
            res.Previews = append(res.Previews, prev)
            if req.DryRun {
                continue
            }
            created, err := s.repo.UpsertSystemOperation(ctx, tenantID, prev)
            if err != nil {
                return nil, err
            }
            if created {
                res.Created++
            } else {
                res.Skipped++
            }
            // marcar resolvido
            _ = s.repo.ResolveInconsistency(ctx, tenantID, inc.ID, nil)
            res.ResolvedIDs = append(res.ResolvedIDs, inc.ID)

        case "SELL_WITHOUT_BUY":
            prev := OperationPreview{
                Ticker:     inc.Ticker,
                Operation:  "BUY",
                Date:       time.Now().Format("2006-01-02"), // placeholder: ideal usar data da venda
                Quantity:   "0",
                UnitPrice:  "",
                ReasonCode: "ZERO_PNL_PRE_API",
                Confidence: "MIRRORED_SELL",
                InconsID:   inc.ID,
            }
            res.Previews = append(res.Previews, prev)
            if req.DryRun { continue }
            created, err := s.repo.UpsertSystemOperation(ctx, tenantID, prev)
            if err != nil { return nil, err }
            if created { res.Created++ } else { res.Skipped++ }
            _ = s.repo.ResolveInconsistency(ctx, tenantID, inc.ID, nil)
            res.ResolvedIDs = append(res.ResolvedIDs, inc.ID)
        }
    }

    res.DurationMs = time.Since(started).Milliseconds()
    return res, nil
}


