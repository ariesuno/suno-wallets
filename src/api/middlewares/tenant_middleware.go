package middlewares

import (
	"net/http"
	"strings"

	"suno-wallets/src/domain/enums"
	"suno-wallets/src/shared/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	TenantIDHeader = "X-Tenant-ID"
	TenantIDKey    = "tenant_id"   // uuid.UUID (compat)
	TenantNameKey  = "tenant_name" // string canonical name (nai, status_invest, ...)
)

// TenantMiddleware middleware para extrair e validar tenant ID
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantIDStr := c.GetHeader(TenantIDHeader)

		if tenantIDStr == "" {
			helpers.LogWarn("Header X-Tenant-ID não fornecido", map[string]interface{}{
				"ip":     c.ClientIP(),
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
			})
			c.JSON(http.StatusBadRequest, gin.H{"error": "MISSING_TENANT_ID", "message": "Header X-Tenant-ID é obrigatório"})
			c.Abort()
			return
		}

		// Aceitar tanto nomes (nai, status_invest, ...) quanto UUIDs legado
		t := strings.TrimSpace(strings.ToLower(tenantIDStr))
		var tenantUUID uuid.UUID
		var tenantName string

		// Caso 1: nome de tenant
		if isValidTenantName(t) {
			tenantName = t
			tenantUUID = mapTenantNameToUUID(t)
		} else {
			// Caso 2: tentar UUID legado
			parsed, err := uuid.Parse(tenantIDStr)
			if err != nil {
				helpers.LogWarn("Header X-Tenant-ID inválido", map[string]interface{}{
					"tenant_id_header": tenantIDStr,
					"ip":               c.ClientIP(),
					"method":           c.Request.Method,
					"path":             c.Request.URL.Path,
				})
				c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_TENANT_ID", "message": "Use tenant name válido ou UUID legado"})
				c.Abort()
				return
			}
			tenantUUID = parsed
			tenantName = mapUUIDToTenantName(parsed)
		}

		// Armazenar ambos para compatibilidade
		c.Set(TenantIDKey, tenantUUID)
		c.Set(TenantNameKey, tenantName)

		helpers.LogInfo("Requisição autenticada por tenant", map[string]interface{}{
			"tenant_id":   tenantUUID.String(),
			"tenant_name": tenantName,
			"ip":          c.ClientIP(),
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
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
	if !ok {
		return uuid.Nil, false
	}
	return id, true
}

// MustGetTenantID extrai o tenant ID do contexto ou retorna erro
func MustGetTenantID(c *gin.Context) uuid.UUID {
	tenantID, ok := GetTenantID(c)
	if !ok {
		helpers.LogError("Tenant ID não encontrado no contexto", nil, map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "INTERNAL_ERROR", "message": "Erro interno do servidor"})
		c.Abort()
		return uuid.Nil
	}
	return tenantID
}

// Funções auxiliares para trabalhar com tenant name (string)
func GetTenantName(c *gin.Context) (string, bool) {
	tn, ok := c.Get(TenantNameKey)
	if !ok {
		return "", false
	}
	s, ok := tn.(string)
	return s, ok
}

func MustGetTenantName(c *gin.Context) string {
	if s, ok := GetTenantName(c); ok && s != "" {
		return s
	}
	// fallback: tentar derivar a partir do UUID
	if id, ok := GetTenantID(c); ok && id != uuid.Nil {
		return mapUUIDToTenantName(id)
	}
	helpers.LogError("Tenant Name não encontrado no contexto", nil, map[string]interface{}{
		"path":   c.Request.URL.Path,
		"method": c.Request.Method,
	})
	c.JSON(http.StatusInternalServerError, gin.H{"error": "INTERNAL_ERROR", "message": "Erro interno do servidor"})
	c.Abort()
	return ""
}

func isValidTenantName(v string) bool {
	for _, t := range enums.ValidTenants() {
		if string(t) == v {
			return true
		}
	}
	return false
}

func mapTenantNameToUUID(name string) uuid.UUID {
	switch strings.ToLower(name) {
	case "status_invest":
		return uuid.MustParse("00000000-0000-0000-0000-000000000001")
	case "nai":
		return uuid.MustParse("00000000-0000-0000-0000-000000000002")
	case "fiis":
		return uuid.MustParse("00000000-0000-0000-0000-000000000003")
	case "funds_explorer":
		return uuid.MustParse("00000000-0000-0000-0000-000000000004")
	case "orcana":
		return uuid.MustParse("00000000-0000-0000-0000-000000000005")
	default:
		return uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}
}

func mapUUIDToTenantName(id uuid.UUID) string {
	switch id.String() {
	case "00000000-0000-0000-0000-000000000001":
		return "status_invest"
	case "00000000-0000-0000-0000-000000000002":
		return "nai"
	case "00000000-0000-0000-0000-000000000003":
		return "fiis"
	case "00000000-0000-0000-0000-000000000004":
		return "funds_explorer"
	case "00000000-0000-0000-0000-000000000005":
		return "orcana"
	default:
		return "status_invest"
	}
}
