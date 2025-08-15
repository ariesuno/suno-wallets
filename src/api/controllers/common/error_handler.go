package common

import (
	"net/http"

	"suno-wallets/src/domain/entities"
	"suno-wallets/src/shared/helpers"

	"github.com/gin-gonic/gin"
)

// ErrorResponse representa uma resposta de erro padronizada
type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Code    string      `json:"code,omitempty"`
}

// HandleError trata erros de forma padronizada em todos os controllers
func HandleError(c *gin.Context, err error, userMessage string, additionalContext ...map[string]interface{}) {
	// Combinar contextos adicionais
	logContext := map[string]interface{}{
		"path":   c.Request.URL.Path,
		"method": c.Request.Method,
	}

	for _, ctx := range additionalContext {
		for k, v := range ctx {
			logContext[k] = v
		}
	}

	// Log do erro
	helpers.LogError(userMessage, err, logContext)

	// Determinar resposta baseada no tipo de erro
	if entities.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "VALIDATION_ERROR",
			Message: err.Error(),
			Code:    "400001",
		})
		return
	}

	if entities.IsNotFoundError(err) {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "NOT_FOUND",
			Message: err.Error(),
			Code:    "404001",
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
		case "RATE_LIMIT_EXCEEDED":
			statusCode = http.StatusTooManyRequests
		}

		c.JSON(statusCode, ErrorResponse{
			Error:   businessErr.Code,
			Message: businessErr.Message,
			Code:    businessErr.Code,
		})
		return
	}

	if entities.IsConflictError(err) {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "CONFLICT",
			Message: err.Error(),
			Code:    "409001",
		})
		return
	}

	// Erro genérico interno
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   "INTERNAL_ERROR",
		Message: "Erro interno do servidor",
		Code:    "500001",
	})
}

// HandleValidationError trata especificamente erros de validação de entrada
func HandleValidationError(c *gin.Context, err error, message string) {
	helpers.LogWarn(message, map[string]interface{}{
		"path":   c.Request.URL.Path,
		"method": c.Request.Method,
		"error":  err.Error(),
	})

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   "INVALID_REQUEST",
		Message: message,
		Details: err.Error(),
		Code:    "400002",
	})
}

// HandleUnauthorized trata erros de autorização
func HandleUnauthorized(c *gin.Context, message string) {
	helpers.LogWarn("Unauthorized access attempt", map[string]interface{}{
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
		"message": message,
	})

	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Error:   "UNAUTHORIZED",
		Message: message,
		Code:    "401001",
	})
}

// HandleForbidden trata erros de permissão
func HandleForbidden(c *gin.Context, message string) {
	helpers.LogWarn("Forbidden access attempt", map[string]interface{}{
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
		"message": message,
	})

	c.JSON(http.StatusForbidden, ErrorResponse{
		Error:   "FORBIDDEN",
		Message: message,
		Code:    "403001",
	})
}

// HandleBadRequest trata requisições malformadas
func HandleBadRequest(c *gin.Context, message string, details interface{}) {
	helpers.LogWarn("Bad request", map[string]interface{}{
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
		"message": message,
		"details": details,
	})

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   "BAD_REQUEST",
		Message: message,
		Details: details,
		Code:    "400003",
	})
}

// HandleServiceUnavailable trata erros de serviço indisponível
func HandleServiceUnavailable(c *gin.Context, message string) {
	helpers.LogError("Service unavailable", nil, map[string]interface{}{
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
		"message": message,
	})

	c.JSON(http.StatusServiceUnavailable, ErrorResponse{
		Error:   "SERVICE_UNAVAILABLE",
		Message: message,
		Code:    "503001",
	})
}
