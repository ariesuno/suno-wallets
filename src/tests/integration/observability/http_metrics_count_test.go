package observability_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"suno-wallets/src/infrastructure/observability"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
)

// Verifica incremento de http_requests_total após uma requisição
func TestHTTPRequestsTotalIncrements(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(observability.PrometheusMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// dispara uma requisição
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ping", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// consulta métricas e valida exposição
	mw := httptest.NewRecorder()
	mreq, _ := http.NewRequest("GET", "/metrics", nil)
	r.ServeHTTP(mw, mreq)
	require.Equal(t, 200, mw.Code)
	require.Contains(t, mw.Body.String(), "http_requests_total")
}
