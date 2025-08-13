package admin

import (
	"net/http"
	"suno-wallets/src/api/middlewares"
	appadm "suno-wallets/src/application/admin"

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
