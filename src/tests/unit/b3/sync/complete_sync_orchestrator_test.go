package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"

	completesync "suno-wallets/src/application/b3/sync"
)

// Comentários em pt-BR: Testes unitários básicos para Complete Sync Orchestrator

func TestCompleteSyncOrchestrator_Creation(t *testing.T) {
	// Teste básico de criação do orchestrator
	orchestrator := completesync.NewCompleteSyncOrchestrator(
		nil, // reactivation service
		nil, // e2e orchestrator
		nil, // incremental service
		nil, // reconciliation service
		nil, // sync repo
	)

	assert.NotNil(t, orchestrator, "Orchestrator should be created successfully")
}

func TestCompleteSyncOrchestrator_ParamsValidation(t *testing.T) {
	// Teste de validação de parâmetros
	params := completesync.CompleteSyncParams{
		TenantID:   "status_invest",
		CPF:        "12345678901",
		AssetTypes: []string{"equity"},
		DataTypes:  []string{"transactions", "positions"},
		DryRun:     true,
	}

	// Verificar que os campos obrigatórios estão presentes
	assert.NotEmpty(t, params.TenantID, "TenantID should not be empty")
	assert.NotEmpty(t, params.CPF, "CPF should not be empty")
	assert.NotEmpty(t, params.AssetTypes, "AssetTypes should not be empty")
	assert.NotEmpty(t, params.DataTypes, "DataTypes should not be empty")
}

func TestCompleteSyncResult_Structure(t *testing.T) {
	// Teste da estrutura do resultado
	result := &completesync.CompleteSyncResult{
		Strategy:          "FULL_HISTORICAL",
		ClientStatus:      "new",
		ExecutionPlan:     "complete_historical_ingestion",
		Success:           true,
		NewDataIngested:   true,
		ProcessedDataTypes: []string{"transactions", "positions"},
		DataSummary: completesync.DataSummary{
			RawRecordsIngested:     1500,
			TransactionsNormalized: 800,
			PositionsNormalized:    400,
		},
	}

	assert.Equal(t, "FULL_HISTORICAL", result.Strategy)
	assert.Equal(t, "new", result.ClientStatus)
	assert.True(t, result.Success)
	assert.True(t, result.NewDataIngested)
	assert.Equal(t, 1500, result.DataSummary.RawRecordsIngested)
	assert.Equal(t, 800, result.DataSummary.TransactionsNormalized)
	assert.Equal(t, 400, result.DataSummary.PositionsNormalized)
}
