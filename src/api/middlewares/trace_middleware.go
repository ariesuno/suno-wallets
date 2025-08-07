package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceMiddleware adiciona trace_id/span_id simples no contexto e resposta
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}
		spanID := uuid.NewString()

		c.Set("trace_id", traceID)
		c.Set("span_id", spanID)
		c.Writer.Header().Set("X-Trace-ID", traceID)
		c.Writer.Header().Set("X-Span-ID", spanID)

		c.Next()
	}
}
