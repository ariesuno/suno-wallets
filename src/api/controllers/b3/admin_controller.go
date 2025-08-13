package b3

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"suno-wallets/src/api/middlewares"
	appE2E "suno-wallets/src/application/b3/e2e"
	incr "suno-wallets/src/application/b3/incremental"
	"suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: controller admin-only para reset + full re-fetch + normalize (E2E)

type AdminController struct {
	orch *appE2E.Orchestrator
	incr *incr.Service
}

func NewAdminController(orch *appE2E.Orchestrator, inc *incr.Service) *AdminController {
	return &AdminController{orch: orch, incr: inc}
}

type resetAndRefetchRequest struct {
	CPF           string   `json:"cpf" binding:"required"`
	AssetTypes    []string `json:"assetTypes"`
	DataTypes     []string `json:"dataTypes"`
	StartOverride *string  `json:"startOverride"`
	EndOverride   *string  `json:"endOverride"`
	Force         bool     `json:"force"`
	DryRun        bool     `json:"dryRun"`
	Mode          string   `json:"mode"`        // archive|hard-delete
	Confirm       string   `json:"confirm"`     // must be RESET_AND_REFETCH
	ConfirmHard   *string  `json:"confirmHard"` // must be YES_DELETE when hard-delete
}

// ResetAndRefetch godoc
// @Summary Reset e reingestão histórica completa B3 (Admin)
// @Description Operação administrativa crítica que reset os dados de um cliente e refaz ingestão histórica completa da B3 com normalização. Requer confirmação explícita e autenticação de admin. Suporta modos archive/hard-delete e dry-run.
// @Tags B3 Admin
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Admin-Secret header string false "Chave secreta de admin (se configurada)"
// @Param request body resetAndRefetchRequest true "Parâmetros da operação reset" example({"cpf": "12345678901", "assetTypes": ["equity"], "dataTypes": ["transactions", "positions"], "mode": "archive", "confirm": "RESET_AND_REFETCH", "dryRun": false})
// @Success 200 {object} map[string]interface{} "Resultado da operação reset e reingestão"
// @Failure 400 {object} map[string]string "Parâmetros inválidos ou confirmação ausente"
// @Failure 403 {object} map[string]string "Acesso negado - requer permissão de admin"
// @Router /b3/admin/reset-and-refetch [post]
func (c *AdminController) ResetAndRefetch(ctx *gin.Context) {
	tenantID := middlewares.MustGetTenantName(ctx)

	// RBAC simples: header X-Admin-Secret deve bater com env ADMIN_SECRET (até termos RBAC real)
	if os.Getenv("ADMIN_SECRET") != "" {
		if ctx.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN", "message": "admin access required"})
			return
		}
	}

	var req resetAndRefetchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateCPF(req.CPF); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Confirm != "RESET_AND_REFETCH" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_CONFIRM", "message": "confirm must be RESET_AND_REFETCH"})
		return
	}
	if req.Mode == "" {
		req.Mode = "archive"
	}
	if req.Mode != "archive" && req.Mode != "hard-delete" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_MODE", "message": "mode must be archive or hard-delete"})
		return
	}
	if req.Mode == "hard-delete" {
		if req.ConfirmHard == nil || *req.ConfirmHard != "YES_DELETE" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "MISSING_CONFIRM_HARD", "message": "confirmHard must be YES_DELETE for hard-delete"})
			return
		}
	}

	// defaults
	if len(req.AssetTypes) == 0 {
		req.AssetTypes = []string{"equity"}
	}
	if len(req.DataTypes) == 0 {
		req.DataTypes = []string{"transactions", "positions"}
	}

	// janela
	earliest := os.Getenv("B3_API_EARLIEST_DATE")
	if earliest == "" {
		earliest = "2019-10-01"
	}
	startStr := earliest
	if req.StartOverride != nil {
		startStr = *req.StartOverride
	}
	endStr := time.Now().UTC().Format("2006-01-02")
	if req.EndOverride != nil {
		endStr = *req.EndOverride
	}
	if err := validation.ValidateDateYMD(startStr); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateDateYMD(endStr); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	startT, _ := time.Parse("2006-01-02", startStr)
	endT, _ := time.Parse("2006-01-02", endStr)

	// Executar orquestração
	out, err := c.orch.Run(ctx, appE2E.Params{
		TenantID:    tenantID,
		CPF:         req.CPF,
		AssetTypes:  req.AssetTypes,
		DataTypes:   req.DataTypes,
		Start:       startT,
		End:         endT,
		Force:       req.Force,
		DryRun:      req.DryRun,
		Mode:        req.Mode,
		RequestedBy: ctx.GetHeader("X-Admin-Actor"),
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"raw": gin.H{
			"saved": out.Raw.Saved, "skipped": out.Raw.Skipped, "errors": out.Raw.Errors,
			"monthsProcessed": out.Raw.MonthsProcessed, "pagesProcessed": out.Raw.PagesProcessed,
		},
		"normalized": gin.H{
			"inserted": out.Normalized.Inserted, "updated": out.Normalized.Updated,
			"skipped": out.Normalized.Skipped, "errors": out.Normalized.Errors,
		},
		"startedAt":  out.StartedAt.Format(time.RFC3339),
		"finishedAt": out.FinishedAt.Format(time.RFC3339),
		"durationMs": out.DurationMs,
		"mode":       out.Mode,
		"force":      out.Force,
		"dryRun":     out.DryRun,
		"tenantId":   tenantID,
	})
}

// IncrementalFromLast godoc
// @Summary Sincronização incremental inteligente B3 (Admin)
// @Description Executa sincronização incremental inteligente a partir do último marco de sincronização ou data específica. Detecta gaps e sincroniza apenas os períodos necessários. Requer autenticação de admin.
// @Tags B3 Admin
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Admin-Secret header string false "Chave secreta de admin (se configurada)"
// @Param request body object true "Parâmetros da sincronização incremental" example({"cpf": "12345678901", "dataTypes": ["transactions", "positions"], "assetTypes": ["equity"], "since": "2024-01-01", "dryRun": false, "concurrency": 2})
// @Success 200 {object} map[string]interface{} "Resultado da sincronização incremental"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 403 {object} map[string]string "Acesso negado - requer permissão de admin"
// @Failure 503 {object} map[string]string "Serviço incremental não configurado"
// @Router /b3/admin/incremental-from-last [post]
func (c *AdminController) IncrementalFromLast(ctx *gin.Context) {
	tenantID := middlewares.MustGetTenantName(ctx)
	if os.Getenv("ADMIN_SECRET") != "" && ctx.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	var req struct {
		CPF         string   `json:"cpf"`
		DataTypes   []string `json:"dataTypes"`
		AssetTypes  []string `json:"assetTypes"`
		Since       *string  `json:"since"`
		EndOverride *string  `json:"endOverride"`
		DryRun      bool     `json:"dryRun"`
		Force       bool     `json:"force"`
		Concurrency int      `json:"concurrency"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateCPF(req.CPF); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var sincePtr, endPtr *time.Time
	if req.Since != nil {
		if err := validation.ValidateDateYMD(*req.Since); err == nil {
			t, _ := time.Parse("2006-01-02", *req.Since)
			sincePtr = &t
		}
	}
	if req.EndOverride != nil {
		if err := validation.ValidateDateYMD(*req.EndOverride); err == nil {
			t, _ := time.Parse("2006-01-02", *req.EndOverride)
			endPtr = &t
		}
	}
	if c.incr == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "incremental service not configured"})
		return
	}
	out, err := c.incr.Run(ctx, incr.Params{TenantID: tenantID, CPF: req.CPF, DataTypes: req.DataTypes, AssetTypes: req.AssetTypes, Since: sincePtr, End: endPtr, Force: req.Force, DryRun: req.DryRun, Concurrency: req.Concurrency})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, out)
}
