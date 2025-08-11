package reconciliation

import (
	"context"
	"time"

	obs "suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"

	"github.com/google/uuid"
)

// Comentários em pt-BR: Service orquestra varredura de inconsistências com regras set-based

type ScanRequest struct {
	CPF               string
	Tickers           []string
	From              *time.Time
	To                *time.Time
	DryRun            bool
	MaxSamplesPerType int
	Concurrency       int
}

type DetectorRepository interface {
	ScanOpeningBalanceMissing(ctx context.Context, tenantID uuid.UUID, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]Finding, error)
	ScanSellWithoutBuy(ctx context.Context, tenantID uuid.UUID, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]Finding, error)
	ScanPositionTxDivergence(ctx context.Context, tenantID uuid.UUID, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]Finding, error)
	TryAcquireLock(ctx context.Context, tenantID uuid.UUID, cpf string, ttlSeconds int) (bool, error)
	ReleaseLock(ctx context.Context, tenantID uuid.UUID, cpf string) error
	UpsertFindings(ctx context.Context, tenantID uuid.UUID, cpf string, findings []Finding) error
	ListInconsistencies(ctx context.Context, tenantID uuid.UUID, cpf, status, typ, ticker string, from, to *time.Time, page, pageSize int) ([]Inconsistency, error)
	GetInconsistency(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*Inconsistency, error)
}

type Finding struct {
	Ticker     string
	Type       string
	Severity   int
	From       *time.Time
	To         *time.Time
	Samples    map[string]interface{}
	Details    map[string]interface{}
	DedupeHash string
}

type Service struct{ repo DetectorRepository }

func NewService(repo DetectorRepository) *Service { return &Service{repo: repo} }

func (s *Service) Scan(ctx context.Context, tenantID uuid.UUID, req ScanRequest) (map[string]any, error) {
	started := time.Now()
	totals := map[string]int{"OPENING_BALANCE_MISSING": 0, "SELL_WITHOUT_BUY": 0, "POSITION_TX_DIVERGENCE": 0}
	log := helpers.GetLoggerWithFields(map[string]interface{}{
		"service":   "inconsistency_detector",
		"tenantId":  tenantID,
		"cpfMasked": maskCPF(req.CPF),
		"dryRun":    req.DryRun,
		"tickers":   req.Tickers,
		"from":      req.From,
		"to":        req.To,
	})

	maxSamples := req.MaxSamplesPerType
	if maxSamples <= 0 {
		maxSamples = 5
	}

	var all []Finding

	// Lock por (tenant, cpf) para evitar corrida
	locked, errLock := s.repo.TryAcquireLock(ctx, tenantID, req.CPF, 120)
	if errLock != nil || !locked {
		return map[string]any{"error": "LOCK_NOT_ACQUIRED"}, errLock
	}
	defer func() { _ = s.repo.ReleaseLock(ctx, tenantID, req.CPF) }()

	// Varre cada regra
	if f, err := s.repo.ScanOpeningBalanceMissing(ctx, tenantID, req.CPF, req.Tickers, req.From, req.To, maxSamples); err != nil {
		log.Error("scan_opening_balance_missing_failed", err, nil)
	} else {
		all = append(all, f...)
		totals["OPENING_BALANCE_MISSING"] += len(f)
	}

	if f, err := s.repo.ScanSellWithoutBuy(ctx, tenantID, req.CPF, req.Tickers, req.From, req.To, maxSamples); err != nil {
		log.Error("scan_sell_without_buy_failed", err, nil)
	} else {
		all = append(all, f...)
		totals["SELL_WITHOUT_BUY"] += len(f)
	}

	if f, err := s.repo.ScanPositionTxDivergence(ctx, tenantID, req.CPF, req.Tickers, req.From, req.To, maxSamples); err != nil {
		log.Error("scan_position_tx_divergence_failed", err, nil)
	} else {
		all = append(all, f...)
		totals["POSITION_TX_DIVERGENCE"] += len(f)
	}

	// Persistência (idempotente) se não for dry-run
	if !req.DryRun && len(all) > 0 {
		if err := s.repo.UpsertFindings(ctx, tenantID, req.CPF, all); err != nil {
			log.Error("persist_findings_failed", err, map[string]interface{}{"count": len(all)})
		}
	}

	// Métricas
	obs.IncReconFound("OPENING_BALANCE_MISSING", totals["OPENING_BALANCE_MISSING"])
	obs.IncReconFound("SELL_WITHOUT_BUY", totals["SELL_WITHOUT_BUY"])
	obs.IncReconFound("POSITION_TX_DIVERGENCE", totals["POSITION_TX_DIVERGENCE"])
	obs.ObserveReconScanRun("success", started)

	// Amostras por tipo para resposta
	responseSamples := map[string][]Finding{"OPENING_BALANCE_MISSING": {}, "SELL_WITHOUT_BUY": {}, "POSITION_TX_DIVERGENCE": {}}
	for _, f := range all {
		lst := responseSamples[f.Type]
		if len(lst) < maxSamples {
			responseSamples[f.Type] = append(lst, f)
		}
	}
	out := map[string]any{
		"totals":        totals,
		"samplesByType": responseSamples,
		"durationMs":    time.Since(started).Milliseconds(),
	}
	log.Info("recon_scan_finished", map[string]interface{}{
		"durationMs":    out["durationMs"],
		"found_by_type": totals,
	})
	return out, nil
}

func maskCPF(cpf string) string {
	if len(cpf) < 3 {
		return "***"
	}
	return "***" + cpf[len(cpf)-3:]
}

// Tipos de leitura para listagem/detalhes
type Inconsistency struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	CPF       string
	Ticker    string
	Type      string
	Status    string
	Severity  int
	UpdatedAt time.Time
	// Campos adicionais para o GET por id
	FirstDetectedAt time.Time
	LastDetectedAt  time.Time
	SampleDates     map[string]interface{}
	Details         map[string]interface{}
}

func (s *Service) List(ctx context.Context, tenantID uuid.UUID, cpf, status, typ, ticker string, from, to *time.Time, page, pageSize int) ([]Inconsistency, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	return s.repo.ListInconsistencies(ctx, tenantID, cpf, status, typ, ticker, from, to, page, pageSize)
}

func (s *Service) Get(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*Inconsistency, error) {
	return s.repo.GetInconsistency(ctx, tenantID, id)
}
