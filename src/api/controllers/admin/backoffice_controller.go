package admin

import (
	"net/http"
	"suno-wallets/src/api/middlewares"
	appadm "suno-wallets/src/application/admin"

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
	prof, err := c.q.Profile(ctx, tenantID.String(), cpf)
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

func (c *Controller) ConfirmAction(ctx *gin.Context) {
	id := ctx.Query("id")
	token := ctx.Query("token")
	if id == "" || token == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id/token required"})
		return
	}
	if err := c.a.Confirm(ctx, uuidFrom(id), token); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "confirmed"})
}

func uuidFrom(s string) uuid.UUID { u, _ := uuid.Parse(s); return u }
