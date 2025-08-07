package observability_test

import (
	"net/http/httptest"
	"testing"

	"suno-wallets/src/infrastructure/observability"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Testa exposição do endpoint /metrics
func TestMetricsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	observability.RegisterMetricsRoute(r)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "# HELP")
}
