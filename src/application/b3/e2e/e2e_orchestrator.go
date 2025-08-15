package e2e

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	appingest "suno-wallets/src/application/b3/ingest"
	appnorm "suno-wallets/src/application/b3/normalize"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"
)

// Comentários em pt-BR: Orquestrador E2E: reset seguro → ingest histórico → normalize

// ResetRepository define operações de reset/locks necessárias pelo orquestrador (implementação em Infrastructure)
type ResetRepository interface {
	TryAcquireLock(ctx context.Context, tenantID string, cpf string) (bool, error)
	ReleaseLock(ctx context.Context, tenantID string, cpf string) error
	Reset(ctx context.Context, tenantID string, cpf string, mode string, archivedBy string) (*ResetResult, error)
}

// ResetResult resume o efeito do reset seguro
type ResetResult struct {
	RawMoved       int
	RawDeleted     int
	NormTxMoved    int
	NormTxDeleted  int
	NormPosMoved   int
	NormPosDeleted int
	PeriodsCleared int
	SyncStateReset bool
}

// Params parâmetros do orquestrador E2E
type Params struct {
	TenantID    string
	CPF         string
	AssetTypes  []string // ex.: ["equity"]
	DataTypes   []string // ex.: ["transactions","positions"]
	Start       time.Time
	End         time.Time
	Force       bool
	DryRun      bool
	Mode        string // "archive" | "hard-delete"
	RequestedBy string // para auditoria
}

// Summary consolida métricas do E2E
type Summary struct {
	Raw        struct{ Saved, Skipped, Errors, MonthsProcessed, PagesProcessed int }
	Normalized struct{ Inserted, Updated, Skipped, Errors int }
	StartedAt  time.Time
	FinishedAt time.Time
	DurationMs int64
	Mode       string
	Force      bool
	DryRun     bool
}

type Orchestrator struct {
	resetRepo ResetRepository
	ingest    IngestPort
	normalize NormalizePort
	// Performance configs
	maxConcurrentIngests int // número máximo de ingestões paralelas
}

// IngestPort define a porta usada pelo orquestrador para ingestão
type IngestPort interface {
	Ingest(ctx context.Context, p appingest.IngestParams) (*appingest.Summary, error)
}

// NormalizePort define a porta usada pelo orquestrador para normalização
type NormalizePort interface {
	Run(ctx context.Context, p appnorm.RunParams) (*appnorm.Summary, error)
}

func NewOrchestrator(resetRepo ResetRepository, ingestSvc *appingest.Service, normSvc *appnorm.LegacyService) *Orchestrator {
	return &Orchestrator{
		resetRepo:            resetRepo,
		ingest:               ingestSvc,
		normalize:            normSvc,
		maxConcurrentIngests: 10, // default: 10 workers paralelos
	}
}

// NewOrchestratorPorts permite injeção de fakes/mocks nos testes
func NewOrchestratorPorts(resetRepo ResetRepository, ingest IngestPort, normalize NormalizePort) *Orchestrator {
	return &Orchestrator{
		resetRepo:            resetRepo,
		ingest:               ingest,
		normalize:            normalize,
		maxConcurrentIngests: 10, // default: 10 workers paralelos
	}
}

// WithConcurrency permite configurar o número de workers paralelos
func (o *Orchestrator) WithConcurrency(maxConcurrent int) *Orchestrator {
	if maxConcurrent > 0 {
		o.maxConcurrentIngests = maxConcurrent
	}
	return o
}

// Run executa o fluxo fim-a-fim. Mantém idempotência delegando às camadas de ingest/normalize.
func (o *Orchestrator) Run(ctx context.Context, p Params) (*Summary, error) {
	started := time.Now()
	sum := &Summary{Mode: p.Mode, Force: p.Force, DryRun: p.DryRun, StartedAt: started}
	logBase := map[string]any{
		"tenantId":  p.TenantID,
		"cpfMasked": maskCPF(p.CPF),
		"mode":      p.Mode,
		"force":     p.Force,
		"dryRun":    p.DryRun,
	}

	// Validar entradas básicas (regras detalhadas ficam no controller)
	if p.Start.After(p.End) {
		return nil, errors.New("start must be <= end")
	}
	if len(p.AssetTypes) == 0 || len(p.DataTypes) == 0 {
		return nil, errors.New("assetTypes and dataTypes must not be empty")
	}

	// Lock por (tenant, cpf)
	locked, err := o.resetRepo.TryAcquireLock(ctx, p.TenantID, p.CPF)
	if err != nil {
		observability.IncE2EError("lock")
		helpers.LogError("e2e lock failed", err, logBase)
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		observability.IncE2EError("lock")
		helpers.LogWarn("e2e lock not acquired", logBase)
		return nil, errors.New("concurrent e2e run in progress for this tenant/cpf")
	}
	defer func() { _ = o.resetRepo.ReleaseLock(ctx, p.TenantID, p.CPF) }()

	// DryRun retorna somente o plano (janelas mensais)
	if p.DryRun {
		// Construção do plano mensal (sem executar)
		months := monthWindows(p.Start, p.End)
		sum.Raw.MonthsProcessed = len(months)
		// Estimativa simples de páginas: 1 por mês
		sum.Raw.PagesProcessed = len(months)
		helpers.LogInfo("e2e dryrun plan", merge(logBase, map[string]any{"months": len(months)}))
		sum.FinishedAt = time.Now()
		sum.DurationMs = time.Since(started).Milliseconds()
		observability.ObserveE2ERun("dryrun", p.Mode, "-", "-", started)
		return sum, nil
	}

	// Reset seguro (archive ou hard-delete)
	if _, err := o.resetRepo.Reset(ctx, p.TenantID, p.CPF, p.Mode, p.RequestedBy); err != nil {
		observability.IncE2EError("reset")
		helpers.LogError("e2e reset failed", err, logBase)
		return nil, fmt.Errorf("reset failed: %w", err)
	}
	resetDuration := time.Since(started).Milliseconds()
	helpers.LogInfo("e2e reset done", merge(logBase, map[string]any{"durationMs": resetDuration}))

	// Ingestão histórica com priorização inteligente
	ingestStart := time.Now()
	if err := o.runPrioritizedIngestion(ctx, p, sum, logBase); err != nil {
		return nil, err
	}
	ingestDuration := time.Since(ingestStart).Milliseconds()
	helpers.LogInfo("e2e prioritized ingestion completed", merge(logBase, map[string]any{
		"durationMs": ingestDuration,
		"rawSaved":   sum.Raw.Saved,
		"workers":    o.maxConcurrentIngests,
	}))

	// Normalização apenas se houver dados raw
	if sum.Raw.Saved > 0 {
		normalizeStart := time.Now()
		if err := o.runPipelineNormalization(ctx, p, sum, logBase); err != nil {
			return nil, err
		}
		normalizeDuration := time.Since(normalizeStart).Milliseconds()
		helpers.LogInfo("e2e pipeline normalization completed", merge(logBase, map[string]any{
			"durationMs":    normalizeDuration,
			"transInserted": sum.Normalized.Inserted,
			"transUpdated":  sum.Normalized.Updated,
		}))
	} else {
		helpers.LogInfo("e2e normalization skipped - no raw data found", logBase)
	}

	sum.FinishedAt = time.Now()
	sum.DurationMs = time.Since(started).Milliseconds()
	helpers.LogInfo("e2e finished", merge(logBase, map[string]any{
		"durationMs": sum.DurationMs,
		"raw": map[string]int{
			"saved": sum.Raw.Saved, "skipped": sum.Raw.Skipped, "errors": sum.Raw.Errors,
		},
		"normalized": map[string]int{
			"inserted": sum.Normalized.Inserted, "updated": sum.Normalized.Updated, "skipped": sum.Normalized.Skipped, "errors": sum.Normalized.Errors,
		},
	}))
	return sum, nil
}

// monthWindows gera janelas mensais de start→end (inclusive)
func monthWindows(start, end time.Time) [][2]time.Time {
	var out [][2]time.Time
	cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cur.After(last) {
		next := cur.AddDate(0, 1, 0).Add(-24 * time.Hour)
		if next.After(end) {
			next = end
		}
		winStart := cur
		if winStart.Before(start) {
			winStart = start
		}
		out = append(out, [2]time.Time{winStart, next})
		cur = cur.AddDate(0, 1, 0)
	}
	return out
}

// maskCPF mascara o CPF mantendo apenas os 2 últimos dígitos
func maskCPF(cpf string) string {
	if len(cpf) != 11 {
		return "invalid"
	}
	return "*********" + cpf[9:]
}

// runPrioritizedIngestion executa ingestão com priorização inteligente
// 1. Primeiro executa transações (se não houver, não precisa buscar posições)
// 2. Depois executa posições apenas se houver transações
func (o *Orchestrator) runPrioritizedIngestion(ctx context.Context, p Params, sum *Summary, logBase map[string]any) error {
	// Etapa 1: Buscar transações primeiro
	hasTransactions := false
	for _, assetType := range p.AssetTypes {
		if contains(p.DataTypes, "transactions") {
			transSum, err := o.runDataTypeIngestion(ctx, p, assetType, "transactions", logBase)
			if err != nil {
				return fmt.Errorf("failed to ingest transactions for %s: %w", assetType, err)
			}
			if transSum != nil {
				sum.Raw.Saved += transSum.Saved
				sum.Raw.Skipped += transSum.Skipped
				sum.Raw.Errors += transSum.Errors
				sum.Raw.MonthsProcessed += transSum.MonthsProcessed
				sum.Raw.PagesProcessed += transSum.PagesProcessed

				if transSum.Saved > 0 {
					hasTransactions = true
				}
			}
		}
	}

	// Etapa 2: Buscar posições apenas se houver transações
	if hasTransactions && contains(p.DataTypes, "positions") {
		for _, assetType := range p.AssetTypes {
			posSum, err := o.runDataTypeIngestion(ctx, p, assetType, "positions", logBase)
			if err != nil {
				return fmt.Errorf("failed to ingest positions for %s: %w", assetType, err)
			}
			if posSum != nil {
				sum.Raw.Saved += posSum.Saved
				sum.Raw.Skipped += posSum.Skipped
				sum.Raw.Errors += posSum.Errors
				sum.Raw.MonthsProcessed += posSum.MonthsProcessed
				sum.Raw.PagesProcessed += posSum.PagesProcessed
			}
		}
		helpers.LogInfo("e2e positions ingestion completed", merge(logBase, map[string]any{
			"hasTransactions": hasTransactions,
		}))
	} else if !hasTransactions {
		helpers.LogInfo("e2e positions ingestion skipped - no transactions found", logBase)
	}

	return nil
}

// runDataTypeIngestion executa ingestão para um tipo de dados específico
func (o *Orchestrator) runDataTypeIngestion(ctx context.Context, p Params, assetType, dataType string, logBase map[string]any) (*appingest.Summary, error) {
	var start, end string

	// Para posições, usar apenas o dia anterior (otimização)
	if dataType == "positions" {
		yesterday := p.End.AddDate(0, 0, -1)
		start = yesterday.Format("2006-01-02")
		end = yesterday.Format("2006-01-02")
		helpers.LogInfo("e2e positions ingestion optimized", merge(logBase, map[string]any{
			"dataType": dataType,
			"period":   start,
		}))
	} else {
		// Para transações, usar período completo até D-1 (ontem)
		start = p.Start.Format("2006-01-02")
		end = p.End.Format("2006-01-02")
		helpers.LogInfo("e2e transactions ingestion period", merge(logBase, map[string]any{
			"dataType": dataType,
			"start":    start,
			"end":      end,
		}))
	}

	return o.ingest.Ingest(ctx, appingest.IngestParams{
		TenantID:  p.TenantID,
		CPF:       p.CPF,
		DataType:  dataType,
		AssetType: assetType,
		Start:     start,
		End:       end,
		FetchAll:  true,
		Force:     p.Force,
		DryRun:    false,
	})
}

// runPipelineNormalization executa normalização em pipeline por tipo
func (o *Orchestrator) runPipelineNormalization(ctx context.Context, p Params, sum *Summary, logBase map[string]any) error {
	// Normalization pipeline: processa cada combinação dataType/assetType em paralelo
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var errs []error

	for _, assetType := range p.AssetTypes {
		for _, dataType := range p.DataTypes {
			wg.Add(1)
			go func(dt, at string) {
				defer wg.Done()

				normSum, err := o.normalize.Run(ctx, appnorm.RunParams{
					TenantID:  p.TenantID,
					CPF:       p.CPF,
					DataType:  dt,
					AssetType: at,
					Start:     p.Start,
					End:       p.End,
					Force:     p.Force,
					DryRun:    false,
				})

				mutex.Lock()
				defer mutex.Unlock()

				if err != nil {
					observability.IncE2EError("normalize")
					helpers.LogError("e2e pipeline normalize failed", err, merge(logBase, map[string]any{
						"dataType":  dt,
						"assetType": at,
					}))
					errs = append(errs, fmt.Errorf("normalize failed for %s/%s: %w", dt, at, err))
				} else if normSum != nil {
					sum.Normalized.Inserted += normSum.Inserted
					sum.Normalized.Updated += normSum.Updated
					sum.Normalized.Skipped += normSum.Skipped
					sum.Normalized.Errors += normSum.Errors

					observability.ObserveE2ERun("success", p.Mode, dt, at, time.Now())
					helpers.LogInfo("e2e normalize stage done", merge(logBase, map[string]any{
						"stage":     "normalize",
						"dataType":  dt,
						"assetType": at,
						"inserted":  normSum.Inserted,
						"updated":   normSum.Updated,
					}))
				}
			}(dataType, assetType)
		}
	}

	wg.Wait()

	// Return first error if any occurred
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// merge combina mapas base + extra para logging
func merge(base map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// contains verifica se um slice contém um elemento
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
