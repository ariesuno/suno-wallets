package controllers

import (
	"net/http"
	"strconv"

	"suno-wallets/src/api/middlewares"
	"suno-wallets/src/application/dtos"
	"suno-wallets/src/application/usecases"
	"suno-wallets/src/domain/entities"
	"suno-wallets/src/shared/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WalletController controller para operações com carteiras
type WalletController struct {
	walletUseCase usecases.WalletUseCase
}

// NewWalletController cria uma nova instância do controller de carteiras
func NewWalletController(walletUseCase usecases.WalletUseCase) *WalletController {
	return &WalletController{
		walletUseCase: walletUseCase,
	}
}

// CreateWallet cria uma nova carteira
// @Summary Criar carteira
// @Description Cria uma nova carteira digital para um usuário
// @Tags wallets
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino"
// @Param wallet body dtos.CreateWalletRequest true "Dados da carteira"
// @Success 201 {object} dtos.WalletResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /wallets [post]
func (ctrl *WalletController) CreateWallet(c *gin.Context) {
	tenantID := middlewares.MustGetTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	var req dtos.CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.LogWarn("Dados inválidos para criação de carteira", map[string]interface{}{
			"tenant_id": tenantID,
			"error":     err.Error(),
		})

		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_REQUEST",
			"message": "Dados de entrada inválidos",
			"details": err.Error(),
		})
		return
	}

	// Definir tenant ID da requisição
	req.TenantID = tenantID

	// Simular usuário logado (implementar autenticação posteriormente)
	if req.CreatedBy == uuid.Nil {
		req.CreatedBy = uuid.New()
	}

	wallet, err := ctrl.walletUseCase.CreateWallet(c.Request.Context(), &req)
	if err != nil {
		ctrl.handleError(c, err, "Falha ao criar carteira")
		return
	}

	c.JSON(http.StatusCreated, wallet)
}

// GetWallet busca uma carteira por ID
// @Summary Buscar carteira
// @Description Busca uma carteira específica por ID
// @Tags wallets
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino"
// @Param id path string true "ID da carteira"
// @Success 200 {object} dtos.WalletResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /wallets/{id} [get]
func (ctrl *WalletController) GetWallet(c *gin.Context) {
	tenantID := middlewares.MustGetTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_WALLET_ID",
			"message": "ID da carteira inválido",
		})
		return
	}

	wallet, err := ctrl.walletUseCase.GetWalletByID(c.Request.Context(), walletID, tenantID)
	if err != nil {
		ctrl.handleError(c, err, "Falha ao buscar carteira")
		return
	}

	c.JSON(http.StatusOK, wallet)
}

// GetWalletsByOwner busca carteiras por proprietário
// @Summary Buscar carteiras por proprietário
// @Description Busca todas as carteiras de um proprietário específico
// @Tags wallets
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino"
// @Param owner_id path string true "ID do proprietário"
// @Success 200 {array} dtos.WalletResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /wallets/owner/{owner_id} [get]
func (ctrl *WalletController) GetWalletsByOwner(c *gin.Context) {
	tenantID := middlewares.MustGetTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	ownerID, err := uuid.Parse(c.Param("owner_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_OWNER_ID",
			"message": "ID do proprietário inválido",
		})
		return
	}

	wallets, err := ctrl.walletUseCase.GetWalletsByOwner(c.Request.Context(), ownerID, tenantID)
	if err != nil {
		ctrl.handleError(c, err, "Falha ao buscar carteiras por proprietário")
		return
	}

	c.JSON(http.StatusOK, wallets)
}

// UpdateWallet atualiza uma carteira existente
// @Summary Atualizar carteira
// @Description Atualiza uma carteira existente
// @Tags wallets
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino"
// @Param id path string true "ID da carteira"
// @Param wallet body dtos.UpdateWalletRequest true "Dados para atualização"
// @Success 200 {object} dtos.WalletResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /wallets/{id} [put]
func (ctrl *WalletController) UpdateWallet(c *gin.Context) {
	tenantID := middlewares.MustGetTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_WALLET_ID",
			"message": "ID da carteira inválido",
		})
		return
	}

	var req dtos.UpdateWalletRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		helpers.LogWarn("Dados inválidos para atualização de carteira", map[string]interface{}{
			"tenant_id": tenantID,
			"wallet_id": walletID,
			"error":     bindErr.Error(),
		})

		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_REQUEST",
			"message": "Dados de entrada inválidos",
			"details": bindErr.Error(),
		})
		return
	}

	// Definir IDs da requisição
	req.ID = walletID
	req.TenantID = tenantID

	// Simular usuário logado
	userID := uuid.New()
	req.UpdatedBy = &userID

	wallet, err := ctrl.walletUseCase.UpdateWallet(c.Request.Context(), &req)
	if err != nil {
		ctrl.handleError(c, err, "Falha ao atualizar carteira")
		return
	}

	c.JSON(http.StatusOK, wallet)
}

// DeleteWallet remove uma carteira
// @Summary Deletar carteira
// @Description Remove uma carteira (soft delete)
// @Tags wallets
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino"
// @Param id path string true "ID da carteira"
// @Success 204
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /wallets/{id} [delete]
func (ctrl *WalletController) DeleteWallet(c *gin.Context) {
	tenantID := middlewares.MustGetTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_WALLET_ID",
			"message": "ID da carteira inválido",
		})
		return
	}

	err = ctrl.walletUseCase.DeleteWallet(c.Request.Context(), walletID, tenantID)
	if err != nil {
		ctrl.handleError(c, err, "Falha ao deletar carteira")
		return
	}

	c.Status(http.StatusNoContent)
}

// ListWallets lista carteiras com filtros e paginação
// @Summary Listar carteiras
// @Description Lista carteiras com filtros opcionais e paginação
// @Tags wallets
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino"
// @Param owner_id query string false "ID do proprietário"
// @Param status query string false "Status da carteira"
// @Param type query string false "Tipo da carteira"
// @Param page query int false "Página" default(1)
// @Param limit query int false "Limite por página" default(10)
// @Param search query string false "Busca por nome"
// @Param sort_by query string false "Campo para ordenação"
// @Param sort_desc query bool false "Ordenação decrescente"
// @Success 200 {object} dtos.ListWalletsResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /wallets [get]
func (ctrl *WalletController) ListWallets(c *gin.Context) {
	tenantID := middlewares.MustGetTenantID(c)
	if tenantID == uuid.Nil {
		return
	}

	req := dtos.ListWalletsRequest{
		TenantID: tenantID,
		Page:     1,
		Limit:    10,
	}

	// Parse query parameters
	if ownerIDStr := c.Query("owner_id"); ownerIDStr != "" {
		if ownerID, err := uuid.Parse(ownerIDStr); err == nil {
			req.OwnerID = &ownerID
		}
	}

	if statusStr := c.Query("status"); statusStr != "" {
		status := entities.WalletStatus(statusStr)
		req.Status = &status
	}

	if typeStr := c.Query("type"); typeStr != "" {
		walletType := entities.WalletType(typeStr)
		req.Type = &walletType
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			req.Limit = limit
		}
	}

	req.Search = c.Query("search")
	req.SortBy = c.Query("sort_by")
	req.SortDesc = c.Query("sort_desc") == "true"

	response, err := ctrl.walletUseCase.ListWallets(c.Request.Context(), &req)
	if err != nil {
		ctrl.handleError(c, err, "Falha ao listar carteiras")
		return
	}

	c.JSON(http.StatusOK, response)
}

// handleError trata erros de forma padronizada
func (ctrl *WalletController) handleError(c *gin.Context, err error, message string) {
	helpers.LogError(message, err, map[string]interface{}{
		"path":   c.Request.URL.Path,
		"method": c.Request.Method,
	})

	// Verificar tipo de erro
	if entities.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": err.Error(),
		})
		return
	}

	if entities.IsNotFoundError(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "NOT_FOUND",
			"message": err.Error(),
		})
		return
	}

	if entities.IsBusinessError(err) {
		businessErr := err.(entities.BusinessError)
		statusCode := http.StatusBadRequest

		// Mapear códigos específicos para status HTTP apropriados
		switch businessErr.Code {
		case "DUPLICATE_DEFAULT_WALLET":
			statusCode = http.StatusConflict
		case "INSUFFICIENT_BALANCE":
			statusCode = http.StatusBadRequest
		case "UNAUTHORIZED_ACCESS":
			statusCode = http.StatusForbidden
		}

		c.JSON(statusCode, gin.H{
			"error":   businessErr.Code,
			"message": businessErr.Message,
		})
		return
	}

	if entities.IsConflictError(err) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "CONFLICT",
			"message": err.Error(),
		})
		return
	}

	// Erro genérico
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "INTERNAL_ERROR",
		"message": "Erro interno do servidor",
	})
}
