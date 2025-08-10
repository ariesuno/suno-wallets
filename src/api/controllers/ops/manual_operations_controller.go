package ops

import (
    "net/http"
    "time"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    appops "suno-wallets/src/application/ops"
    "suno-wallets/src/api/middlewares"
)

// Comentários em pt-BR: controller CRUD para USER_MANUAL

type ManualOperationsController struct{ svc *appops.ManualOperationsService }

func NewManualOperationsController(svc *appops.ManualOperationsService) *ManualOperationsController { return &ManualOperationsController{svc: svc} }

func (c2 *ManualOperationsController) Create(ctx *gin.Context) {
    tenantID, ok := middlewares.GetTenantID(ctx)
    if !ok { ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"}); return }
    var body struct {
        CPF string `json:"cpf"`
        Ticker string `json:"ticker"`
        AssetType string `json:"assetType"`
        OperationDate string `json:"operationDate"`
        OperationType string `json:"operationType"`
        Quantity float64 `json:"quantity"`
        UnitPrice *float64 `json:"unitPrice"`
        Currency string `json:"currency"`
    }
    if err := ctx.ShouldBindJSON(&body); err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"}); return }
    d, _ := time.Parse("2006-01-02", body.OperationDate)
    id, err := c2.svc.Create(ctx, appops.ManualOperation{TenantID: tenantID, CPF: body.CPF, Ticker: body.Ticker, AssetType: body.AssetType, OperationDate: d, OperationType: body.OperationType, Quantity: body.Quantity, UnitPrice: body.UnitPrice, Currency: body.Currency})
    if err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    ctx.JSON(http.StatusOK, gin.H{"id": id})
}

func (c2 *ManualOperationsController) Update(ctx *gin.Context) {
    tenantID, ok := middlewares.GetTenantID(ctx)
    if !ok { ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"}); return }
    id, err := uuid.Parse(ctx.Param("id"))
    if err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    var body struct {
        CPF string `json:"cpf"`
        Ticker string `json:"ticker"`
        AssetType string `json:"assetType"`
        OperationDate string `json:"operationDate"`
        OperationType string `json:"operationType"`
        Quantity float64 `json:"quantity"`
        UnitPrice *float64 `json:"unitPrice"`
        Currency string `json:"currency"`
    }
    if err := ctx.ShouldBindJSON(&body); err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"}); return }
    d, _ := time.Parse("2006-01-02", body.OperationDate)
    err = c2.svc.Update(ctx, appops.ManualOperation{ID: id, TenantID: tenantID, CPF: body.CPF, Ticker: body.Ticker, AssetType: body.AssetType, OperationDate: d, OperationType: body.OperationType, Quantity: body.Quantity, UnitPrice: body.UnitPrice, Currency: body.Currency})
    if err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    ctx.JSON(http.StatusOK, gin.H{"id": id})
}

func (c2 *ManualOperationsController) Delete(ctx *gin.Context) {
    tenantID, ok := middlewares.GetTenantID(ctx)
    if !ok { ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"}); return }
    id, err := uuid.Parse(ctx.Param("id"))
    if err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    if err := c2.svc.SoftDelete(ctx, tenantID, id); err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    ctx.JSON(http.StatusOK, gin.H{"deleted": true})
}


