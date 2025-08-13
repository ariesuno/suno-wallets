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

// Get godoc
// @Summary Obter política do cliente (Admin)
// @Description Retorna a política de fonte de dados ativa para um cliente específico (B3_ONLY, MANUAL_ONLY ou HYBRID).
// @Tags Client Policy
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Success 200 {object} map[string]interface{} "Política ativa do cliente"
// @Failure 400 {object} map[string]string "CPF obrigatório"
// @Router /admin/client/policy [get]
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
	mode := c.svc.GetMode(ctx, tenantID.String(), cpf)
	ctx.JSON(http.StatusOK, gin.H{"mode": mode})
}

// Upsert godoc
// @Summary Definir política do cliente (Admin)
// @Description Cria ou atualiza a política de fonte de dados para um cliente. Suporta B3_ONLY, MANUAL_ONLY e HYBRID.
// @Tags Client Policy
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param request body object true "Dados da política" example({"cpf": "12345678901", "mode": "B3_ONLY", "reason": "Integração automática"})
// @Success 201 {object} map[string]interface{} "Política configurada com sucesso"
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /admin/client/policy [post]
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
	if err := c.svc.Upsert(ctx, tenantID.String(), body.CPF, m, body.Reason, "admin:api"); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"status": "ok"})
}

// Audit godoc
// @Summary Histórico de mudanças de política (Admin)
// @Description Retorna o histórico de mudanças na política de fonte de dados de um cliente específico.
// @Tags Client Policy
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Success 200 {object} map[string]interface{} "Histórico de mudanças da política"
// @Failure 400 {object} map[string]string "CPF obrigatório"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /admin/client/policy/audit [get]
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
	items, err := c.svc.ListAudit(ctx, tenantID.String(), cpf, 50)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

// DryRun godoc
// @Summary Simular impacto de mudança de política (Admin)
// @Description Simula o impacto de alterar a política de um cliente, mostrando jobs afetados, endpoints bloqueados e mudanças no sistema.
// @Tags Client Policy
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param request body object true "Simulação de mudança" example({"cpf": "12345678901", "newMode": "MANUAL_ONLY"})
// @Success 200 {object} map[string]interface{} "Análise de impacto da mudança"
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Router /admin/client/policy/dry-run [post]
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
