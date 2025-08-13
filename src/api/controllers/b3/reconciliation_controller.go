package b3

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"suno-wallets/src/api/middlewares"
	apprecon "suno-wallets/src/application/b3/reconciliation"
	reconrepo "suno-wallets/src/infrastructure/b3/reconciliation"
	"suno-wallets/src/shared/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Comentários em pt-BR: Controller admin/read para inconsistency detector (1.17)

type ReconciliationController struct{ svc *apprecon.Service }

func NewReconciliationController(dbRepo *reconrepo.Repository) *ReconciliationController {
	return &ReconciliationController{svc: apprecon.NewService(dbRepo)}
}

// Auto-fix wiring simplificado (1.18)
type AutoFixController struct {
	svc *apprecon.SystemOperationsService
}

func NewAutoFixController(sysRepo *reconrepo.SysOpsRepository, price apprecon.PriceLookupPort) *AutoFixController {
	return &AutoFixController{svc: apprecon.NewSystemOperationsService(sysRepo, price)}
}

// Scan godoc
// @Summary Executar scan de reconciliação (Admin)
// @Description Executa um scan completo de reconciliação para detectar inconsistências entre dados normalizados e posições calculadas. Requer autenticação de admin.
// @Tags B3 Reconciliation
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Admin-Secret header string false "Chave secreta de admin (se configurada)"
// @Param request body object true "Parâmetros do scan" example({"cpf": "12345678901", "concurrency": 4, "dryRun": false})
// @Success 200 {object} map[string]interface{} "Resultado do scan de reconciliação"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 403 {object} map[string]string "Acesso negado - requer permissão de admin"
// @Failure 500 {object} map[string]string "Erro na execução do scan"
// @Router /b3/reconciliation/scan [post]
func (rc *ReconciliationController) Scan(c *gin.Context) {
	if os.Getenv("ADMIN_SECRET") != "" && c.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
		c.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	tenantId := middlewares.MustGetTenantName(c)
	var req apprecon.ScanRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Concurrency < 0 {
		req.Concurrency = 0
	}
	start := time.Now()
	result, err := rc.svc.Scan(c, tenantId, req)
	if err != nil {
		helpers.LogError("recon_scan_error", err, map[string]interface{}{"tenantId": tenantId})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
		return
	}
	// anexa duration também no controller
	result["durationMsController"] = time.Since(start).Milliseconds()
	c.JSON(http.StatusOK, result)
}

// List godoc
// @Summary Listar inconsistências detectadas
// @Description Retorna lista paginada de inconsistências detectadas no sistema de reconciliação, com filtros opcionais por CPF, status, tipo, ticker e período.
// @Tags B3 Reconciliation
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string false "Filtrar por CPF específico" example(12345678901)
// @Param status query string false "Filtrar por status (open, resolved, ignored)" example(open)
// @Param type query string false "Filtrar por tipo de inconsistência" example(position_mismatch)
// @Param ticker query string false "Filtrar por ticker/código do ativo" example(PETR4)
// @Param from query string false "Data inicial no formato YYYY-MM-DD" example(2024-01-01)
// @Param to query string false "Data final no formato YYYY-MM-DD" example(2024-12-31)
// @Param page query int false "Número da página (padrão: 1)" example(1)
// @Param pageSize query int false "Tamanho da página (1-200, padrão: 50)" example(50)
// @Success 200 {object} map[string]interface{} "Lista paginada de inconsistências"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /b3/reconciliation/inconsistencies [get]
func (rc *ReconciliationController) List(c *gin.Context) {
	// Placeholder: leitura simples inicial por tenant (poderá evoluir com filtros/paginação)
	tenantId := middlewares.MustGetTenantName(c)
	cpf := c.Query("cpf")
	status := c.Query("status")
	typ := c.Query("type")
	ticker := c.Query("ticker")
	// filtros adicionais e paginação
	var fromPtr, toPtr *time.Time
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			fromPtr = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			toPtr = &t
		}
	}
	page, pageSize := 1, 50
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			pageSize = n
		}
	}
	items, err := rc.svc.List(c, tenantId, cpf, status, typ, ticker, fromPtr, toPtr, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "pageSize": pageSize})
}

// Get godoc
// @Summary Obter inconsistência específica
// @Description Retorna detalhes completos de uma inconsistência específica identificada pelo ID.
// @Tags B3 Reconciliation
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param id path string true "ID da inconsistência (UUID)" example(123e4567-e89b-12d3-a456-426614174000)
// @Success 200 {object} map[string]interface{} "Detalhes da inconsistência"
// @Failure 400 {object} map[string]string "ID inválido"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /b3/reconciliation/inconsistencies/{id} [get]
func (rc *ReconciliationController) Get(c *gin.Context) {
	tenantId := middlewares.MustGetTenantName(c)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	item, err := rc.svc.Get(c, tenantId, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get failed"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// AutoFix godoc
// @Summary Executar correção automática (Admin)
// @Description Executa correção automática de inconsistências detectadas no sistema de reconciliação. Suporta filtros por CPF, tipos e tickers específicos. Requer autenticação de admin.
// @Tags B3 Reconciliation
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Admin-Secret header string false "Chave secreta de admin (se configurada)"
// @Param request body object true "Parâmetros da correção automática" example({"cpf": "12345678901", "types": ["position_mismatch"], "tickers": ["PETR4"], "dryRun": false})
// @Success 200 {object} map[string]interface{} "Resultado da correção automática"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 403 {object} map[string]string "Acesso negado - requer permissão de admin"
// @Failure 500 {object} map[string]string "Erro na execução da correção"
// @Router /b3/reconciliation/auto-fix [post]
func (ac *AutoFixController) AutoFix(c *gin.Context) {
	if os.Getenv("ADMIN_SECRET") != "" && c.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
		c.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	tenantId := middlewares.MustGetTenantName(c)
	var req apprecon.AutoFixRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	out, err := ac.svc.AutoFix(c, tenantId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// AutoFixByID godoc
// @Summary Corrigir inconsistência específica (Admin)
// @Description Executa correção automática de uma inconsistência específica identificada pelo ID. Requer autenticação de admin.
// @Tags B3 Reconciliation
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Admin-Secret header string false "Chave secreta de admin (se configurada)"
// @Param id path string true "ID da inconsistência (UUID)" example(123e4567-e89b-12d3-a456-426614174000)
// @Param dryRun query bool false "Modo simulação (não aplica correções)" example(false)
// @Success 200 {object} map[string]interface{} "Resultado da correção específica"
// @Failure 400 {object} map[string]string "ID inválido"
// @Failure 403 {object} map[string]string "Acesso negado - requer permissão de admin"
// @Failure 404 {object} map[string]string "Inconsistência não encontrada"
// @Failure 500 {object} map[string]string "Erro na execução da correção"
// @Router /b3/reconciliation/auto-fix/{id} [post]
func (ac *AutoFixController) AutoFixByID(c *gin.Context) {
	if os.Getenv("ADMIN_SECRET") != "" && c.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
		c.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	tenantId := middlewares.MustGetTenantName(c)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	// Buscar inconsistency e processar isoladamente
	inc, err := ac.svc.Repo.GetInconsistencyByID(c, tenantId, id)
	if err != nil || inc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	req := apprecon.AutoFixRequest{CPF: inc.CPF, Types: []string{inc.Type}, Tickers: []string{inc.Ticker}, DryRun: c.Query("dryRun") == "true"}
	out, err := ac.svc.AutoFix(c, tenantId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "result": out})
}
