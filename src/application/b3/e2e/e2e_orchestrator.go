package e2e

import (
	"context"
	"errors"
	"fmt"
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
}

// IngestPort define a porta usada pelo orquestrador para ingestão
type IngestPort interface {
	Ingest(ctx context.Context, p appingest.IngestParams) (*appingest.Summary, error)
}

// NormalizePort define a porta usada pelo orquestrador para normalização
type NormalizePort interface {
	Run(ctx context.Context, p appnorm.RunParams) (*appnorm.Summary, error)
}

func NewOrchestrator(resetRepo ResetRepository, ingestSvc *appingest.Service, normSvc *appnorm.Service) *Orchestrator {
	return &Orchestrator{resetRepo: resetRepo, ingest: ingestSvc, normalize: normSvc}
}

// NewOrchestratorPorts permite injeção de fakes/mocks nos testes
func NewOrchestratorPorts(resetRepo ResetRepository, ingest IngestPort, normalize NormalizePort) *Orchestrator {
	return &Orchestrator{resetRepo: resetRepo, ingest: ingest, normalize: normalize}
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

	// Ingestão histórica por mês e tipo
	for _, assetType := range p.AssetTypes {
		for _, dataType := range p.DataTypes {
			months := monthWindows(p.Start, p.End)
			for _, w := range months {
				// Chamar ingest com force conforme parâmetro
				ingSum, err := o.ingest.Ingest(ctx, appingest.IngestParams{
					TenantID:  p.TenantID,
					CPF:       p.CPF,
					DataType:  dataType,
					AssetType: assetType,
					Start:     w[0].Format("2006-01-02"),
					End:       w[1].Format("2006-01-02"),
					FetchAll:  true,
					Force:     p.Force,
					DryRun:    false,
				})
				if err != nil {
					observability.IncE2EError("ingest")
					helpers.LogError("e2e ingest failed", err, merge(logBase, map[string]any{"dataType": dataType, "assetType": assetType}))
					return nil, fmt.Errorf("ingest failed for %s/%s: %w", dataType, assetType, err)
				}
				sum.Raw.Saved += ingSum.Saved
				sum.Raw.Skipped += ingSum.Skipped
				sum.Raw.Errors += ingSum.Errors
				sum.Raw.MonthsProcessed += ingSum.MonthsProcessed
				sum.Raw.PagesProcessed += ingSum.PagesProcessed
			}

			// Normalização para o período inteiro deste dataType/assetType
			normSum, err := o.normalize.Run(ctx, appnorm.RunParams{
				TenantID:  p.TenantID,
				CPF:       p.CPF,
				DataType:  dataType,
				AssetType: assetType,
				Start:     p.Start,
				End:       p.End,
				Force:     p.Force,
				DryRun:    false,
			})
			if err != nil {
				observability.IncE2EError("normalize")
				helpers.LogError("e2e normalize failed", err, merge(logBase, map[string]any{"dataType": dataType, "assetType": assetType}))
				return nil, fmt.Errorf("normalize failed for %s/%s: %w", dataType, assetType, err)
			}
			sum.Normalized.Inserted += normSum.Inserted
			sum.Normalized.Updated += normSum.Updated
			sum.Normalized.Skipped += normSum.Skipped
			sum.Normalized.Errors += normSum.Errors

			observability.ObserveE2ERun("success", p.Mode, dataType, assetType, started)
			helpers.LogInfo("e2e stage done", merge(logBase, map[string]any{"stage": "normalize", "dataType": dataType, "assetType": assetType}))
		}
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
