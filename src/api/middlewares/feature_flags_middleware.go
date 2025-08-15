package middlewares

import (
	"context"

	"suno-wallets/src/shared/features"

	"github.com/gin-gonic/gin"
)

// FeatureFlagsMiddleware injeta feature flags no contexto da request
type FeatureFlagsMiddleware struct {
	flagManager *features.FeatureFlagManager
}

// NewFeatureFlagsMiddleware cria nova instância do middleware
func NewFeatureFlagsMiddleware(flagManager *features.FeatureFlagManager) *FeatureFlagsMiddleware {
	return &FeatureFlagsMiddleware{
		flagManager: flagManager,
	}
}

// Handler retorna o middleware Gin para feature flags
func (ffm *FeatureFlagsMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Criar contexto de avaliação baseado na request
		evalCtx := &features.EvaluationContext{
			TenantID: c.GetString("tenant_id"), // Set pelo tenant middleware
			UserID:   c.GetString("user_id"),   // Se disponível
			CPF:      c.Query("cpf"),           // Para requests com CPF
		}

		// Adicionar custom properties se necessário
		evalCtx.Custom = make(map[string]interface{})
		if route := c.FullPath(); route != "" {
			evalCtx.Custom["route"] = route
		}
		if method := c.Request.Method; method != "" {
			evalCtx.Custom["method"] = method
		}

		// Criar hash para distribuição consistente
		if evalCtx.TenantID != "" {
			evalCtx.Hash = evalCtx.TenantID
			if evalCtx.UserID != "" {
				evalCtx.Hash += ":" + evalCtx.UserID
			}
		}

		// Injetar feature flags manager e contexto no request context
		type contextKey string
		const (
			featureFlagsManagerKey contextKey = "feature_flags_manager"
			featureFlagsContextKey contextKey = "feature_flags_context"
		)

		ctx := context.WithValue(c.Request.Context(), featureFlagsManagerKey, ffm.flagManager)
		ctx = context.WithValue(ctx, featureFlagsContextKey, evalCtx)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// IsFeatureEnabled helper function para usar em controllers
func IsFeatureEnabled(c *gin.Context, flagName string) bool {
	manager := GetFeatureFlagManager(c)
	evalCtx := GetFeatureFlagContext(c)

	if manager == nil || evalCtx == nil {
		return false
	}

	return manager.IsEnabled(c.Request.Context(), flagName, evalCtx)
}

// GetFeatureConfig helper function para obter configuração de feature
func GetFeatureConfig(c *gin.Context, flagName string) map[string]interface{} {
	manager := GetFeatureFlagManager(c)
	evalCtx := GetFeatureFlagContext(c)

	if manager == nil || evalCtx == nil {
		return nil
	}

	return manager.GetConfig(c.Request.Context(), flagName, evalCtx)
}

// GetFeatureFlagManager extrai o manager do contexto
func GetFeatureFlagManager(c *gin.Context) *features.FeatureFlagManager {
	if manager := c.Request.Context().Value("feature_flags_manager"); manager != nil {
		if ffm, ok := manager.(*features.FeatureFlagManager); ok {
			return ffm
		}
	}
	return nil
}

// GetFeatureFlagContext extrai o contexto de avaliação
func GetFeatureFlagContext(c *gin.Context) *features.EvaluationContext {
	if evalCtx := c.Request.Context().Value("feature_flags_context"); evalCtx != nil {
		if ctx, ok := evalCtx.(*features.EvaluationContext); ok {
			return ctx
		}
	}
	return nil
}
