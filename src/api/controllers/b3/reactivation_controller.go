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

// POST /b3/admin/reactivation/analyze - Analisa estratégia de reativação (admin-only)
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

// POST /b3/admin/reactivation/execute - Executa plano de reativação (admin-only)
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

// GET /b3/client/reactivation/status - Status de reativação para o cliente (público)
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
