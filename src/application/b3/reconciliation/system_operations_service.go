package reconciliation

import (
    "context"
    "os"
    "time"

    obs "suno-wallets/src/infrastructure/observability"
    "suno-wallets/src/shared/helpers"
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
    CPF         string
    Ticker      string
    Operation   string
    Date        string
    Quantity    float64
    UnitPrice   *float64
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
    GetInconsistencyByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*Inconsistency, error)

    // geração no ledger (idempotente)
    UpsertSystemOperation(ctx context.Context, tenantID uuid.UUID, op OperationPreview) (bool, error)

    // atualização do status da inconsistência e details (anexar generated_op_ids)
    ResolveInconsistency(ctx context.Context, tenantID uuid.UUID, inconsID uuid.UUID, generatedIDs []uuid.UUID) error

    // dados auxiliares
    GetFirstSellTxDetails(ctx context.Context, tenantID uuid.UUID, cpf string, ticker string) (date string, qty float64, price float64, found bool, err error)
    DerivePriceFromPosition(ctx context.Context, tenantID uuid.UUID, cpf, ticker, date string) (price float64, found bool, err error)
}

type PriceLookupPort interface {
    GetClosingPrice(ctx context.Context, ticker string, date time.Time) (float64, bool, error)
}

type SystemOperationsService struct {
    Repo  SystemOpsRepository
    price PriceLookupPort
}

func NewSystemOperationsService(repo SystemOpsRepository, price PriceLookupPort) *SystemOperationsService {
    return &SystemOperationsService{Repo: repo, price: price}
}

func (s *SystemOperationsService) AutoFix(ctx context.Context, tenantID uuid.UUID, req AutoFixRequest) (*AutoFixResult, error) {
    started := time.Now()
    log := helpers.GetLoggerWithFields(map[string]interface{}{
        "service":   "autofix_service",
        "tenantId":  tenantID.String(),
        "cpfMasked": maskCPF(req.CPF),
        "types":     req.Types,
        "tickers":   req.Tickers,
        "dryRun":    req.DryRun,
        "force":     req.Force,
    })

    locked, err := s.Repo.TryAcquireLock(ctx, tenantID, req.CPF, 180)
    if err != nil || !locked {
        return nil, err
    }
    defer func() { _ = s.Repo.ReleaseLock(ctx, tenantID, req.CPF) }()

    // buscar inconsistências abertas para os tipos alvo
    incs, err := s.Repo.ListOpenInconsistencies(ctx, tenantID, req.CPF, req.Types, req.Tickers)
    if err != nil {
        obs.ObserveAutofixRun("error", started)
        return nil, err
    }

    res := &AutoFixResult{}

    for _, inc := range incs {
        switch inc.Type {
        case "OPENING_BALANCE_MISSING":
            // calcular data e quantidade
            firstDate := ""
            if inc.SampleDates != nil {
                if v, ok := inc.SampleDates["first_position_date"].(string); ok { firstDate = v }
            }
            opDate := firstDate
            if opDate == "" { opDate = time.Now().Format("2006-01-02") }
            var posQty, netTx float64
            if inc.Details != nil {
                if v, ok := inc.Details["position_qty"].(float64); ok { posQty = v }
                if v, ok := inc.Details["net_tx_until_first_position"].(float64); ok { netTx = v }
            }
            qty := posQty - netTx
            if qty < 0 { qty = 0 }
            // preço
            var unitPrice *float64
            confidence := "UNKNOWN"
            // tentar mercado
            if s.price != nil {
                if d, err := time.Parse("2006-01-02", opDate); err == nil {
                    if p, ok, _ := s.price.GetClosingPrice(ctx, inc.Ticker, d); ok {
                        unitPrice = &p
                        confidence = "MARKET_CLOSE"
                    }
                }
            }
            // derivar de posição
            if unitPrice == nil {
                if p, ok, _ := s.Repo.DerivePriceFromPosition(ctx, tenantID, req.CPF, inc.Ticker, opDate); ok {
                    unitPrice = &p
                    confidence = "DERIVED_POSITION"
                }
            }
            // checar AUTO_FIX_REQUIRE_PRICE
            requirePrice := false
            if v := getenv("AUTO_FIX_REQUIRE_PRICE"); v == "true" || v == "1" { requirePrice = true }
            if requirePrice && unitPrice == nil {
                res.Pending++
                res.PendingIDs = append(res.PendingIDs, inc.ID)
                obs.IncAutofixPending("PENDING_PRICE", 1)
                continue
            }
            prev := OperationPreview{CPF: req.CPF, Ticker: inc.Ticker, Operation: "OPENING_BALANCE", Date: opDate, Quantity: qty, UnitPrice: unitPrice, ReasonCode: "OPENING_BALANCE_PRE_API", Confidence: confidence, InconsID: inc.ID}
            res.Previews = append(res.Previews, prev)
            if req.DryRun {
                continue
            }
            created, err := s.Repo.UpsertSystemOperation(ctx, tenantID, prev)
            if err != nil {
                obs.ObserveAutofixRun("error", started)
                return nil, err
            }
            if created {
                res.Created++
                obs.IncAutofixOpsCreated("OPENING_BALANCE_PRE_API", 1)
            } else {
                res.Skipped++
            }
            // marcar resolvido
            _ = s.Repo.ResolveInconsistency(ctx, tenantID, inc.ID, nil)
            res.ResolvedIDs = append(res.ResolvedIDs, inc.ID)

        case "SELL_WITHOUT_BUY":
            dateStr, qty, price, found, _ := s.Repo.GetFirstSellTxDetails(ctx, tenantID, req.CPF, inc.Ticker)
            if !found { dateStr = time.Now().Format("2006-01-02") }
            unitPrice := &price
            prev := OperationPreview{CPF: req.CPF, Ticker: inc.Ticker, Operation: "BUY", Date: dateStr, Quantity: qty, UnitPrice: unitPrice, ReasonCode: "ZERO_PNL_PRE_API", Confidence: "MIRRORED_SELL", InconsID: inc.ID}
            res.Previews = append(res.Previews, prev)
            if req.DryRun { continue }
            created, err := s.Repo.UpsertSystemOperation(ctx, tenantID, prev)
            if err != nil { obs.ObserveAutofixRun("error", started); return nil, err }
            if created { res.Created++; obs.IncAutofixOpsCreated("ZERO_PNL_PRE_API", 1) } else { res.Skipped++ }
            _ = s.Repo.ResolveInconsistency(ctx, tenantID, inc.ID, nil)
            res.ResolvedIDs = append(res.ResolvedIDs, inc.ID)
        }
    }

    res.DurationMs = time.Since(started).Milliseconds()
    log.Info("autofix_finished", map[string]interface{}{
        "created": res.Created,
        "skipped": res.Skipped,
        "pending": res.Pending,
        "durationMs": res.DurationMs,
    })
    obs.ObserveAutofixRun("success", started)
    return res, nil
}

// getenv wrapper para facilitar testes
func getenv(key string) string { return os.Getenv(key) }


