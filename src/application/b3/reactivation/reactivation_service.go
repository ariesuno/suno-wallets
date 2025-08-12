package reactivation

import (
	"context"
	"fmt"
	"time"

	"suno-wallets/src/application/b3/incremental"
	"suno-wallets/src/application/b3/ingest"
	syncrepo "suno-wallets/src/infrastructure/b3/sync"
	"suno-wallets/src/shared/helpers"
)

// Comentários em pt-BR: serviço inteligente para reativação de clientes inativos

type Service struct {
	syncRepo       syncrepo.Repository
	ingestSvc      *ingest.Service
	incrementalSvc *incremental.Service
}

func NewService(syncRepo syncrepo.Repository, ingestSvc *ingest.Service, incrementalSvc *incremental.Service) *Service {
	return &Service{
		syncRepo:       syncRepo,
		ingestSvc:      ingestSvc,
		incrementalSvc: incrementalSvc,
	}
}

type ReactivationParams struct {
	TenantID    string
	CPF         string
	CurrentDate time.Time // Data atual de fechamento do pregão (ex: 2025-08-08)
}

type ReactivationStrategy string

const (
	StrategyFullHistorical  ReactivationStrategy = "FULL_HISTORICAL"  // Cliente novo - histórico completo
	StrategyIncrementalOnly ReactivationStrategy = "INCREMENTAL_ONLY" // Gap pequeno - só incremental
	StrategyHybridOptimized ReactivationStrategy = "HYBRID_OPTIMIZED" // Gap grande - mix de consolidado + incremental
)

type ReactivationPlan struct {
	Strategy         ReactivationStrategy `json:"strategy"`
	Reason           string               `json:"reason"`
	GapDays          int                  `json:"gapDays"`
	LastSyncDate     *time.Time           `json:"lastSyncDate,omitempty"`
	Steps            []ReactivationStep   `json:"steps"`
	EstimatedMinutes int                  `json:"estimatedMinutes"`
}

type ReactivationStep struct {
	Type        string    `json:"type"` // "historical_monthly", "incremental_daily"
	Description string    `json:"description"`
	FromDate    time.Time `json:"fromDate"`
	ToDate      time.Time `json:"toDate"`
	DataTypes   []string  `json:"dataTypes"`
}

const (
	B3_EPOCH_DATE            = "2019-11-01" // Data de início da API B3
	MAX_INCREMENTAL_GAP_DAYS = 30           // Máximo de dias para usar só incremental
	HYBRID_THRESHOLD_DAYS    = 90           // Acima disso, usar estratégia híbrida
)

// AnalyzeReactivation determina a estratégia otimizada para reativar um cliente
func (s *Service) AnalyzeReactivation(ctx context.Context, params ReactivationParams) (*ReactivationPlan, error) {
	plan := &ReactivationPlan{
		Steps: []ReactivationStep{},
	}

	// Buscar estado atual do cliente
	syncState, err := s.syncRepo.GetByTenantCPF(ctx, params.TenantID, params.CPF)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar estado de sync: %w", err)
	}

	// Determinar última data de sincronização (considerando transações e posições)
	var lastSync *time.Time
	if syncState != nil {
		if syncState.LastTxSyncAt != nil && syncState.LastPosSyncAt != nil {
			// Pegar a mais antiga entre as duas
			if syncState.LastTxSyncAt.Before(*syncState.LastPosSyncAt) {
				lastSync = syncState.LastTxSyncAt
			} else {
				lastSync = syncState.LastPosSyncAt
			}
		} else if syncState.LastTxSyncAt != nil {
			lastSync = syncState.LastTxSyncAt
		} else if syncState.LastPosSyncAt != nil {
			lastSync = syncState.LastPosSyncAt
		}
	}

	// ESTRATÉGIA 1: Cliente completamente novo
	if lastSync == nil {
		epochDate, _ := time.Parse("2006-01-02", B3_EPOCH_DATE)
		plan.Strategy = StrategyFullHistorical
		plan.Reason = "Cliente novo - sem histórico B3"
		plan.GapDays = int(params.CurrentDate.Sub(epochDate).Hours() / 24)
		plan.EstimatedMinutes = s.estimateHistoricalTime(epochDate, params.CurrentDate)

		plan.Steps = append(plan.Steps, ReactivationStep{
			Type:        "historical_monthly",
			Description: fmt.Sprintf("Processar histórico completo desde %s", B3_EPOCH_DATE),
			FromDate:    epochDate,
			ToDate:      params.CurrentDate,
			DataTypes:   []string{"transactions", "positions"},
		})

		helpers.LogInfo("reactivation plan: full historical", map[string]interface{}{
			"tenant_id":  params.TenantID,
			"cpf_masked": maskCPF(params.CPF),
			"gap_days":   plan.GapDays,
			"strategy":   plan.Strategy,
		})
		return plan, nil
	}

	// Calcular gap em dias
	gapDays := int(params.CurrentDate.Sub(*lastSync).Hours() / 24)
	plan.GapDays = gapDays
	plan.LastSyncDate = lastSync

	// ESTRATÉGIA 2: Gap pequeno - só incremental
	if gapDays <= MAX_INCREMENTAL_GAP_DAYS {
		plan.Strategy = StrategyIncrementalOnly
		plan.Reason = fmt.Sprintf("Gap pequeno (%d dias) - sincronização incremental otimizada", gapDays)
		plan.EstimatedMinutes = gapDays / 5 // ~5 dias por minuto para incremental

		plan.Steps = append(plan.Steps, ReactivationStep{
			Type:        "incremental_daily",
			Description: "Sincronização incremental dia-a-dia",
			FromDate:    lastSync.AddDate(0, 0, 1), // Dia seguinte
			ToDate:      params.CurrentDate,
			DataTypes:   []string{"transactions", "positions"},
		})

		helpers.LogInfo("reactivation plan: incremental only", map[string]interface{}{
			"tenant_id":  params.TenantID,
			"cpf_masked": maskCPF(params.CPF),
			"gap_days":   gapDays,
			"strategy":   plan.Strategy,
		})
		return plan, nil
	}

	// ESTRATÉGIA 3: Gap grande - estratégia híbrida otimizada
	if gapDays > HYBRID_THRESHOLD_DAYS {
		plan.Strategy = StrategyHybridOptimized
		plan.Reason = fmt.Sprintf("Gap grande (%d dias) - mix de consolidado mensal + incremental final", gapDays)

		// Dividir em: meses completos (consolidado) + dias finais (incremental)
		monthsEnd := time.Date(params.CurrentDate.Year(), params.CurrentDate.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)

		if lastSync.Before(monthsEnd) {
			// Fase 1: Meses completos via ingestão histórica
			plan.Steps = append(plan.Steps, ReactivationStep{
				Type:        "historical_monthly",
				Description: fmt.Sprintf("Consolidado mensal de %s até %s", lastSync.Format("2006-01-02"), monthsEnd.Format("2006-01-02")),
				FromDate:    lastSync.AddDate(0, 0, 1),
				ToDate:      monthsEnd,
				DataTypes:   []string{"transactions", "positions"},
			})
		}

		// Fase 2: Dias do mês atual via incremental
		monthStart := time.Date(params.CurrentDate.Year(), params.CurrentDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		if monthStart.Before(params.CurrentDate) {
			plan.Steps = append(plan.Steps, ReactivationStep{
				Type:        "incremental_daily",
				Description: fmt.Sprintf("Incremental do mês atual: %s até %s", monthStart.Format("2006-01-02"), params.CurrentDate.Format("2006-01-02")),
				FromDate:    monthStart,
				ToDate:      params.CurrentDate,
				DataTypes:   []string{"transactions", "positions"},
			})
		}

		plan.EstimatedMinutes = s.estimateHybridTime(gapDays)

		helpers.LogInfo("reactivation plan: hybrid optimized", map[string]interface{}{
			"tenant_id":  params.TenantID,
			"cpf_masked": maskCPF(params.CPF),
			"gap_days":   gapDays,
			"strategy":   plan.Strategy,
			"steps":      len(plan.Steps),
		})
		return plan, nil
	}

	// ESTRATÉGIA PADRÃO: Gap médio - só incremental
	plan.Strategy = StrategyIncrementalOnly
	plan.Reason = fmt.Sprintf("Gap médio (%d dias) - sincronização incremental", gapDays)
	plan.EstimatedMinutes = gapDays / 3 // ~3 dias por minuto

	plan.Steps = append(plan.Steps, ReactivationStep{
		Type:        "incremental_daily",
		Description: "Sincronização incremental otimizada",
		FromDate:    lastSync.AddDate(0, 0, 1),
		ToDate:      params.CurrentDate,
		DataTypes:   []string{"transactions", "positions"},
	})

	helpers.LogInfo("reactivation plan: default incremental", map[string]interface{}{
		"tenant_id":  params.TenantID,
		"cpf_masked": maskCPF(params.CPF),
		"gap_days":   gapDays,
		"strategy":   plan.Strategy,
	})

	return plan, nil
}

// ExecuteReactivation executa o plano de reativação
func (s *Service) ExecuteReactivation(ctx context.Context, params ReactivationParams, dryRun bool) (*ReactivationPlan, error) {
	plan, err := s.AnalyzeReactivation(ctx, params)
	if err != nil {
		return nil, err
	}

	if dryRun {
		plan.Reason += " (DRY RUN - não executado)"
		return plan, nil
	}

	// Executar cada step do plano
	for i, step := range plan.Steps {
		helpers.LogInfo("executing reactivation step", map[string]interface{}{
			"tenant_id":   params.TenantID,
			"cpf_masked":  maskCPF(params.CPF),
			"step":        i + 1,
			"total_steps": len(plan.Steps),
			"type":        step.Type,
		})

		switch step.Type {
		case "historical_monthly":
			_, err = s.ingestSvc.Ingest(ctx, ingest.IngestParams{
				TenantID:  params.TenantID,
				CPF:       params.CPF,
				DataType:  "transactions", // Fazer para cada tipo
				AssetType: "equity",
				Start:     step.FromDate.Format("2006-01-02"),
				End:       step.ToDate.Format("2006-01-02"),
				FetchAll:  true,
				Force:     false,
				DryRun:    false,
			})
			if err != nil {
				return plan, fmt.Errorf("erro na ingestão histórica: %w", err)
			}

		case "incremental_daily":
			_, err = s.incrementalSvc.Run(ctx, incremental.Params{
				TenantID:   params.TenantID,
				CPF:        params.CPF,
				DataTypes:  step.DataTypes,
				AssetTypes: []string{"equity"},
				Since:      &step.FromDate,
				End:        &step.ToDate,
				Force:      false,
				DryRun:     false,
			})
			if err != nil {
				return plan, fmt.Errorf("erro na sincronização incremental: %w", err)
			}
		}
	}

	helpers.LogInfo("reactivation completed", map[string]interface{}{
		"tenant_id":       params.TenantID,
		"cpf_masked":      maskCPF(params.CPF),
		"strategy":        plan.Strategy,
		"steps_completed": len(plan.Steps),
	})

	return plan, nil
}

// Funções auxiliares de estimativa
func (s *Service) estimateHistoricalTime(from, to time.Time) int {
	months := int(to.Sub(from).Hours() / (24 * 30))
	return months * 2 // ~2 minutos por mês
}

func (s *Service) estimateHybridTime(gapDays int) int {
	months := gapDays / 30
	return (months * 2) + (gapDays%30)/5 // Meses + dias incrementais
}

func maskCPF(cpf string) string {
	if len(cpf) < 3 {
		return "***"
	}
	return cpf[:3] + "********"
}
