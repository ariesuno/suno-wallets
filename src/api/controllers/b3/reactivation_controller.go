package b3

import (
	"net/http"
	"time"

	"suno-wallets/src/api/middlewares"
	"suno-wallets/src/application/b3/reactivation"
	"suno-wallets/src/shared/validation"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: controller para reativação inteligente de clientes

type ReactivationController struct {
	svc *reactivation.Service
}

func NewReactivationController(svc *reactivation.Service) *ReactivationController {
	return &ReactivationController{svc: svc}
}

// AnalyzeReactivation godoc
// @Summary Analisar estratégia de reativação (Admin)
// @Description Analisa a situação de sincronização de um cliente e sugere a melhor estratégia para reativação, considerando gaps de dados e última sincronização.
// @Tags B3 Reactivation
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param request body object true "Parâmetros de análise" example({"cpf": "12345678901", "currentDate": "2024-01-15"})
// @Success 200 {object} map[string]interface{} "Plano de reativação sugerido"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 500 {object} map[string]string "Erro na análise"
// @Router /b3/admin/reactivation/analyze [post]
func (c *ReactivationController) AnalyzeReactivation(ctx *gin.Context) {
	tenantID := middlewares.MustGetTenantName(ctx)

	var req struct {
		CPF         string `json:"cpf" binding:"required"`
		CurrentDate string `json:"currentDate,omitempty"` // YYYY-MM-DD, opcional (default: hoje)
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_REQUEST", "details": err.Error()})
		return
	}

	// Validação do CPF
	if err := validation.ValidateCPF(req.CPF); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse da data atual (default: hoje)
	var currentDate time.Time
	if req.CurrentDate == "" {
		currentDate = time.Now().UTC()
	} else {
		var err error
		currentDate, err = time.Parse("2006-01-02", req.CurrentDate)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_DATE_FORMAT", "message": "Use formato YYYY-MM-DD"})
			return
		}
	}

	// Normalizar para início do dia
	currentDate = time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)

	plan, err := c.svc.AnalyzeReactivation(ctx, reactivation.ReactivationParams{
		TenantID:    tenantID,
		CPF:         req.CPF,
		CurrentDate: currentDate,
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "ANALYSIS_FAILED",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"plan":    plan,
		"metadata": gin.H{
			"cpf":         req.CPF,
			"currentDate": currentDate.Format("2006-01-02"),
			"analyzedAt":  time.Now().UTC(),
			"tenantId":    tenantID,
		},
	})
}

// ExecuteReactivation godoc
// @Summary Executar plano de reativação (Admin)
// @Description Executa a estratégia de reativação previamente analisada, sincronizando dados faltantes do cliente com a B3. Suporta modo dry-run para simulação.
// @Tags B3 Reactivation
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param request body object true "Parâmetros de execução" example({"cpf": "12345678901", "currentDate": "2024-01-15", "dryRun": false})
// @Success 200 {object} map[string]interface{} "Resultado da execução"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 500 {object} map[string]string "Erro na execução"
// @Router /b3/admin/reactivation/execute [post]
func (c *ReactivationController) ExecuteReactivation(ctx *gin.Context) {
	tenantID := middlewares.MustGetTenantName(ctx)

	var req struct {
		CPF         string `json:"cpf" binding:"required"`
		CurrentDate string `json:"currentDate,omitempty"` // YYYY-MM-DD, opcional (default: hoje)
		DryRun      bool   `json:"dryRun,omitempty"`      // Default: false
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_REQUEST", "details": err.Error()})
		return
	}

	// Validação do CPF
	if err := validation.ValidateCPF(req.CPF); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse da data atual (default: hoje)
	var currentDate time.Time
	if req.CurrentDate == "" {
		currentDate = time.Now().UTC()
	} else {
		var err error
		currentDate, err = time.Parse("2006-01-02", req.CurrentDate)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_DATE_FORMAT", "message": "Use formato YYYY-MM-DD"})
			return
		}
	}

	// Normalizar para início do dia
	currentDate = time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)

	result, err := c.svc.ExecuteReactivation(ctx, reactivation.ReactivationParams{
		TenantID:    tenantID,
		CPF:         req.CPF,
		CurrentDate: currentDate,
	}, req.DryRun)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "EXECUTION_FAILED",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
		"metadata": gin.H{
			"cpf":         req.CPF,
			"currentDate": currentDate.Format("2006-01-02"),
			"executedAt":  time.Now().UTC(),
			"tenantId":    tenantID,
			"dryRun":      req.DryRun,
		},
	})
}

// GetReactivationStatus godoc
// @Summary Status de reativação do cliente
// @Description Verifica o status atual de sincronização do cliente e informa se há necessidade de reativação, sem executar operações.
// @Tags B3 Reactivation
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Success 200 {object} map[string]interface{} "Status de reativação"
// @Failure 400 {object} map[string]string "CPF inválido"
// @Failure 500 {object} map[string]string "Erro na verificação"
// @Router /b3/client/reactivation/status [get]
func (c *ReactivationController) GetReactivationStatus(ctx *gin.Context) {
	tenantID := middlewares.MustGetTenantName(ctx)

	cpf := ctx.Query("cpf")
	if cpf == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "CPF_REQUIRED"})
		return
	}

	// Validação do CPF
	if err := validation.ValidateCPF(cpf); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Análise apenas para entender o status (sem executar)
	plan, err := c.svc.AnalyzeReactivation(ctx, reactivation.ReactivationParams{
		TenantID:    tenantID,
		CPF:         cpf,
		CurrentDate: time.Now().UTC(),
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "STATUS_CHECK_FAILED",
			"message": err.Error(),
		})
		return
	}

	// Resposta simplificada para cliente (sem detalhes técnicos)
	status := "up_to_date"
	if plan.GapDays > 1 {
		status = "needs_sync"
	}
	if plan.Strategy == reactivation.StrategyFullHistorical {
		status = "new_client"
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  status,
		"summary": gin.H{
			"gapDays":           plan.GapDays,
			"lastSyncDate":      plan.LastSyncDate,
			"strategy":          plan.Strategy,
			"estimatedMinutes":  plan.EstimatedMinutes,
			"needsReactivation": plan.GapDays > 1,
		},
		"checkedAt": time.Now().UTC(),
	})
}
