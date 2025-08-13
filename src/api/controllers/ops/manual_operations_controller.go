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

// Create godoc
// @Summary Criar operação manual
// @Description Cria uma nova operação manual no sistema de ledger. Permite inserir transações que não foram capturadas automaticamente da B3.
// @Tags Operations
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param request body object true "Dados da operação manual" example({"cpf": "12345678901", "ticker": "PETR4", "assetType": "equity", "operationDate": "2024-01-15", "operationType": "buy", "quantity": 100, "unitPrice": 25.50, "currency": "BRL"})
// @Success 201 {object} map[string]interface{} "Operação criada com sucesso"
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 409 {object} map[string]string "Política B3_ONLY bloqueia operações manuais"
// @Router /ops/manual [post]
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

// Update godoc
// @Summary Atualizar operação manual
// @Description Atualiza uma operação manual existente identificada pelo ID.
// @Tags Operations
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param id path string true "ID da operação (UUID)" example(123e4567-e89b-12d3-a456-426614174000)
// @Param request body object true "Dados atualizados da operação" example({"cpf": "12345678901", "ticker": "PETR4", "assetType": "equity", "operationDate": "2024-01-15", "operationType": "sell", "quantity": 50, "unitPrice": 26.00, "currency": "BRL"})
// @Success 200 {object} map[string]interface{} "Operação atualizada com sucesso"
// @Failure 400 {object} map[string]string "ID ou dados inválidos"
// @Router /ops/manual/{id} [put]
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

// Delete godoc
// @Summary Excluir operação manual
// @Description Executa soft delete de uma operação manual específica.
// @Tags Operations
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param id path string true "ID da operação (UUID)" example(123e4567-e89b-12d3-a456-426614174000)
// @Success 200 {object} map[string]interface{} "Operação excluída com sucesso"
// @Failure 400 {object} map[string]string "ID inválido ou erro na exclusão"
// @Router /ops/manual/{id} [delete]
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

// List godoc
// @Summary Listar operações manuais
// @Description Retorna lista paginada de operações manuais, com filtro opcional por CPF.
// @Tags Operations
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string false "Filtrar por CPF específico" example(12345678901)
// @Success 200 {object} map[string]interface{} "Lista paginada de operações manuais"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /ops/manual [get]
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
