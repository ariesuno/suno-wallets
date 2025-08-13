package b3

import (
	"net/http"

	"suno-wallets/src/api/middlewares"
	appsync "suno-wallets/src/application/b3/sync"
	"suno-wallets/src/shared/validation"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: controller para sync incremental diário

type SyncController struct{ svc *appsync.Service }

func NewSyncController(svc *appsync.Service) *SyncController { return &SyncController{svc: svc} }

type runRequest struct {
	Scope      string   `json:"scope" example:"single|tenant"`
	CPF        string   `json:"cpf" example:"12345678901"`
	DataTypes  []string `json:"dataTypes" example:"transactions,positions"`
	AssetTypes []string `json:"assetTypes" example:"equity"`
	Force      bool     `json:"force"`
	DryRun     bool     `json:"dryRun"`
	Limit      int      `json:"limit" example:"100"`
}

// Run godoc
// @Summary Executar sincronização incremental B3
// @Description Executa sincronização incremental com a B3 para atualizar dados de transações e posições. Suporta escopo single (CPF específico) ou tenant (todos os CPFs). Inclui controles de força, dry-run e limite.
// @Tags B3 Sync
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param request body runRequest true "Parâmetros da sincronização" example({"scope": "single", "cpf": "12345678901", "dataTypes": ["transactions", "positions"], "assetTypes": ["equity"], "force": false, "dryRun": false, "limit": 100})
// @Success 200 {object} map[string]interface{} "Resultado da sincronização com contadores"
// @Failure 400 {object} map[string]string "Parâmetros inválidos ou erro na sincronização"
// @Router /b3/sync/run [post]
func (c *SyncController) Run(ctx *gin.Context) {
	tenantID := middlewares.MustGetTenantName(ctx)
	var req runRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Scope == "single" {
		if err := validation.ValidateCPF(req.CPF); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	sum, err := c.svc.Run(ctx, appsync.RunParams{TenantID: tenantID, Scope: appsync.RunScope(req.Scope), CPF: req.CPF, DataTypes: req.DataTypes, AssetTypes: req.AssetTypes, Force: req.Force, DryRun: req.DryRun, Limit: req.Limit})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"clients": sum.Clients, "success": sum.Success, "failed": sum.Failed, "newRaw": sum.NewRAW})
}

// LastSync godoc
// @Summary Obter últimas sincronizações do cliente
// @Description Retorna os timestamps das últimas sincronizações de transações e posições para um CPF específico, útil para controle de incrementais.
// @Tags B3 Sync
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Success 200 {object} map[string]interface{} "Timestamps das últimas sincronizações"
// @Failure 400 {object} map[string]string "CPF inválido"
// @Router /b3/client/last-sync [get]
func (c *SyncController) LastSync(ctx *gin.Context) {
	tenantID := middlewares.MustGetTenantName(ctx)
	cpf := ctx.Query("cpf")
	if err := validation.ValidateCPF(cpf); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	st, err := c.svc.Repo().GetByTenantCPF(ctx, tenantID, cpf)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if st == nil {
		ctx.JSON(http.StatusOK, gin.H{"status": "not_found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"lastTxSyncAt": st.LastTxSyncAt, "lastPosSyncAt": st.LastPosSyncAt})
}

// Status godoc
// @Summary Obter status de sincronização do cliente
// @Description Retorna status detalhado da sincronização de um cliente, incluindo flags de ativação, necessidade de reprocessamento, últimas sincronizações, resultados e contadores de falha.
// @Tags B3 Sync
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Success 200 {object} map[string]interface{} "Status completo de sincronização"
// @Failure 400 {object} map[string]string "CPF inválido"
// @Router /b3/client/status [get]
func (c *SyncController) Status(ctx *gin.Context) {
	tenantID := middlewares.MustGetTenantName(ctx)
	cpf := ctx.Query("cpf")
	if err := validation.ValidateCPF(cpf); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	st, err := c.svc.Repo().GetByTenantCPF(ctx, tenantID, cpf)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if st == nil {
		ctx.JSON(http.StatusOK, gin.H{"status": "not_found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"isActive":       st.IsActive,
		"needsReprocess": st.NeedsReprocess,
		"lastTxSyncAt":   st.LastTxSyncAt,
		"lastPosSyncAt":  st.LastPosSyncAt,
		"lastCheckedAt":  st.LastCheckedAt,
		"lastResult":     st.LastResult,
		"failureCount":   st.FailureCount,
	})
}
