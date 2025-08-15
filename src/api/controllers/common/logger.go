package common

import (
	"time"

	"suno-wallets/src/shared/helpers"

	"github.com/gin-gonic/gin"
)

// LogLevel representa níveis de log
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelDebug LogLevel = "debug"
)

// ControllerLogger fornece logging estruturado para controllers
type ControllerLogger struct {
	context *gin.Context
}

// NewControllerLogger cria um novo logger para controller
func NewControllerLogger(c *gin.Context) *ControllerLogger {
	return &ControllerLogger{context: c}
}

// getBaseContext retorna contexto base para logs
func (cl *ControllerLogger) getBaseContext() map[string]interface{} {
	return map[string]interface{}{
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"path":           cl.context.Request.URL.Path,
		"method":         cl.context.Request.Method,
		"remote_addr":    cl.context.ClientIP(),
		"user_agent":     cl.context.GetHeader("User-Agent"),
		"tenant_id":      cl.context.GetString("tenant_id"),
		"user_id":        cl.context.GetString("user_id"),
		"request_id":     cl.context.GetString("request_id"),
		"correlation_id": cl.context.GetHeader("X-Correlation-ID"),
	}
}

// Info logs information level message
func (cl *ControllerLogger) Info(message string, additionalFields ...map[string]interface{}) {
	context := cl.getBaseContext()
	context["level"] = LogLevelInfo

	for _, fields := range additionalFields {
		for k, v := range fields {
			context[k] = v
		}
	}

	helpers.LogInfo(message, context)
}

// Warn logs warning level message
func (cl *ControllerLogger) Warn(message string, additionalFields ...map[string]interface{}) {
	context := cl.getBaseContext()
	context["level"] = LogLevelWarn

	for _, fields := range additionalFields {
		for k, v := range fields {
			context[k] = v
		}
	}

	helpers.LogWarn(message, context)
}

// Error logs error level message
func (cl *ControllerLogger) Error(message string, err error, additionalFields ...map[string]interface{}) {
	context := cl.getBaseContext()
	context["level"] = LogLevelError
	if err != nil {
		context["error"] = err.Error()
	}

	for _, fields := range additionalFields {
		for k, v := range fields {
			context[k] = v
		}
	}

	helpers.LogError(message, err, context)
}

// Debug logs debug level message
func (cl *ControllerLogger) Debug(message string, additionalFields ...map[string]interface{}) {
	context := cl.getBaseContext()
	context["level"] = LogLevelDebug

	for _, fields := range additionalFields {
		for k, v := range fields {
			context[k] = v
		}
	}

	helpers.LogInfo("[DEBUG] "+message, context)
}

// LogRequest logs incoming request details
func (cl *ControllerLogger) LogRequest(action string, additionalFields ...map[string]interface{}) {
	context := cl.getBaseContext()
	context["action"] = action
	context["content_type"] = cl.context.GetHeader("Content-Type")
	context["content_length"] = cl.context.GetHeader("Content-Length")

	for _, fields := range additionalFields {
		for k, v := range fields {
			context[k] = v
		}
	}

	helpers.LogInfo("Request received", context)
}

// LogResponse logs response details
func (cl *ControllerLogger) LogResponse(statusCode int, duration time.Duration, additionalFields ...map[string]interface{}) {
	context := cl.getBaseContext()
	context["status_code"] = statusCode
	context["duration_ms"] = duration.Milliseconds()
	context["duration_human"] = duration.String()

	for _, fields := range additionalFields {
		for k, v := range fields {
			context[k] = v
		}
	}

	message := "Request completed"
	if statusCode >= 400 {
		helpers.LogWarn(message, context)
	} else {
		helpers.LogInfo(message, context)
	}
}

// LogBusinessEvent logs business-specific events
func (cl *ControllerLogger) LogBusinessEvent(event string, entityType string, entityID string, additionalFields ...map[string]interface{}) {
	context := cl.getBaseContext()
	context["event"] = event
	context["entity_type"] = entityType
	context["entity_id"] = entityID

	for _, fields := range additionalFields {
		for k, v := range fields {
			context[k] = v
		}
	}

	helpers.LogInfo("Business event", context)
}

// LogPerformance logs performance metrics
func (cl *ControllerLogger) LogPerformance(operation string, duration time.Duration, additionalFields ...map[string]interface{}) {
	context := cl.getBaseContext()
	context["operation"] = operation
	context["duration_ms"] = duration.Milliseconds()
	context["duration_human"] = duration.String()

	for _, fields := range additionalFields {
		for k, v := range fields {
			context[k] = v
		}
	}

	// Log como warning se duração for muito alta
	if duration > 5*time.Second {
		helpers.LogWarn("Slow operation detected", context)
	} else {
		helpers.LogInfo("Performance metric", context)
	}
}

// LogSecurityEvent logs security-related events
func (cl *ControllerLogger) LogSecurityEvent(event string, severity string, additionalFields ...map[string]interface{}) {
	context := cl.getBaseContext()
	context["security_event"] = event
	context["severity"] = severity
	context["remote_addr"] = cl.context.ClientIP()
	context["user_agent"] = cl.context.GetHeader("User-Agent")

	for _, fields := range additionalFields {
		for k, v := range fields {
			context[k] = v
		}
	}

	if severity == "high" || severity == "critical" {
		helpers.LogError("Security event", nil, context)
	} else {
		helpers.LogWarn("Security event", context)
	}
}
