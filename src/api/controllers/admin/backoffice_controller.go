package admin

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"suno-wallets/src/api/middlewares"
	appadm "suno-wallets/src/application/admin"
	"suno-wallets/src/infrastructure/observability"
	"time"

	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Comentários em pt-BR: controller do Backoffice Admin (profile, actions básicas)

type Controller struct {
	q *appadm.QueryService
	a *appadm.ActionsService
}

func NewController(q *appadm.QueryService, a *appadm.ActionsService) *Controller {
	return &Controller{q: q, a: a}
}

// Profile godoc
// @Summary Perfil completo do cliente (Admin)
// @Description Retorna informações detalhadas do perfil de um cliente, incluindo dados de sincronização, políticas ativas e estatísticas de operações.
// @Tags Admin Backoffice
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Success 200 {object} map[string]interface{} "Perfil completo do cliente"
// @Failure 400 {object} map[string]string "CPF obrigatório ou tenant ausente"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /admin/backoffice/profile [get]
func (c *Controller) Profile(ctx *gin.Context) {
	defer func() {
		result := "SUCCESS"
		if ctx.Writer.Status() >= 400 {
			result = "ERROR"
		}
		observability.IncAdminProfileRequest(result)
	}()

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
	prof, err := c.q.Profile(ctx, tenantID.String(), cpf)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, prof)
}

// RequestAction godoc
// @Summary Solicitar ação administrativa
// @Description Cria uma solicitação de ação administrativa que requer confirmação posterior. Suporta TTL configurável e payload personalizado. Requer role ADMIN ou SUPPORT.
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Role header string true "Role do usuário (ADMIN ou SUPPORT)" example(ADMIN)
// @Param request body object true "Dados da solicitação de ação" example({"cpf": "12345678901", "action": "DELETE_CLIENT_DATA", "requestedBy": "admin@company.com", "ttl": 600, "payload": {"reason": "LGPD request"}})
// @Success 201 {object} map[string]interface{} "Ação solicitada com token de confirmação"
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 403 {object} map[string]string "Role insuficiente - requer ADMIN ou SUPPORT"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /admin/backoffice/actions/request [post]
func (c *Controller) RequestAction(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	role := ctx.GetHeader("X-Role")
	if role != "ADMIN" && role != "SUPPORT" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	var body struct {
		CPF         string                 `json:"cpf"`
		Action      string                 `json:"action"`
		RequestedBy string                 `json:"requestedBy"`
		TTL         int                    `json:"ttl"`
		Payload     map[string]interface{} `json:"payload"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.TTL <= 0 {
		body.TTL = 600
	}
	id, token, err := c.a.Request(ctx, tenantID.String(), body.CPF, body.Action, body.RequestedBy, body.TTL, body.Payload)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"id": id, "confirmToken": token})
}

// ConfirmAction godoc
// @Summary Confirmar ação administrativa
// @Description Confirma e executa uma ação administrativa previamente solicitada usando ID e token de confirmação. Requer role ADMIN.
// @Tags Admin Backoffice
// @Produce json
// @Param X-Role header string true "Role do usuário (deve ser ADMIN)" example(ADMIN)
// @Param id query string true "ID da ação solicitada" example(123e4567-e89b-12d3-a456-426614174000)
// @Param token query string true "Token de confirmação gerado na solicitação" example(abc123def456)
// @Success 200 {object} map[string]interface{} "Ação confirmada e executada"
// @Failure 400 {object} map[string]string "ID/token inválido ou ação expirada"
// @Failure 403 {object} map[string]string "Acesso negado - requer role ADMIN"
// @Router /admin/backoffice/actions/confirm [post]
func (c *Controller) ConfirmAction(ctx *gin.Context) {
	id := ctx.Query("id")
	token := ctx.Query("token")
	if id == "" || token == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id/token required"})
		return
	}
	role := ctx.GetHeader("X-Role")
	if role != "ADMIN" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	if err := c.a.Confirm(ctx, uuidFrom(id), token); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "confirmed"})
}

func uuidFrom(s string) uuid.UUID { u, _ := uuid.Parse(s); return u }

// Search godoc
// @Summary Buscar clientes e dados (Admin)
// @Description Busca clientes e informações relacionadas usando termo de pesquisa. Suporta busca por CPF, nome, email ou outras informações do cliente.
// @Tags Admin Backoffice
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param query query string false "Termo de busca (CPF, nome, email, etc.)" example(12345678901)
// @Param limit query int false "Limite de resultados (padrão: 20)" example(20)
// @Success 200 {object} map[string]interface{} "Resultados da busca"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /admin/backoffice/search [get]
func (c *Controller) Search(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	q := ctx.Query("query")
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	items, err := c.q.Search(ctx, tenantID.String(), q, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

// ListActions godoc
// @Summary Listar ações administrativas (Admin)
// @Description Retorna lista paginada de ações administrativas com filtros opcionais por CPF, tipo de ação e status.
// @Tags Admin Backoffice
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string false "Filtrar por CPF específico" example(12345678901)
// @Param action query string false "Filtrar por tipo de ação" example(DELETE_CLIENT_DATA)
// @Param status query string false "Filtrar por status (PENDING, CONFIRMED, EXPIRED)" example(PENDING)
// @Param page query int false "Número da página (padrão: 1)" example(1)
// @Param pageSize query int false "Tamanho da página (padrão: 50)" example(50)
// @Success 200 {object} map[string]interface{} "Lista paginada de ações administrativas"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /admin/backoffice/actions [get]
func (c *Controller) ListActions(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	cpf := ctx.Query("cpf")
	action := ctx.Query("action")
	status := ctx.Query("status")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "50"))
	items, err := c.q.ListActions(ctx, tenantID.String(), cpf, action, status, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items, "page": page, "pageSize": pageSize})
}

// GetAction godoc
// @Summary Obter detalhes de ação administrativa (Admin)
// @Description Retorna detalhes completos de uma ação administrativa específica identificada pelo ID.
// @Tags Admin Backoffice
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param id path string true "ID da ação administrativa (UUID)" example(123e4567-e89b-12d3-a456-426614174000)
// @Success 200 {object} map[string]interface{} "Detalhes da ação administrativa"
// @Failure 400 {object} map[string]string "ID inválido"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /admin/backoffice/actions/{id} [get]
func (c *Controller) GetAction(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	item, err := c.q.GetAction(ctx, tenantID.String(), id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

// ExportLedger godoc
// @Summary Exportar ledger do cliente (Admin)
// @Description Exporta o ledger completo de operações de um cliente específico. Permite filtrar dados da B3 e configurar limite de registros.
// @Tags Admin Backoffice
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Param limit query int false "Limite de registros (padrão: 50000)" example(50000)
// @Param excludeB3 query bool false "Excluir dados da B3 do export (padrão: false)" example(false)
// @Success 200 {object} map[string]interface{} "Ledger exportado do cliente"
// @Failure 400 {object} map[string]string "CPF obrigatório"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /admin/backoffice/export/ledger [get]
func (c *Controller) ExportLedger(ctx *gin.Context) {
	defer observability.IncAdminExport("json") // formato sempre JSON neste endpoint

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
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "50000"))
	excludeB3 := ctx.DefaultQuery("excludeB3", "false") == "true"
	items, err := c.q.ExportLedger(ctx, tenantID.String(), cpf, excludeB3, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

// ExportLedgerCSV godoc
// @Summary Exportar ledger em CSV com streaming (Admin)
// @Description Exporta o ledger completo de operações de um cliente em formato CSV com streaming para grandes volumes
// @Tags Admin Backoffice
// @Produce text/csv
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Param limit query int false "Limite de registros (padrão: 100000)" example(100000)
// @Param excludeB3 query bool false "Excluir dados da B3 do export (padrão: false)" example(false)
// @Success 200 {file} file "Arquivo CSV do ledger"
// @Failure 400 {object} map[string]string "CPF obrigatório"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /admin/backoffice/export/ledger/csv [get]
func (c *Controller) ExportLedgerCSV(ctx *gin.Context) {
	defer observability.IncAdminExport("csv")

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
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "100000"))
	excludeB3 := ctx.DefaultQuery("excludeB3", "false") == "true"

	// Configurar headers para download de arquivo CSV
	filename := fmt.Sprintf("ledger_%s_%s.csv", cpf[len(cpf)-4:], time.Now().Format("20060102_150405"))
	ctx.Header("Content-Type", "text/csv")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	ctx.Header("Cache-Control", "no-cache")

	// Criar writer CSV que escreve diretamente na response
	writer := csv.NewWriter(ctx.Writer)
	defer writer.Flush()

	// Escrever cabeçalho CSV
	header := []string{"ID", "Ticker", "Asset Type", "Operation Date", "Operation Type", "Source", "Quantity", "Unit Price", "Currency", "Reason Code", "Price Confidence"}
	if err := writer.Write(header); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write CSV header"})
		return
	}

	// Callback para processar cada lote de dados
	callback := func(batch []map[string]interface{}) error {
		for _, row := range batch {
			// Converter map para slice de strings para CSV
			record := []string{
				fmt.Sprintf("%v", row["id"]),
				fmt.Sprintf("%v", row["ticker"]),
				fmt.Sprintf("%v", row["assetType"]),
				fmt.Sprintf("%v", row["operationDate"]),
				fmt.Sprintf("%v", row["operationType"]),
				fmt.Sprintf("%v", row["source"]),
				fmt.Sprintf("%.2f", row["quantity"]),
				formatPrice(row["unitPrice"]),
				fmt.Sprintf("%v", row["currency"]),
				fmt.Sprintf("%v", row["reasonCode"]),
				fmt.Sprintf("%v", row["priceConfidence"]),
			}
			if err := writer.Write(record); err != nil {
				return fmt.Errorf("failed to write CSV record: %v", err)
			}
		}
		// Flush a cada lote para streaming
		writer.Flush()
		return nil
	}

	// Executar export com streaming
	if err := c.q.ExportLedgerStream(ctx, tenantID.String(), cpf, excludeB3, limit, callback); err != nil {
		// Se já começou a escrever resposta, não pode mais enviar JSON de erro
		// Log do erro e termina a resposta
		fmt.Fprintf(ctx.Writer, "\n# ERROR: %s\n", err.Error())
		return
	}
}

// formatPrice - helper para formatar preços no CSV
func formatPrice(price interface{}) string {
	if price == nil {
		return ""
	}
	if p, ok := price.(*float64); ok && p != nil {
		return fmt.Sprintf("%.2f", *p)
	}
	if p, ok := price.(float64); ok {
		return fmt.Sprintf("%.2f", p)
	}
	return fmt.Sprintf("%v", price)
}

// B3FullFetch godoc
// @Summary Executar B3 Full Fetch (Admin)
// @Description Dispara busca completa histórica da B3 por janelas mensais para um cliente específico
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Role header string true "Role do usuário (ADMIN ou SUPPORT)" example(ADMIN)
// @Param request body object true "Dados da requisição" example({"cpf": "12345678901", "from": "2020-01", "to": "2020-12", "dryRun": false})
// @Success 201 {object} map[string]interface{} "Ação solicitada com token de confirmação"
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 403 {object} map[string]string "Role insuficiente"
// @Router /admin/backoffice/actions/b3-full-fetch [post]
func (c *Controller) B3FullFetch(ctx *gin.Context) {
	c.handleSpecificAction(ctx, "B3_FULL_FETCH")
}

// B3Incremental godoc
// @Summary Executar B3 Incremental (Admin)
// @Description Dispara busca incremental da B3 para uma data específica ou última janela aberta
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Role header string true "Role do usuário (ADMIN ou SUPPORT)" example(ADMIN)
// @Param request body object true "Dados da requisição" example({"cpf": "12345678901", "date": "2020-12-01", "dryRun": false})
// @Success 201 {object} map[string]interface{} "Ação solicitada com token de confirmação"
// @Router /admin/backoffice/actions/b3-incremental [post]
func (c *Controller) B3Incremental(ctx *gin.Context) {
	c.handleSpecificAction(ctx, "B3_INCREMENTAL_FETCH")
}

// ReconScan godoc
// @Summary Executar Reconciliation Scan (Admin)
// @Description Dispara detector de inconsistências para um cliente específico
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Role header string true "Role do usuário (ADMIN ou SUPPORT)" example(ADMIN)
// @Param request body object true "Dados da requisição" example({"cpf": "12345678901", "types": ["OPENING_BALANCE_MISSING"], "dryRun": true})
// @Success 201 {object} map[string]interface{} "Ação solicitada com token de confirmação"
// @Router /admin/backoffice/actions/recon-scan [post]
func (c *Controller) ReconScan(ctx *gin.Context) {
	c.handleSpecificAction(ctx, "RECON_SCAN")
}

// AutoFix godoc
// @Summary Executar Auto Fix (Admin)
// @Description Dispara correções automáticas de inconsistências identificadas
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Dados da requisição" example({"cpf": "12345678901", "types": ["OPENING_BALANCE_MISSING"], "dryRun": false})
// @Success 201 {object} map[string]interface{} "Ação solicitada com token de confirmação"
// @Router /admin/backoffice/actions/auto-fix [post]
func (c *Controller) AutoFix(ctx *gin.Context) {
	role := ctx.GetHeader("X-Role")
	if role != "ADMIN" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN - only ADMIN can auto-fix"})
		return
	}
	c.handleSpecificAction(ctx, "AUTO_FIX")
}

// DedupScan godoc
// @Summary Executar Dedup Scan (Admin)
// @Description Gera candidatos de duplicação para análise manual
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Role header string true "Role do usuário (ADMIN ou SUPPORT)" example(ADMIN)
// @Param request body object true "Dados da requisição" example({"cpf": "12345678901", "scanMode": "BATCH", "dryRun": false})
// @Success 201 {object} map[string]interface{} "Ação solicitada com token de confirmação"
// @Router /admin/backoffice/actions/dedup-scan [post]
func (c *Controller) DedupScan(ctx *gin.Context) {
	c.handleSpecificAction(ctx, "DEDUPE_SCAN")
}

// DedupResolve godoc
// @Summary Resolver Duplicações (Admin)
// @Description Resolve candidatos de duplicação com ação específica (merge/override/ignore)
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Dados da requisição" example({"action": "MERGE", "candidateIds": ["id1", "id2"]})
// @Success 201 {object} map[string]interface{} "Ação solicitada com token de confirmação"
// @Router /admin/backoffice/actions/dedup-resolve [post]
func (c *Controller) DedupResolve(ctx *gin.Context) {
	role := ctx.GetHeader("X-Role")
	if role != "ADMIN" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN - only ADMIN can resolve dedup"})
		return
	}
	c.handleSpecificAction(ctx, "DEDUPE_RESOLVE")
}

// PolicySet godoc
// @Summary Definir Policy do Cliente (Admin)
// @Description Define modo de fonte de dados do cliente (B3_ONLY/MANUAL_ONLY/HYBRID)
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Dados da requisição" example({"cpf": "12345678901", "mode": "HYBRID", "reason": "Cliente solicitou ajustes manuais"})
// @Success 201 {object} map[string]interface{} "Ação solicitada com token de confirmação"
// @Router /admin/backoffice/actions/policy-set [post]
func (c *Controller) PolicySet(ctx *gin.Context) {
	role := ctx.GetHeader("X-Role")
	if role != "ADMIN" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN - only ADMIN can set policy"})
		return
	}
	c.handleSpecificAction(ctx, "POLICY_UPDATE")
}

// ClientReset godoc
// @Summary Reset de Cliente (Admin)
// @Description **PERIGOSO** - Reset completo ou parcial dos dados do cliente (exige dupla confirmação sempre)
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Dados da requisição" example({"cpf": "12345678901", "what": "LEDGER_ONLY", "dryRun": true})
// @Success 201 {object} map[string]interface{} "Ação solicitada com token de confirmação"
// @Router /admin/backoffice/actions/client-reset [post]
func (c *Controller) ClientReset(ctx *gin.Context) {
	role := ctx.GetHeader("X-Role")
	if role != "ADMIN" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN - only ADMIN can reset client"})
		return
	}
	c.handleSpecificAction(ctx, "CLIENT_RESET")
}

// ZeroAndRefetch godoc
// @Summary Zero and Refetch (Admin)
// @Description **PERIGOSO** - Zera ledger normalizado e reexecuta full fetch (exige dupla confirmação sempre)
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Dados da requisição" example({"cpf": "12345678901", "from": "2019-10-01", "dryRun": false})
// @Success 201 {object} map[string]interface{} "Ação solicitada com token de confirmação"
// @Router /admin/backoffice/actions/zero-and-refetch [post]
func (c *Controller) ZeroAndRefetch(ctx *gin.Context) {
	role := ctx.GetHeader("X-Role")
	if role != "ADMIN" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN - only ADMIN can zero-and-refetch"})
		return
	}
	c.handleSpecificAction(ctx, "CLIENT_ZERO_AND_REFETCH")
}

// Confirmação específica para cada ação
// ConfirmB3FullFetch godoc
// @Summary Confirmar B3 Full Fetch
// @Description Confirma e executa a ação de B3 Full Fetch previamente solicitada
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Token de confirmação" example({"confirmToken": "abc123def456"})
// @Success 200 {object} map[string]interface{} "Ação confirmada e executada"
// @Router /admin/backoffice/actions/b3-full-fetch/confirm [post]
func (c *Controller) ConfirmB3FullFetch(ctx *gin.Context) {
	c.handleSpecificConfirm(ctx, "B3_FULL_FETCH")
}

// ConfirmB3Incremental godoc
// @Summary Confirmar B3 Incremental
// @Description Confirma e executa a ação de B3 Incremental previamente solicitada
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Token de confirmação" example({"confirmToken": "abc123def456"})
// @Success 200 {object} map[string]interface{} "Ação confirmada e executada"
// @Router /admin/backoffice/actions/b3-incremental/confirm [post]
func (c *Controller) ConfirmB3Incremental(ctx *gin.Context) {
	c.handleSpecificConfirm(ctx, "B3_INCREMENTAL_FETCH")
}

// ConfirmReconScan godoc
// @Summary Confirmar Recon Scan
// @Description Confirma e executa a ação de Reconciliation Scan previamente solicitada
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Token de confirmação" example({"confirmToken": "abc123def456"})
// @Success 200 {object} map[string]interface{} "Ação confirmada e executada"
// @Router /admin/backoffice/actions/recon-scan/confirm [post]
func (c *Controller) ConfirmReconScan(ctx *gin.Context) {
	c.handleSpecificConfirm(ctx, "RECON_SCAN")
}

// ConfirmAutoFix godoc
// @Summary Confirmar Auto Fix
// @Description Confirma e executa a ação de Auto Fix previamente solicitada
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Token de confirmação" example({"confirmToken": "abc123def456"})
// @Success 200 {object} map[string]interface{} "Ação confirmada e executada"
// @Router /admin/backoffice/actions/auto-fix/confirm [post]
func (c *Controller) ConfirmAutoFix(ctx *gin.Context) {
	c.handleSpecificConfirm(ctx, "AUTO_FIX")
}

// ConfirmDedupScan godoc
// @Summary Confirmar Dedup Scan
// @Description Confirma e executa a ação de Dedup Scan previamente solicitada
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Token de confirmação" example({"confirmToken": "abc123def456"})
// @Success 200 {object} map[string]interface{} "Ação confirmada e executada"
// @Router /admin/backoffice/actions/dedup-scan/confirm [post]
func (c *Controller) ConfirmDedupScan(ctx *gin.Context) {
	c.handleSpecificConfirm(ctx, "DEDUPE_SCAN")
}

// ConfirmDedupResolve godoc
// @Summary Confirmar Dedup Resolve
// @Description Confirma e executa a ação de Dedup Resolve previamente solicitada
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Token de confirmação" example({"confirmToken": "abc123def456"})
// @Success 200 {object} map[string]interface{} "Ação confirmada e executada"
// @Router /admin/backoffice/actions/dedup-resolve/confirm [post]
func (c *Controller) ConfirmDedupResolve(ctx *gin.Context) {
	c.handleSpecificConfirm(ctx, "DEDUPE_RESOLVE")
}

// ConfirmPolicySet godoc
// @Summary Confirmar Policy Set
// @Description Confirma e executa a ação de Policy Set previamente solicitada
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Token de confirmação" example({"confirmToken": "abc123def456"})
// @Success 200 {object} map[string]interface{} "Ação confirmada e executada"
// @Router /admin/backoffice/actions/policy-set/confirm [post]
func (c *Controller) ConfirmPolicySet(ctx *gin.Context) {
	c.handleSpecificConfirm(ctx, "POLICY_UPDATE")
}

// ConfirmClientReset godoc
// @Summary Confirmar Client Reset
// @Description Confirma e executa a ação de Client Reset previamente solicitada (PERIGOSA)
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Token de confirmação" example({"confirmToken": "abc123def456"})
// @Success 200 {object} map[string]interface{} "Ação confirmada e executada"
// @Router /admin/backoffice/actions/client-reset/confirm [post]
func (c *Controller) ConfirmClientReset(ctx *gin.Context) {
	c.handleSpecificConfirm(ctx, "CLIENT_RESET")
}

// ConfirmZeroAndRefetch godoc
// @Summary Confirmar Zero and Refetch
// @Description Confirma e executa a ação de Zero and Refetch previamente solicitada (PERIGOSA)
// @Tags Admin Backoffice
// @Accept json
// @Produce json
// @Param X-Role header string true "Role do usuário (ADMIN)" example(ADMIN)
// @Param request body object true "Token de confirmação" example({"confirmToken": "abc123def456"})
// @Success 200 {object} map[string]interface{} "Ação confirmada e executada"
// @Router /admin/backoffice/actions/zero-and-refetch/confirm [post]
func (c *Controller) ConfirmZeroAndRefetch(ctx *gin.Context) {
	c.handleSpecificConfirm(ctx, "CLIENT_ZERO_AND_REFETCH")
}

// handleSpecificAction - handler genérico para ações específicas
func (c *Controller) handleSpecificAction(ctx *gin.Context, actionType string) {
	start := time.Now()
	defer func() {
		status := "REQUESTED"
		if ctx.Writer.Status() >= 400 {
			status = "ERROR"
		}
		observability.ObserveAdminAction(actionType, status, start)
	}()

	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	role := ctx.GetHeader("X-Role")
	if role != "ADMIN" && role != "SUPPORT" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}

	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	cpf, ok := body["cpf"].(string)
	if !ok || cpf == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cpf required"})
		return
	}

	requestedBy := ctx.GetHeader("X-User-ID")
	if requestedBy == "" {
		requestedBy = "admin-system"
	}

	ttl := 600 // 10 minutes default
	if ttlVal, ok := body["ttl"].(float64); ok {
		ttl = int(ttlVal)
	}

	id, token, err := c.a.Request(ctx, tenantID.String(), cpf, actionType, requestedBy, ttl, body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"id": id, "confirmToken": token, "action": actionType})
}

// handleSpecificConfirm - handler genérico para confirmações específicas
func (c *Controller) handleSpecificConfirm(ctx *gin.Context, actionType string) {
	start := time.Now()
	defer func() {
		status := "CONFIRMED"
		if ctx.Writer.Status() >= 400 {
			status = "ERROR"
		}
		observability.ObserveAdminAction(actionType, status, start)
	}()

	role := ctx.GetHeader("X-Role")
	if role != "ADMIN" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN - only ADMIN can confirm actions"})
		return
	}

	var body struct {
		ID           string `json:"id,omitempty"`
		ConfirmToken string `json:"confirmToken"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	// Se não passou ID no body, tenta pegar da query param (compatibilidade)
	if body.ID == "" {
		body.ID = ctx.Query("id")
	}

	if body.ID == "" || body.ConfirmToken == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id and confirmToken required"})
		return
	}

	if err := c.a.Confirm(ctx, uuidFrom(body.ID), body.ConfirmToken); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "confirmed", "action": actionType})
}
