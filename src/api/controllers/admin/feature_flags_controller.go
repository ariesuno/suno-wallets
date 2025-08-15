package admin

import (
	"net/http"
	"time"

	"suno-wallets/src/shared/features"
	"suno-wallets/src/shared/helpers"

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
	flags, err := ffc.flagManager.ListFlags(c.Request.Context())
	if err != nil {
		helpers.LogError("failed to list feature flags", err, map[string]interface{}{
			"tenant_id": c.GetString("tenant_id"),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list feature flags",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"flags": flags,
		"count": len(flags),
	})
}

// GetFlag obtém uma feature flag específica
func (ffc *FeatureFlagsController) GetFlag(c *gin.Context) {
	flagName := c.Param("name")
	if flagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "flag name is required",
		})
		return
	}

	flag, err := ffc.flagManager.GetFlag(c.Request.Context(), flagName)
	if err != nil {
		helpers.LogError("failed to get feature flag", err, map[string]interface{}{
			"flag_name": flagName,
			"tenant_id": c.GetString("tenant_id"),
		})
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

	c.JSON(http.StatusOK, flag)
}

// CreateFlag cria uma nova feature flag
func (ffc *FeatureFlagsController) CreateFlag(c *gin.Context) {
	var req CreateFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// Verificar se flag já existe
	existing, err := ffc.flagManager.GetFlag(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to check existing flag",
		})
		return
	}

	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "feature flag already exists",
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
		CreatedBy:   getUserID(c), // TODO: Implementar extração de user
	}

	if err := ffc.flagManager.SetFlag(c.Request.Context(), flag); err != nil {
		helpers.LogError("failed to create feature flag", err, map[string]interface{}{
			"flag_name": req.Name,
			"tenant_id": c.GetString("tenant_id"),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create feature flag",
		})
		return
	}

	helpers.LogInfo("feature flag created", map[string]interface{}{
		"flag_name":  req.Name,
		"enabled":    req.Enabled,
		"rollout":    req.Rollout,
		"tenant_id":  c.GetString("tenant_id"),
		"created_by": flag.CreatedBy,
	})

	c.JSON(http.StatusCreated, flag)
}

// UpdateFlag atualiza uma feature flag existente
func (ffc *FeatureFlagsController) UpdateFlag(c *gin.Context) {
	flagName := c.Param("name")
	if flagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "flag name is required",
		})
		return
	}

	var req UpdateFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
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
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "rollout must be between 0 and 100",
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
		helpers.LogError("failed to update feature flag", err, map[string]interface{}{
			"flag_name": flagName,
			"tenant_id": c.GetString("tenant_id"),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update feature flag",
		})
		return
	}

	helpers.LogInfo("feature flag updated", map[string]interface{}{
		"flag_name":  flagName,
		"enabled":    flag.Enabled,
		"rollout":    flag.Rollout,
		"tenant_id":  c.GetString("tenant_id"),
		"updated_by": flag.CreatedBy,
	})

	c.JSON(http.StatusOK, flag)
}

// DeleteFlag remove uma feature flag
func (ffc *FeatureFlagsController) DeleteFlag(c *gin.Context) {
	flagName := c.Param("name")
	if flagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "flag name is required",
		})
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
		helpers.LogError("failed to delete feature flag", err, map[string]interface{}{
			"flag_name": flagName,
			"tenant_id": c.GetString("tenant_id"),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete feature flag",
		})
		return
	}

	helpers.LogInfo("feature flag deleted", map[string]interface{}{
		"flag_name":  flagName,
		"tenant_id":  c.GetString("tenant_id"),
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
	flagName := c.Param("name")
	if flagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "flag name is required",
		})
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
		helpers.LogError("failed to toggle feature flag", err, map[string]interface{}{
			"flag_name": flagName,
			"tenant_id": c.GetString("tenant_id"),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to toggle feature flag",
		})
		return
	}

	helpers.LogInfo("feature flag toggled", map[string]interface{}{
		"flag_name":  flagName,
		"enabled":    flag.Enabled,
		"tenant_id":  c.GetString("tenant_id"),
		"toggled_by": flag.CreatedBy,
	})

	c.JSON(http.StatusOK, gin.H{
		"flag_name": flagName,
		"enabled":   flag.Enabled,
		"message":   "feature flag toggled successfully",
	})
}

// getUserID extrai user ID do contexto (placeholder)
func getUserID(c *gin.Context) string {
	// TODO: Implementar extração real do user ID
	if userID := c.GetString("user_id"); userID != "" {
		return userID
	}
	return "system"
}
