package sync

import (
	"context"
	"fmt"
	"time"

	appE2E "suno-wallets/src/application/b3/e2e"
	"suno-wallets/src/application/b3/incremental"
	"suno-wallets/src/application/b3/reactivation"
	apprecon "suno-wallets/src/application/b3/reconciliation"
	syncrepo "suno-wallets/src/infrastructure/b3/sync"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"
)

// Comentários em pt-BR: Orchestrador para sincronização completa e inteligente B3

type CompleteSyncOrchestrator struct {
	reactivationSvc *reactivation.Service
	e2eOrchestrator *appE2E.Orchestrator
	incrementalSvc  *incremental.Service
	reconcileSvc    *apprecon.Service
	syncRepo        syncrepo.Repository
}

func NewCompleteSyncOrchestrator(
	reactivationSvc *reactivation.Service,
	e2eOrchestrator *appE2E.Orchestrator,
	incrementalSvc *incremental.Service,
	reconcileSvc *apprecon.Service,
	syncRepo syncrepo.Repository,
) *CompleteSyncOrchestrator {
	return &CompleteSyncOrchestrator{
		reactivationSvc: reactivationSvc,
		e2eOrchestrator: e2eOrchestrator,
		incrementalSvc:  incrementalSvc,
		reconcileSvc:    reconcileSvc,
		syncRepo:        syncRepo,
	}
}

type CompleteSyncParams struct {
	TenantID              string   `json:"tenantId"`
	CPF                   string   `json:"cpf" binding:"required"`
	AssetTypes            []string `json:"assetTypes,omitempty"`
	DataTypes             []string `json:"dataTypes,omitempty"`
	IncludeReconciliation bool     `json:"includeReconciliation,omitempty"`
	DryRun                bool     `json:"dryRun,omitempty"`
	Force                 bool     `json:"force,omitempty"`
}

type CompleteSyncResult struct {
	Strategy             string                         `json:"strategy"`
	ClientStatus         string                         `json:"clientStatus"` // new, existing_current, existing_outdated
	ExecutionPlan        string                         `json:"executionPlan"`
	ReactivationPlan     *reactivation.ReactivationPlan `json:"reactivationPlan,omitempty"`
	E2EResult            *appE2E.Summary                `json:"e2eResult,omitempty"`
	IncrementalResult    *incremental.Summary           `json:"incrementalResult,omitempty"`
	ReconciliationResult map[string]interface{}         `json:"reconciliationResult,omitempty"`
	StartedAt            time.Time                      `json:"startedAt"`
	FinishedAt           time.Time                      `json:"finishedAt"`
	DurationMs           int64                          `json:"durationMs"`
	Success              bool                           `json:"success"`
	ErrorMessage         string                         `json:"errorMessage,omitempty"`
	ProcessedDataTypes   []string                       `json:"processedDataTypes"`
	NewDataIngested      bool                           `json:"newDataIngested"`
	DataSummary          DataSummary                    `json:"dataSummary"`
}

type DataSummary struct {
	RawRecordsIngested     int `json:"rawRecordsIngested"`
	TransactionsNormalized int `json:"transactionsNormalized"`
	PositionsNormalized    int `json:"positionsNormalized"`
	InconsistenciesFound   int `json:"inconsistenciesFound,omitempty"`
}

// ExecuteCompleteSync executa sincronização completa e inteligente para um cliente
func (s *CompleteSyncOrchestrator) ExecuteCompleteSync(ctx context.Context, params CompleteSyncParams) (*CompleteSyncResult, error) {
	started := time.Now()
	result := &CompleteSyncResult{
		StartedAt:          started,
		ProcessedDataTypes: params.DataTypes,
		Success:            true,
		DataSummary:        DataSummary{},
	}

	logBase := map[string]interface{}{
		"tenantId":   params.TenantID,
		"cpfMasked":  maskCPF(params.CPF),
		"dataTypes":  params.DataTypes,
		"assetTypes": params.AssetTypes,
		"dryRun":     params.DryRun,
	}

	helpers.LogInfo("complete sync orchestrator started", logBase)
	observability.IncCompleteSyncOrchestrator("started")

	// Passo 1: Analisar situação do cliente usando Reactivation Service
	reactivationPlan, err := s.reactivationSvc.AnalyzeReactivation(ctx, reactivation.ReactivationParams{
		TenantID:    params.TenantID,
		CPF:         params.CPF,
		CurrentDate: time.Now(),
	})
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Failed to analyze client sync status: %v", err)
		result.FinishedAt = time.Now()
		result.DurationMs = time.Since(started).Milliseconds()
		observability.IncCompleteSyncOrchestrator("error")
		return result, err
	}

	result.ReactivationPlan = reactivationPlan
	result.Strategy = string(reactivationPlan.Strategy)

	// Passo 2: Determinar status do cliente e plano de execução
	switch reactivationPlan.Strategy {
	case reactivation.StrategyFullHistorical:
		result.ClientStatus = "new"
		result.ExecutionPlan = "complete_historical_ingestion"

		if params.DryRun {
			result.ExecutionPlan += "_dry_run"
			helpers.LogInfo("complete sync: would execute full historical ingestion (dry run)", logBase)
		} else {
			// Cliente novo - usar E2E Orchestrator para ingestão histórica completa
			e2eResult, err := s.executeFullHistoricalIngestion(ctx, params, logBase)
			if err != nil {
				result.Success = false
				result.ErrorMessage = fmt.Sprintf("Full historical ingestion failed: %v", err)
				observability.IncCompleteSyncOrchestrator("error")
			} else {
				result.E2EResult = e2eResult
				result.NewDataIngested = e2eResult.Raw.Saved > 0
				result.DataSummary.RawRecordsIngested = e2eResult.Raw.Saved
				result.DataSummary.TransactionsNormalized = e2eResult.Normalized.Inserted
				result.DataSummary.PositionsNormalized = e2eResult.Normalized.Updated
				helpers.LogInfo("complete sync: historical ingestion completed", merge(logBase, map[string]interface{}{
					"rawIngested":     e2eResult.Raw.Saved,
					"transNormalized": e2eResult.Normalized.Inserted,
				}))
			}
		}

	case reactivation.StrategyIncrementalOnly:
		result.ClientStatus = "existing_current"
		result.ExecutionPlan = "incremental_sync_ingestion"

		if params.DryRun {
			result.ExecutionPlan += "_dry_run"
			helpers.LogInfo("complete sync: would execute incremental ingestion (dry run)", logBase)
		} else {
			// Cliente com gap pequeno - usar Incremental Service
			incrResult, err := s.executeIncrementalIngestion(ctx, params, reactivationPlan, logBase)
			if err != nil {
				result.Success = false
				result.ErrorMessage = fmt.Sprintf("Incremental ingestion failed: %v", err)
				observability.IncCompleteSyncOrchestrator("error")
			} else {
				result.IncrementalResult = incrResult
				result.NewDataIngested = s.hasNewDataFromIncremental(incrResult)
				s.extractDataSummaryFromIncremental(incrResult, &result.DataSummary)
				helpers.LogInfo("complete sync: incremental ingestion completed", merge(logBase, map[string]interface{}{
					"newDataIngested": result.NewDataIngested,
				}))
			}
		}

	case reactivation.StrategyHybridOptimized:
		result.ClientStatus = "existing_outdated"
		result.ExecutionPlan = "hybrid_reactivation_ingestion"

		if params.DryRun {
			result.ExecutionPlan += "_dry_run"
			helpers.LogInfo("complete sync: would execute hybrid reactivation (dry run)", logBase)
		} else {
			// Cliente com gap grande - usar Reactivation Service completo
			reactivationResult, err := s.executeHybridReactivation(ctx, params, logBase)
			if err != nil {
				result.Success = false
				result.ErrorMessage = fmt.Sprintf("Hybrid reactivation failed: %v", err)
				observability.IncCompleteSyncOrchestrator("error")
			} else {
				result.ReactivationPlan = reactivationResult
				result.NewDataIngested = true // Reactivation sempre implica novos dados
				helpers.LogInfo("complete sync: hybrid reactivation completed", logBase)
			}
		}

	default:
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Unknown sync strategy: %s", reactivationPlan.Strategy)
		observability.IncCompleteSyncOrchestrator("error")
	}

	// Passo 3: Reconciliação (opcional) - apenas se houve ingestão de novos dados
	if result.Success && params.IncludeReconciliation && result.NewDataIngested && !params.DryRun {
		reconResult, err := s.executeReconciliation(ctx, params, logBase)
		if err != nil {
			// Reconciliação é opcional - logar mas não falhar todo o processo
			helpers.LogError("complete sync: reconciliation failed", err, logBase)
			result.ReconciliationResult = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
		} else {
			result.ReconciliationResult = reconResult
			if inconsistencies, ok := reconResult["inconsistenciesFound"].(int); ok {
				result.DataSummary.InconsistenciesFound = inconsistencies
			}
			helpers.LogInfo("complete sync: reconciliation completed", logBase)
		}
	}

	// Finalizar resultado
	result.FinishedAt = time.Now()
	result.DurationMs = time.Since(started).Milliseconds()

	if result.Success {
		observability.IncCompleteSyncOrchestrator("success")
		helpers.LogInfo("complete sync orchestrator finished successfully", merge(logBase, map[string]interface{}{
			"strategy":        result.Strategy,
			"executionPlan":   result.ExecutionPlan,
			"newDataIngested": result.NewDataIngested,
			"durationMs":      result.DurationMs,
		}))
	}

	return result, nil
}

// executeFullHistoricalIngestion executa ingestão histórica completa para cliente novo
func (s *CompleteSyncOrchestrator) executeFullHistoricalIngestion(ctx context.Context, params CompleteSyncParams, logBase map[string]interface{}) (*appE2E.Summary, error) {
	epochDate, _ := time.Parse("2006-01-02", "2019-11-01") // Data de início da API B3
	// Para transações, processar até D-1 (ontem) para incluir mês corrente completo
	yesterday := time.Now().AddDate(0, 0, -1)

	return s.e2eOrchestrator.Run(ctx, appE2E.Params{
		TenantID:    params.TenantID,
		CPF:         params.CPF,
		AssetTypes:  params.AssetTypes,
		DataTypes:   params.DataTypes,
		Start:       epochDate,
		End:         yesterday,
		Force:       params.Force,
		DryRun:      false, // Já validado no nível superior
		Mode:        "archive",
		RequestedBy: "complete_sync_orchestrator",
	})
}

// executeIncrementalIngestion executa ingestão incremental
func (s *CompleteSyncOrchestrator) executeIncrementalIngestion(ctx context.Context, params CompleteSyncParams, plan *reactivation.ReactivationPlan, logBase map[string]interface{}) (*incremental.Summary, error) {
	var since *time.Time
	if plan.LastSyncDate != nil {
		nextDay := plan.LastSyncDate.AddDate(0, 0, 1)
		since = &nextDay
	}

	return s.incrementalSvc.Run(ctx, incremental.Params{
		TenantID:   params.TenantID,
		CPF:        params.CPF,
		DataTypes:  params.DataTypes,
		AssetTypes: params.AssetTypes,
		Since:      since,
		End:        nil, // Até hoje
		Force:      params.Force,
		DryRun:     false,
	})
}

// executeHybridReactivation executa reativação híbrida inteligente
func (s *CompleteSyncOrchestrator) executeHybridReactivation(ctx context.Context, params CompleteSyncParams, logBase map[string]interface{}) (*reactivation.ReactivationPlan, error) {
	return s.reactivationSvc.ExecuteReactivation(ctx, reactivation.ReactivationParams{
		TenantID:    params.TenantID,
		CPF:         params.CPF,
		CurrentDate: time.Now(),
	}, false)
}

// executeReconciliation executa reconciliação dos dados ingeridos
func (s *CompleteSyncOrchestrator) executeReconciliation(ctx context.Context, params CompleteSyncParams, logBase map[string]interface{}) (map[string]interface{}, error) {
	if s.reconcileSvc == nil {
		return map[string]interface{}{
			"status": "skipped",
			"reason": "reconciliation service not available",
		}, nil
	}

	result, err := s.reconcileSvc.Scan(ctx, params.TenantID, apprecon.ScanRequest{
		CPF:         params.CPF,
		Tickers:     nil, // Deixar vazio para varrer todos os tickers
		From:        nil,
		To:          nil,
		DryRun:      false,
		Concurrency: 2,
	})
	if err != nil {
		return nil, err
	}

	inconsistenciesCount := 0
	if created, ok := result["created"].(int); ok {
		inconsistenciesCount = created
	}

	return map[string]interface{}{
		"status":               "completed",
		"inconsistenciesFound": inconsistenciesCount,
		"scanResult":           result,
	}, nil
}

// hasNewDataFromIncremental verifica se houve novos dados no resultado incremental
func (s *CompleteSyncOrchestrator) hasNewDataFromIncremental(result *incremental.Summary) bool {
	if result.Transactions != nil && result.Transactions.Normalized.Inserted > 0 {
		return true
	}
	if result.Positions != nil && result.Positions.Normalized.Inserted > 0 {
		return true
	}
	return false
}

// extractDataSummaryFromIncremental extrai resumo de dados do resultado incremental
func (s *CompleteSyncOrchestrator) extractDataSummaryFromIncremental(result *incremental.Summary, summary *DataSummary) {
	if result.Transactions != nil {
		summary.TransactionsNormalized = result.Transactions.Normalized.Inserted
	}
	if result.Positions != nil {
		summary.PositionsNormalized = result.Positions.Normalized.Inserted
	}
	// Raw data é contabilizado através dos normalized results
	summary.RawRecordsIngested = summary.TransactionsNormalized + summary.PositionsNormalized
}

// Funções auxiliares
func maskCPF(cpf string) string {
	if len(cpf) < 3 {
		return "***"
	}
	return cpf[:3] + "********"
}

func merge(base map[string]interface{}, additional map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range base {
		result[k] = v
	}
	for k, v := range additional {
		result[k] = v
	}
	return result
}
