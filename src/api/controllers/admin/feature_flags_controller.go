package admin

import (
	"net/http"
	"time"

	"suno-wallets/src/api/controllers/common"
	"suno-wallets/src/shared/features"

	"github.com/gin-gonic/gin"
)

// FeatureFlagsController gerencia feature flags via API
type FeatureFlagsController struct {
	flagManager *features.FeatureFlagManager
}

// NewFeatureFlagsController cria nova instância do controller
func NewFeatureFlagsController(flagManager *features.FeatureFlagManager) *FeatureFlagsController {
	return &FeatureFlagsController{
		flagManager: flagManager,
	}
}

// CreateFlagRequest representa request para criar flag
type CreateFlagRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Enabled     bool                   `json:"enabled"`
	Description string                 `json:"description" binding:"required"`
	Rollout     int                    `json:"rollout" binding:"min=0,max=100"`
	Config      map[string]interface{} `json:"config"`
	Conditions  []features.Condition   `json:"conditions"`
}

// UpdateFlagRequest representa request para atualizar flag
type UpdateFlagRequest struct {
	Enabled     *bool                  `json:"enabled"`
	Description *string                `json:"description"`
	Rollout     *int                   `json:"rollout"`
	Config      map[string]interface{} `json:"config"`
	Conditions  []features.Condition   `json:"conditions"`
}

// EvaluateFlagRequest representa request para avaliar flag
type EvaluateFlagRequest struct {
	FlagName string                 `json:"flag_name" binding:"required"`
	TenantID string                 `json:"tenant_id"`
	UserID   string                 `json:"user_id"`
	CPF      string                 `json:"cpf"`
	Custom   map[string]interface{} `json:"custom"`
}

// ListFlags lista todas as feature flags
func (ffc *FeatureFlagsController) ListFlags(c *gin.Context) {
	logger := common.NewControllerLogger(c)
	logger.LogRequest("list_feature_flags")

	flags, err := ffc.flagManager.ListFlags(c.Request.Context())
	if err != nil {
		common.HandleError(c, err, "Falha ao listar feature flags")
		return
	}

	logger.LogBusinessEvent("feature_flags_listed", "feature_flag", "multiple", map[string]interface{}{
		"flags_count": len(flags),
	})

	c.JSON(http.StatusOK, gin.H{
		"flags": flags,
		"count": len(flags),
	})
}

// GetFlag obtém uma feature flag específica
func (ffc *FeatureFlagsController) GetFlag(c *gin.Context) {
	logger := common.NewControllerLogger(c)
	flagName := c.Param("name")

	if flagName == "" {
		common.HandleBadRequest(c, "Nome da feature flag é obrigatório", nil)
		return
	}

	logger.LogRequest("get_feature_flag", map[string]interface{}{
		"flag_name": flagName,
	})

	flag, err := ffc.flagManager.GetFlag(c.Request.Context(), flagName)
	if err != nil {
		common.HandleError(c, err, "Falha ao buscar feature flag", map[string]interface{}{
			"flag_name": flagName,
		})
		return
	}

	if flag == nil {
		common.HandleBadRequest(c, "Feature flag não encontrada", map[string]interface{}{
			"flag_name": flagName,
		})
		return
	}

	logger.LogBusinessEvent("feature_flag_retrieved", "feature_flag", flagName)
	c.JSON(http.StatusOK, flag)
}

// CreateFlag cria uma nova feature flag
func (ffc *FeatureFlagsController) CreateFlag(c *gin.Context) {
	logger := common.NewControllerLogger(c)

	var req CreateFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.HandleValidationError(c, err, "Dados inválidos para criação de feature flag")
		return
	}

	logger.LogRequest("create_feature_flag", map[string]interface{}{
		"flag_name": req.Name,
		"enabled":   req.Enabled,
		"rollout":   req.Rollout,
	})

	// Verificar se flag já existe
	existing, err := ffc.flagManager.GetFlag(c.Request.Context(), req.Name)
	if err != nil {
		common.HandleError(c, err, "Falha ao verificar feature flag existente", map[string]interface{}{
			"flag_name": req.Name,
		})
		return
	}

	if existing != nil {
		common.HandleBadRequest(c, "Feature flag já existe", map[string]interface{}{
			"flag_name": req.Name,
		})
		return
	}

	// Criar flag
	flag := &features.FeatureFlag{
		Name:        req.Name,
		Enabled:     req.Enabled,
		Description: req.Description,
		Rollout:     req.Rollout,
		Config:      req.Config,
		Conditions:  req.Conditions,
		CreatedBy:   getUserID(c),
	}

	if err := ffc.flagManager.SetFlag(c.Request.Context(), flag); err != nil {
		common.HandleError(c, err, "Falha ao criar feature flag", map[string]interface{}{
			"flag_name": req.Name,
		})
		return
	}

	logger.LogBusinessEvent("feature_flag_created", "feature_flag", req.Name, map[string]interface{}{
		"enabled":    req.Enabled,
		"rollout":    req.Rollout,
		"created_by": flag.CreatedBy,
	})

	c.JSON(http.StatusCreated, flag)
}

// UpdateFlag atualiza uma feature flag existente
func (ffc *FeatureFlagsController) UpdateFlag(c *gin.Context) {
	logger := common.NewControllerLogger(c)
	flagName := c.Param("name")
	if flagName == "" {
		common.HandleBadRequest(c, "Nome da feature flag é obrigatório", nil)
		return
	}

	var req UpdateFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.HandleValidationError(c, err, "Dados inválidos para atualização de feature flag")
		return
	}

	// Buscar flag existente
	flag, err := ffc.flagManager.GetFlag(c.Request.Context(), flagName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get feature flag",
		})
		return
	}

	if flag == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "feature flag not found",
		})
		return
	}

	// Atualizar campos fornecidos
	if req.Enabled != nil {
		flag.Enabled = *req.Enabled
	}
	if req.Description != nil {
		flag.Description = *req.Description
	}
	if req.Rollout != nil {
		if *req.Rollout < 0 || *req.Rollout > 100 {
			common.HandleBadRequest(c, "Valor de rollout deve estar entre 0 e 100", map[string]interface{}{
				"provided_rollout": *req.Rollout,
			})
			return
		}
		flag.Rollout = *req.Rollout
	}
	if req.Config != nil {
		flag.Config = req.Config
	}
	if req.Conditions != nil {
		flag.Conditions = req.Conditions
	}

	flag.CreatedBy = getUserID(c) // Atualizar quem modificou

	if err := ffc.flagManager.SetFlag(c.Request.Context(), flag); err != nil {
		common.HandleError(c, err, "Falha ao atualizar feature flag", map[string]interface{}{
			"flag_name": flagName,
		})
		return
	}

	logger.LogBusinessEvent("feature_flag_updated", "feature_flag", flagName, map[string]interface{}{
		"enabled":    flag.Enabled,
		"rollout":    flag.Rollout,
		"updated_by": flag.CreatedBy,
	})

	c.JSON(http.StatusOK, flag)
}

// DeleteFlag remove uma feature flag
func (ffc *FeatureFlagsController) DeleteFlag(c *gin.Context) {
	logger := common.NewControllerLogger(c)
	flagName := c.Param("name")
	if flagName == "" {
		common.HandleBadRequest(c, "Nome da feature flag é obrigatório", nil)
		return
	}

	// Verificar se flag existe
	flag, err := ffc.flagManager.GetFlag(c.Request.Context(), flagName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get feature flag",
		})
		return
	}

	if flag == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "feature flag not found",
		})
		return
	}

	if err := ffc.flagManager.DeleteFlag(c.Request.Context(), flagName); err != nil {
		common.HandleError(c, err, "Falha ao deletar feature flag", map[string]interface{}{
			"flag_name": flagName,
		})
		return
	}

	logger.LogBusinessEvent("feature_flag_deleted", "feature_flag", flagName, map[string]interface{}{
		"deleted_by": getUserID(c),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "feature flag deleted successfully",
	})
}

// EvaluateFlag avalia uma feature flag para contexto específico
func (ffc *FeatureFlagsController) EvaluateFlag(c *gin.Context) {
	var req EvaluateFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// Criar contexto de avaliação
	evalCtx := &features.EvaluationContext{
		TenantID: req.TenantID,
		UserID:   req.UserID,
		CPF:      req.CPF,
		Custom:   req.Custom,
	}

	if evalCtx.TenantID != "" {
		evalCtx.Hash = evalCtx.TenantID
		if evalCtx.UserID != "" {
			evalCtx.Hash += ":" + evalCtx.UserID
		}
	}

	// Avaliar flag
	enabled := ffc.flagManager.IsEnabled(c.Request.Context(), req.FlagName, evalCtx)
	config := ffc.flagManager.GetConfig(c.Request.Context(), req.FlagName, evalCtx)

	c.JSON(http.StatusOK, gin.H{
		"flag_name":    req.FlagName,
		"enabled":      enabled,
		"config":       config,
		"evaluated_at": time.Now(),
		"context":      evalCtx,
	})
}

// ToggleFlag facilita habilitar/desabilitar uma flag rapidamente
func (ffc *FeatureFlagsController) ToggleFlag(c *gin.Context) {
	logger := common.NewControllerLogger(c)
	flagName := c.Param("name")
	if flagName == "" {
		common.HandleBadRequest(c, "Nome da feature flag é obrigatório", nil)
		return
	}

	// Buscar flag existente
	flag, err := ffc.flagManager.GetFlag(c.Request.Context(), flagName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get feature flag",
		})
		return
	}

	if flag == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "feature flag not found",
		})
		return
	}

	// Toggle enabled state
	flag.Enabled = !flag.Enabled
	flag.CreatedBy = getUserID(c)

	if err := ffc.flagManager.SetFlag(c.Request.Context(), flag); err != nil {
		common.HandleError(c, err, "Falha ao alterar feature flag", map[string]interface{}{
			"flag_name": flagName,
		})
		return
	}

	logger.LogBusinessEvent("feature_flag_toggled", "feature_flag", flagName, map[string]interface{}{
		"enabled":    flag.Enabled,
		"toggled_by": flag.CreatedBy,
	})

	c.JSON(http.StatusOK, gin.H{
		"flag_name": flagName,
		"enabled":   flag.Enabled,
		"message":   "feature flag toggled successfully",
	})
}

// getUserID extrai user ID do contexto (implementação real)
func getUserID(c *gin.Context) string {
	// Tenta extrair do middleware de autenticação
	if userID := c.GetString("user_id"); userID != "" {
		return userID
	}

	// Tenta extrair do header Authorization ou outros headers customizados
	if userID := c.GetHeader("X-User-ID"); userID != "" {
		return userID
	}

	// Tenta extrair claims de JWT se disponível
	if claims, exists := c.Get("user_claims"); exists {
		if claimsMap, ok := claims.(map[string]interface{}); ok {
			if userID, ok := claimsMap["user_id"].(string); ok && userID != "" {
				return userID
			}
			if sub, ok := claimsMap["sub"].(string); ok && sub != "" {
				return sub
			}
		}
	}

	// Fallback para tenant_id se disponível (para operações de sistema)
	if tenantID := c.GetString("tenant_id"); tenantID != "" {
		return "system:" + tenantID
	}

	return "system"
}
