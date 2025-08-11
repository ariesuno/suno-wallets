package clientpolicy

import (
	"net/http"
	"suno-wallets/src/api/middlewares"
	appsvc "suno-wallets/src/application/clientpolicy"
	dom "suno-wallets/src/domain/clientpolicy"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: endpoints admin para leitura/upsert/audit da client policy

type Controller struct{ svc *appsvc.Service }

func NewController(s *appsvc.Service) *Controller { return &Controller{svc: s} }

func (c *Controller) Get(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	cpf := ctx.Query("cpf")
	if cpf == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cpf required"})
		return
	}
	mode := c.svc.GetMode(ctx, tenantID, cpf)
	ctx.JSON(http.StatusOK, gin.H{"mode": mode})
}

func (c *Controller) Upsert(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	var body struct {
		CPF    string `json:"cpf"`
		Mode   string `json:"mode"`
		Reason string `json:"reason"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.CPF == "" || body.Mode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cpf/mode required"})
		return
	}
	m := parseMode(body.Mode)
	if err := c.svc.Upsert(ctx, tenantID, body.CPF, m, body.Reason, "admin:api"); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"status": "ok"})
}

func (c *Controller) Audit(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	cpf := ctx.Query("cpf")
	if cpf == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cpf required"})
		return
	}
	items, err := c.svc.ListAudit(ctx, tenantID, cpf, 50)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

// Dry-run de impacto (admin)
func (c *Controller) DryRun(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	var body struct {
		CPF     string `json:"cpf"`
		NewMode string `json:"newMode"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.CPF == "" || body.NewMode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cpf/newMode required"})
		return
	}
	m := parseMode(body.NewMode)
	resp := map[string]any{
		"affectedJobs":     []string{"B3_FullFetch", "B3_Incremental", "DedupScan", "ManualOpsCRUD"},
		"blockedEndpoints": []string{},
		"readSideChanges":  []string{},
	}
	switch m {
	case dom.ModeB3Only:
		resp["blockedEndpoints"] = []string{"/ops/manual (POST, PUT, DELETE)"}
	case dom.ModeManualOnly:
		resp["blockedEndpoints"] = []string{"/b3/fetch/*"}
		resp["readSideChanges"] = []string{"ignore B3_RAW in queries"}
	case dom.ModeHybrid:
		// nenhum bloqueio
	}
	_ = tenantID
	ctx.JSON(http.StatusOK, resp)
}

func parseMode(s string) dom.Mode {
	switch s {
	case string(dom.ModeB3Only):
		return dom.ModeB3Only
	case string(dom.ModeManualOnly):
		return dom.ModeManualOnly
	default:
		return dom.ModeHybrid
	}
}
