package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	appE2E "suno-wallets/src/application/b3/e2e"
	"suno-wallets/src/application/b3/incremental"
	"suno-wallets/src/application/b3/reactivation"
	completesync "suno-wallets/src/application/b3/sync"
)

// Mock para Reactivation Service
type MockReactivationService struct {
	mock.Mock
}

func (m *MockReactivationService) AnalyzeReactivation(ctx context.Context, params reactivation.ReactivationParams) (*reactivation.ReactivationPlan, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*reactivation.ReactivationPlan), args.Error(1)
}

func (m *MockReactivationService) ExecuteReactivation(ctx context.Context, params reactivation.ReactivationParams, dryRun bool) (*reactivation.ReactivationPlan, error) {
	args := m.Called(ctx, params, dryRun)
	return args.Get(0).(*reactivation.ReactivationPlan), args.Error(1)
}

// Mock para E2E Orchestrator
type MockE2EOrchestrator struct {
	mock.Mock
}

func (m *MockE2EOrchestrator) Run(ctx context.Context, params appE2E.Params) (*appE2E.Summary, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*appE2E.Summary), args.Error(1)
}

// Mock para Incremental Service
type MockIncrementalService struct {
	mock.Mock
}

func (m *MockIncrementalService) Run(ctx context.Context, params incremental.Params) (*incremental.Summary, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*incremental.Summary), args.Error(1)
}

// Comentários em pt-BR: Testes unitários para Complete Sync Orchestrator

func TestCompleteSyncOrchestrator_NewClient_FullHistorical(t *testing.T) {
	// Setup mocks
	mockReactivation := new(MockReactivationService)
	mockE2E := new(MockE2EOrchestrator)
	mockIncremental := new(MockIncrementalService)

	orchestrator := completesync.NewCompleteSyncOrchestrator(
		mockReactivation,
		mockE2E,
		mockIncremental,
		nil, // reconciliation service
		nil, // sync repo
	)

	// Configurar mocks para cliente novo
	reactivationPlan := &reactivation.ReactivationPlan{
		Strategy:         reactivation.StrategyFullHistorical,
		Reason:           "Cliente novo - sem histórico B3",
		GapDays:          1500,
		LastSyncDate:     nil,
		EstimatedMinutes: 120,
	}

	e2eResult := &appE2E.Summary{
		Raw: appE2E.RawSummary{
			Saved:            1500,
			Skipped:          0,
			Errors:           0,
			MonthsProcessed:  24,
			PagesProcessed:   150,
		},
		Normalized: appE2E.NormalizedSummary{
			Inserted: 800,
			Updated:  0,
			Skipped:  0,
			Errors:   0,
		},
		StartedAt:  time.Now(),
		FinishedAt: time.Now().Add(30 * time.Minute),
		DurationMs: 1800000, // 30 minutos
	}

	mockReactivation.On("AnalyzeReactivation", mock.Anything, mock.MatchedBy(func(params reactivation.ReactivationParams) bool {
		return params.TenantID == "status_invest" && params.CPF == "12345678901"
	})).Return(reactivationPlan, nil)

	mockE2E.On("Run", mock.Anything, mock.MatchedBy(func(params appE2E.Params) bool {
		return params.TenantID == "status_invest" && params.CPF == "12345678901" && params.RequestedBy == "complete_sync_orchestrator"
	})).Return(e2eResult, nil)

	// Executar teste
	params := completesync.CompleteSyncParams{
		TenantID:              "status_invest",
		CPF:                   "12345678901",
		AssetTypes:            []string{"equity"},
		DataTypes:             []string{"transactions", "positions"},
		IncludeReconciliation: false,
		DryRun:                false,
		Force:                 false,
	}

	result, err := orchestrator.ExecuteCompleteSync(context.Background(), params)

	// Verificar resultado
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "FULL_HISTORICAL", result.Strategy)
	assert.Equal(t, "new", result.ClientStatus)
	assert.Equal(t, "complete_historical_ingestion", result.ExecutionPlan)
	assert.True(t, result.NewDataIngested)
	assert.Equal(t, 1500, result.DataSummary.RawRecordsIngested)
	assert.Equal(t, 800, result.DataSummary.TransactionsNormalized)
	assert.NotNil(t, result.E2EResult)

	// Verificar que mocks foram chamados
	mockReactivation.AssertExpectations(t)
	mockE2E.AssertExpectations(t)
}

func TestCompleteSyncOrchestrator_ExistingClient_Incremental(t *testing.T) {
	// Setup mocks
	mockReactivation := new(MockReactivationService)
	mockE2E := new(MockE2EOrchestrator)
	mockIncremental := new(MockIncrementalService)

	orchestrator := completesync.NewCompleteSyncOrchestrator(
		mockReactivation,
		mockE2E,
		mockIncremental,
		nil, // reconciliation service
		nil, // sync repo
	)

	// Configurar mocks para cliente existente com gap pequeno
	lastSync := time.Now().AddDate(0, 0, -5) // 5 dias atrás
	reactivationPlan := &reactivation.ReactivationPlan{
		Strategy:         reactivation.StrategyIncrementalOnly,
		Reason:           "Gap pequeno (5 dias) - sincronização incremental otimizada",
		GapDays:          5,
		LastSyncDate:     &lastSync,
		EstimatedMinutes: 1,
	}

	incrementalResult := &incremental.Summary{
		Transactions: &incremental.TypeSummary{
			From: "2024-01-10",
			To:   "2024-01-15",
			Normalized: incremental.NormalizeSummary{
				Inserted: 5,
				Updated:  0,
				Skipped:  0,
				Errors:   0,
			},
		},
		StartedAt:  time.Now(),
		FinishedAt: time.Now().Add(1 * time.Minute),
		DurationMs: 60000, // 1 minuto
	}

	mockReactivation.On("AnalyzeReactivation", mock.Anything, mock.MatchedBy(func(params reactivation.ReactivationParams) bool {
		return params.TenantID == "status_invest" && params.CPF == "98765432100"
	})).Return(reactivationPlan, nil)

	mockIncremental.On("Run", mock.Anything, mock.MatchedBy(func(params incremental.Params) bool {
		return params.TenantID == "status_invest" && params.CPF == "98765432100"
	})).Return(incrementalResult, nil)

	// Executar teste
	params := completesync.CompleteSyncParams{
		TenantID:              "status_invest",
		CPF:                   "98765432100",
		AssetTypes:            []string{"equity"},
		DataTypes:             []string{"transactions", "positions"},
		IncludeReconciliation: false,
		DryRun:                false,
		Force:                 false,
	}

	result, err := orchestrator.ExecuteCompleteSync(context.Background(), params)

	// Verificar resultado
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "INCREMENTAL_ONLY", result.Strategy)
	assert.Equal(t, "existing_current", result.ClientStatus)
	assert.Equal(t, "incremental_sync_ingestion", result.ExecutionPlan)
	assert.True(t, result.NewDataIngested)
	assert.Equal(t, 5, result.DataSummary.TransactionsNormalized)
	assert.NotNil(t, result.IncrementalResult)

	// Verificar que mocks foram chamados
	mockReactivation.AssertExpectations(t)
	mockIncremental.AssertExpectations(t)
}

func TestCompleteSyncOrchestrator_DryRun(t *testing.T) {
	// Setup mocks
	mockReactivation := new(MockReactivationService)
	mockE2E := new(MockE2EOrchestrator)
	mockIncremental := new(MockIncrementalService)

	orchestrator := completesync.NewCompleteSyncOrchestrator(
		mockReactivation,
		mockE2E,
		mockIncremental,
		nil, // reconciliation service
		nil, // sync repo
	)

	// Configurar mock para dry run
	reactivationPlan := &reactivation.ReactivationPlan{
		Strategy:         reactivation.StrategyFullHistorical,
		Reason:           "Cliente novo - sem histórico B3",
		GapDays:          1500,
		LastSyncDate:     nil,
		EstimatedMinutes: 120,
	}

	mockReactivation.On("AnalyzeReactivation", mock.Anything, mock.Anything).Return(reactivationPlan, nil)

	// Executar teste com dry run
	params := completesync.CompleteSyncParams{
		TenantID:              "status_invest",
		CPF:                   "12345678901",
		AssetTypes:            []string{"equity"},
		DataTypes:             []string{"transactions", "positions"},
		IncludeReconciliation: false,
		DryRun:                true, // Dry run ativo
		Force:                 false,
	}

	result, err := orchestrator.ExecuteCompleteSync(context.Background(), params)

	// Verificar resultado
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "FULL_HISTORICAL", result.Strategy)
	assert.Equal(t, "new", result.ClientStatus)
	assert.Equal(t, "complete_historical_ingestion_dry_run", result.ExecutionPlan)
	assert.False(t, result.NewDataIngested) // Dry run não ingere dados
	assert.Nil(t, result.E2EResult)         // Sem execução real

	// Verificar que apenas o reactivation foi chamado (análise)
	mockReactivation.AssertExpectations(t)
	// E2E e Incremental não devem ter sido chamados no dry run
	mockE2E.AssertNotCalled(t, "Run")
	mockIncremental.AssertNotCalled(t, "Run")
}

func TestCompleteSyncOrchestrator_ReactivationPlanValidation(t *testing.T) {
	// Setup mocks
	mockReactivation := new(MockReactivationService)
	mockE2E := new(MockE2EOrchestrator)
	mockIncremental := new(MockIncrementalService)

	orchestrator := completesync.NewCompleteSyncOrchestrator(
		mockReactivation,
		mockE2E,
		mockIncremental,
		nil, // reconciliation service
		nil, // sync repo
	)

	// Teste diferentes estratégias de reativação
	testCases := []struct {
		name           string
		strategy       reactivation.ReactivationStrategy
		expectedStatus string
		expectedPlan   string
		gapDays        int
	}{
		{
			name:           "New Client",
			strategy:       reactivation.StrategyFullHistorical,
			expectedStatus: "new",
			expectedPlan:   "complete_historical_ingestion_dry_run",
			gapDays:        1500,
		},
		{
			name:           "Current Client",
			strategy:       reactivation.StrategyIncrementalOnly,
			expectedStatus: "existing_current",
			expectedPlan:   "incremental_sync_ingestion_dry_run",
			gapDays:        5,
		},
		{
			name:           "Outdated Client",
			strategy:       reactivation.StrategyHybridOptimized,
			expectedStatus: "existing_outdated",
			expectedPlan:   "hybrid_reactivation_ingestion_dry_run",
			gapDays:        120,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mocks para cada teste
			mockReactivation.ExpectedCalls = nil

			reactivationPlan := &reactivation.ReactivationPlan{
				Strategy:         tc.strategy,
				Reason:           "Test strategy",
				GapDays:          tc.gapDays,
				EstimatedMinutes: 10,
			}

			mockReactivation.On("AnalyzeReactivation", mock.Anything, mock.Anything).Return(reactivationPlan, nil)

			params := completesync.CompleteSyncParams{
				TenantID:   "status_invest",
				CPF:        "12345678901",
				AssetTypes: []string{"equity"},
				DataTypes:  []string{"transactions", "positions"},
				DryRun:     true, // Sempre dry run para teste de validação
			}

			result, err := orchestrator.ExecuteCompleteSync(context.Background(), params)

			assert.NoError(t, err)
			assert.True(t, result.Success)
			assert.Equal(t, string(tc.strategy), result.Strategy)
			assert.Equal(t, tc.expectedStatus, result.ClientStatus)
			assert.Equal(t, tc.expectedPlan, result.ExecutionPlan)

			mockReactivation.AssertExpectations(t)
		})
	}
}
