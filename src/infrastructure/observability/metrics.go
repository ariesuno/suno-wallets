package observability

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// build_info{version,commit,buildDate} = 1
var buildInfoGauge = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "build_info",
		Help: "Informações de build (gauge com valor fixo 1)",
	},
	[]string{"version", "commit", "buildDate"},
)

// RegisterBuildInfo seta a métrica build_info para 1 com labels do build.
func RegisterBuildInfo(version, commit, buildDate string) {
	buildInfoGauge.WithLabelValues(version, commit, buildDate).Set(1)
}

// RegisterMetricsRoute registra endpoint /metrics (Prometheus)
func RegisterMetricsRoute(router *gin.Engine) {
	router.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
}
