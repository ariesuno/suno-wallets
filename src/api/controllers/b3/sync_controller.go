package b3

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "suno-wallets/src/api/middlewares"
    appsync "suno-wallets/src/application/b3/sync"
    "suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: controller para sync incremental diário

type SyncController struct{ svc *appsync.Service }

func NewSyncController(svc *appsync.Service) *SyncController { return &SyncController{svc: svc} }

type runRequest struct {
    Scope       string   `json:"scope" example:"single|tenant"`
    CPF         string   `json:"cpf" example:"12345678901"`
    DataTypes   []string `json:"dataTypes" example:"transactions,positions"`
    AssetTypes  []string `json:"assetTypes" example:"equity"`
    Force       bool     `json:"force"`
    DryRun      bool     `json:"dryRun"`
    Limit       int      `json:"limit" example:"100"`
}

// Run godoc
// @Summary Executa sync incremental (diário)
// @Tags B3 Sync
// @Accept json
// @Produce json
// @Param X-Tenant-Id header string true "ID do inquilino"
// @Param request body runRequest true "Parâmetros"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /b3/sync/run [post]
func (c *SyncController) Run(ctx *gin.Context) {
    tenantID, ok := middlewares.GetTenantID(ctx)
    if !ok { return }
    var req runRequest
    if err := ctx.ShouldBindJSON(&req); err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    if req.Scope == "single" {
        if err := validation.ValidateCPF(req.CPF); err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    }
    sum, err := c.svc.Run(ctx, appsync.RunParams{TenantID: tenantID, Scope: appsync.RunScope(req.Scope), CPF: req.CPF, DataTypes: req.DataTypes, AssetTypes: req.AssetTypes, Force: req.Force, DryRun: req.DryRun, Limit: req.Limit})
    if err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    ctx.JSON(http.StatusOK, gin.H{"clients": sum.Clients, "success": sum.Success, "failed": sum.Failed, "newRaw": sum.NewRAW})
}

// LastSync godoc
// @Summary Últimos marcos de sync por CPF
// @Tags B3 Sync
// @Produce json
// @Param X-Tenant-Id header string true "ID do inquilino"
// @Param cpf query string true "CPF"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /b3/client/last-sync [get]
func (c *SyncController) LastSync(ctx *gin.Context) {
    tenantID, ok := middlewares.GetTenantID(ctx)
    if !ok { return }
    cpf := ctx.Query("cpf")
    if err := validation.ValidateCPF(cpf); err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    st, err := c.svc.Repo().GetByTenantCPF(ctx, tenantID, cpf)
    if err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    if st == nil { ctx.JSON(http.StatusOK, gin.H{"status": "not_found"}); return }
    ctx.JSON(http.StatusOK, gin.H{"lastTxSyncAt": st.LastTxSyncAt, "lastPosSyncAt": st.LastPosSyncAt})
}

// Status godoc
// @Summary Consulta estado de sync por CPF
// @Tags B3 Sync
// @Produce json
// @Param X-Tenant-Id header string true "ID do inquilino"
// @Param cpf query string true "CPF"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /b3/client/status [get]
func (c *SyncController) Status(ctx *gin.Context) {
    tenantID, ok := middlewares.GetTenantID(ctx)
    if !ok { return }
    cpf := ctx.Query("cpf")
    if err := validation.ValidateCPF(cpf); err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    st, err := c.svc.Repo().GetByTenantCPF(ctx, tenantID, cpf)
    if err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    if st == nil { ctx.JSON(http.StatusOK, gin.H{"status": "not_found"}); return }
    ctx.JSON(http.StatusOK, gin.H{
        "isActive": st.IsActive,
        "needsReprocess": st.NeedsReprocess,
        "lastTxSyncAt": st.LastTxSyncAt,
        "lastPosSyncAt": st.LastPosSyncAt,
        "lastCheckedAt": st.LastCheckedAt,
        "lastResult": st.LastResult,
        "failureCount": st.FailureCount,
    })
}


