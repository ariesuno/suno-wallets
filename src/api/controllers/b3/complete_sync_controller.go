package b3

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"suno-wallets/src/api/middlewares"
	completesync "suno-wallets/src/application/b3/sync"
	"suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: controller para sincronização completa e ingestão B3

type CompleteSyncController struct {
	orchestrator *completesync.CompleteSyncOrchestrator
}

func NewCompleteSyncController(orchestrator *completesync.CompleteSyncOrchestrator) *CompleteSyncController {
	return &CompleteSyncController{orchestrator: orchestrator}
}

type completeSyncRequest struct {
	CPF                   string   `json:"cpf" binding:"required"`
	AssetTypes            []string `json:"assetTypes,omitempty"`
	DataTypes             []string `json:"dataTypes,omitempty"`
	IncludeReconciliation bool     `json:"includeReconciliation,omitempty"`
	DryRun                bool     `json:"dryRun,omitempty"`
	Force                 bool     `json:"force,omitempty"`
}

// ExecuteCompleteSync godoc
// @Summary Sincronização completa e ingestão B3
// @Description Executa fluxo completo e inteligente de sincronização B3: analisa automaticamente o status do cliente (novo vs existente), escolhe a estratégia ideal (ingestão histórica completa vs incremental vs reativação híbrida), executa todo o pipeline de ingestão e normalização, e opcionalmente realiza reconciliação. Este é o endpoint unificado para processamento completo de dados B3.
// @Tags B3 Complete Sync
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param request body completeSyncRequest true "Parâmetros da sincronização completa" example({"cpf": "12345678901", "assetTypes": ["equity"], "dataTypes": ["transactions", "positions"], "includeReconciliation": true, "dryRun": false, "force": false})
// @Success 200 {object} completesync.CompleteSyncResult "Resultado da sincronização completa"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 500 {object} map[string]string "Erro na sincronização"
// @Router /b3/sync/complete-ingestion [post]
func (c *CompleteSyncController) ExecuteCompleteSync(ctx *gin.Context) {
	started := time.Now()
	tenantID := middlewares.MustGetTenantName(ctx)

	var req completeSyncRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validação obrigatória de CPF
	if err := validation.ValidateCPF(req.CPF); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Aplicar defaults seguindo padrões do projeto
	if len(req.AssetTypes) == 0 {
		req.AssetTypes = []string{"equities", "fixed-income", "treasury-bonds", "derivatives"}
	}
	if len(req.DataTypes) == 0 {
		req.DataTypes = []string{"transactions", "positions"}
	}

	// Preparar parâmetros para o orchestrator
	params := completesync.CompleteSyncParams{
		TenantID:              tenantID,
		CPF:                   req.CPF,
		AssetTypes:            req.AssetTypes,
		DataTypes:             req.DataTypes,
		IncludeReconciliation: req.IncludeReconciliation,
		DryRun:                req.DryRun,
		Force:                 req.Force,
	}

	// Executar sincronização completa
	result, err := c.orchestrator.ExecuteCompleteSync(ctx, params)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":      err.Error(),
			"durationMs": time.Since(started).Milliseconds(),
		})
		return
	}

	// Retornar resultado completo
	ctx.JSON(http.StatusOK, result)
}

// GetSyncStatus godoc
// @Summary Status de sincronização do cliente
// @Description Analisa o status atual de sincronização de um cliente sem executar nenhuma ação, retornando a estratégia recomendada e informações de última sincronização.
// @Tags B3 Complete Sync
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Success 200 {object} map[string]interface{} "Status e estratégia recomendada"
// @Failure 400 {object} map[string]string "CPF inválido"
// @Failure 500 {object} map[string]string "Erro na análise"
// @Router /b3/sync/status-analysis [get]
func (c *CompleteSyncController) GetSyncStatus(ctx *gin.Context) {
	tenantID := middlewares.MustGetTenantName(ctx)
	cpf := ctx.Query("cpf")

	if err := validation.ValidateCPF(cpf); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Executar apenas análise (sem processamento real)
	params := completesync.CompleteSyncParams{
		TenantID:              tenantID,
		CPF:                   cpf,
		AssetTypes:            []string{"equities", "fixed-income", "treasury-bonds", "derivatives"},
		DataTypes:             []string{"transactions", "positions"},
		IncludeReconciliation: false,
		DryRun:                true, // Sempre dry run para análise
		Force:                 false,
	}

	result, err := c.orchestrator.ExecuteCompleteSync(ctx, params)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Retornar apenas informações de análise
	ctx.JSON(http.StatusOK, gin.H{
		"strategy":          result.Strategy,
		"clientStatus":      result.ClientStatus,
		"recommendedPlan":   result.ExecutionPlan,
		"reactivationPlan":  result.ReactivationPlan,
		"estimatedDuration": result.ReactivationPlan.EstimatedMinutes,
		"lastSyncDate":      result.ReactivationPlan.LastSyncDate,
		"gapDays":           result.ReactivationPlan.GapDays,
		"analysisTime":      result.DurationMs,
	})
}
