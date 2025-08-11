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
	prof, err := c.q.Profile(ctx, tenantID, cpf)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, prof)
}

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
	id, token, err := c.a.Request(ctx, tenantID, body.CPF, body.Action, body.RequestedBy, body.TTL, body.Payload)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"id": id, "confirmToken": token})
}

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

// Search: GET /admin/backoffice/search?query=&limit=
func (c *Controller) Search(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	q := ctx.Query("query")
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	items, err := c.q.Search(ctx, tenantID, q, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

// List actions: GET /admin/backoffice/actions?cpf=&action=&status=&page=&pageSize=
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
	items, err := c.q.ListActions(ctx, tenantID, cpf, action, status, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items, "page": page, "pageSize": pageSize})
}

// Get action detail: GET /admin/backoffice/actions/:id
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
	item, err := c.q.GetAction(ctx, tenantID, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

// Export ledger: GET /admin/backoffice/export/ledger?cpf=&limit=&excludeB3=false
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
	items, err := c.q.ExportLedger(ctx, tenantID, cpf, excludeB3, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}
