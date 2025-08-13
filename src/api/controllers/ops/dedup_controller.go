package ops

import (
	"net/http"
	"os"
	"suno-wallets/src/api/middlewares"
	appops "suno-wallets/src/application/ops"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: controller para scan/resolve/list de dedup

type DedupController struct{ svc *appops.DedupService }

func NewDedupController(svc *appops.DedupService) *DedupController { return &DedupController{svc: svc} }

// Scan godoc
// @Summary Executar scan de deduplicação (Admin)
// @Description Executa varredura para detectar possíveis transações duplicadas no sistema. Requer autenticação de admin.
// @Tags Operations Dedup
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Admin-Secret header string false "Chave secreta de admin (se configurada)"
// @Param request body object true "Parâmetros do scan de deduplicação" example({"cpf": "12345678901", "startDate": "2024-01-01", "endDate": "2024-12-31", "dryRun": false})
// @Success 200 {object} map[string]interface{} "Resultado do scan - candidatos criados"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 403 {object} map[string]string "Acesso negado - requer permissão de admin"
// @Failure 500 {object} map[string]string "Erro na execução do scan"
// @Router /ops/dedup/scan [post]
func (c2 *DedupController) Scan(ctx *gin.Context) {
	if os.Getenv("ADMIN_SECRET") != "" && ctx.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	var body appops.ScanRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	created, err := c2.svc.Scan(ctx, tenantID, body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"created": created})
}

// Resolve godoc
// @Summary Resolver candidatos de deduplicação (Admin)
// @Description Resolve candidatos de deduplicação aplicando ações específicas (manter, remover, mesclar). Requer autenticação de admin.
// @Tags Operations Dedup
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Admin-Secret header string false "Chave secreta de admin (se configurada)"
// @Param request body object true "Parâmetros de resolução" example({"candidateIds": ["123e4567-e89b-12d3-a456-426614174000"], "action": "REMOVE_DUPLICATE", "keepTransactionId": "456e7890-e89b-12d3-a456-426614174000"})
// @Success 200 {object} map[string]interface{} "Resolução aplicada com sucesso"
// @Failure 400 {object} map[string]string "Parâmetros inválidos ou erro na resolução"
// @Failure 403 {object} map[string]string "Acesso negado - requer permissão de admin"
// @Router /ops/dedup/resolve [post]
func (c2 *DedupController) Resolve(ctx *gin.Context) {
	if os.Getenv("ADMIN_SECRET") != "" && ctx.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	var body appops.ResolveRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := c2.svc.Resolve(ctx, tenantID, body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// List godoc
// @Summary Listar candidatos de deduplicação
// @Description Retorna lista paginada de candidatos de deduplicação detectados, com filtros opcionais por CPF e status.
// @Tags Operations Dedup
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string false "Filtrar por CPF específico" example(12345678901)
// @Param status query string false "Filtrar por status (PENDING, RESOLVED, IGNORED)" example(PENDING)
// @Success 200 {object} map[string]interface{} "Lista de candidatos de deduplicação"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /ops/dedup/candidates [get]
func (c2 *DedupController) List(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	cpf := ctx.Query("cpf")
	status := ctx.Query("status")
	page, pageSize := 1, 50
	items, err := c2.svc.ListCandidates(ctx, tenantID, cpf, status, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}
