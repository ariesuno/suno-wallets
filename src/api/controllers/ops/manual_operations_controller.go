package ops

import (
	"net/http"
	"suno-wallets/src/api/middlewares"
	appops "suno-wallets/src/application/ops"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Comentários em pt-BR: controller CRUD para USER_MANUAL

type ManualOperationsController struct {
	svc *appops.ManualOperationsService
}

func NewManualOperationsController(svc *appops.ManualOperationsService) *ManualOperationsController {
	return &ManualOperationsController{svc: svc}
}

func (c2 *ManualOperationsController) Create(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	var body struct {
		CPF           string   `json:"cpf"`
		Ticker        string   `json:"ticker"`
		AssetType     string   `json:"assetType"`
		OperationDate string   `json:"operationDate"`
		OperationType string   `json:"operationType"`
		Quantity      float64  `json:"quantity"`
		UnitPrice     *float64 `json:"unitPrice"`
		Currency      string   `json:"currency"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	d, _ := time.Parse("2006-01-02", body.OperationDate)
	id, err := c2.svc.Create(ctx, appops.ManualOperation{TenantID: tenantID, CPF: body.CPF, Ticker: body.Ticker, AssetType: body.AssetType, OperationDate: d, OperationType: body.OperationType, Quantity: body.Quantity, UnitPrice: body.UnitPrice, Currency: body.Currency})
	if err != nil {
		if err.Error() == "POLICY_BLOCK_B3_ONLY" {
			ctx.JSON(http.StatusConflict, gin.H{"error": "B3_ONLY policy blocks USER_MANUAL creation"})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"id": id})
}

func (c2 *ManualOperationsController) Update(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		CPF           string   `json:"cpf"`
		Ticker        string   `json:"ticker"`
		AssetType     string   `json:"assetType"`
		OperationDate string   `json:"operationDate"`
		OperationType string   `json:"operationType"`
		Quantity      float64  `json:"quantity"`
		UnitPrice     *float64 `json:"unitPrice"`
		Currency      string   `json:"currency"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	d, _ := time.Parse("2006-01-02", body.OperationDate)
	err = c2.svc.Update(ctx, appops.ManualOperation{ID: id, TenantID: tenantID, CPF: body.CPF, Ticker: body.Ticker, AssetType: body.AssetType, OperationDate: d, OperationType: body.OperationType, Quantity: body.Quantity, UnitPrice: body.UnitPrice, Currency: body.Currency})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"id": id})
}

func (c2 *ManualOperationsController) Delete(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := c2.svc.SoftDelete(ctx, tenantID, id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"deleted": true})
}

// GET /ops/manual (lista paginada)
func (c2 *ManualOperationsController) List(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	cpf := ctx.Query("cpf")
	page, pageSize := 1, 50
	items, err := c2.svc.List(ctx, tenantID, cpf, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items, "page": page, "pageSize": pageSize})
}
