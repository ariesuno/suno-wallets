package observability

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total de requisições HTTP por status, método e rota",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duração das requisições HTTP em segundos",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)
)

// PrometheusMiddleware coleta métricas de requisições HTTP
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		// route é o path template quando disponível
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		duration := time.Since(start).Seconds()
		httpRequestsTotal.WithLabelValues(method, route, status).Inc()
		httpRequestDurationSeconds.WithLabelValues(method, route, status).Observe(duration)
	}
}

// ExposeHTTPMetrics retorna referências às métricas (útil para testes/inspeção)
func ExposeHTTPMetrics() (*prometheus.CounterVec, *prometheus.HistogramVec) {
	return httpRequestsTotal, httpRequestDurationSeconds
}
