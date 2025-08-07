package middlewares

import (
	"net/http"

	"suno-wallets/src/shared/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	TenantIDHeader = "X-Tenant-ID"
	TenantIDKey    = "tenant_id"
)

// TenantMiddleware middleware para extrair e validar tenant ID
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantIDStr := c.GetHeader(TenantIDHeader)

		// Verificar se o header está presente
		if tenantIDStr == "" {
			helpers.LogWarn("Header X-Tenant-ID não fornecido", map[string]interface{}{
				"ip":     c.ClientIP(),
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
			})

			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "MISSING_TENANT_ID",
				"message": "Header X-Tenant-ID é obrigatório",
			})
			c.Abort()
			return
		}

		// Validar formato UUID
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			helpers.LogWarn("Header X-Tenant-ID inválido", map[string]interface{}{
				"tenant_id_header": tenantIDStr,
				"ip":               c.ClientIP(),
				"method":           c.Request.Method,
				"path":             c.Request.URL.Path,
			})

			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_TENANT_ID",
				"message": "Header X-Tenant-ID deve ser um UUID válido",
			})
			c.Abort()
			return
		}

		// Armazenar tenant ID no contexto
		c.Set(TenantIDKey, tenantID)

		// Log da requisição com tenant
		helpers.LogInfo("Requisição autenticada por tenant", map[string]interface{}{
			"tenant_id": tenantID,
			"ip":        c.ClientIP(),
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
		})

		c.Next()
	}
}

// GetTenantID extrai o tenant ID do contexto
func GetTenantID(c *gin.Context) (uuid.UUID, bool) {
	tenantID, exists := c.Get(TenantIDKey)
	if !exists {
		return uuid.Nil, false
	}

	id, ok := tenantID.(uuid.UUID)
	return id, ok
}

// MustGetTenantID extrai o tenant ID do contexto ou retorna erro
func MustGetTenantID(c *gin.Context) uuid.UUID {
	tenantID, ok := GetTenantID(c)
	if !ok {
		helpers.LogError("Tenant ID não encontrado no contexto", nil, map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_ERROR",
			"message": "Erro interno do servidor",
		})
		c.Abort()
		return uuid.Nil
	}

	return tenantID
}
