package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	externalAPIDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "external_api_duration_seconds",
			Help:    "Duração de chamadas a APIs externas",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider", "endpoint", "status"},
	)

	externalAPITotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "external_api_requests_total",
			Help: "Total de chamadas a APIs externas",
		},
		[]string{"provider", "endpoint", "status"},
	)
)

// ObserveExternalAPI registra duração e contagem de chamadas externas
func ObserveExternalAPI(provider, endpoint, status string, start time.Time) {
	externalAPITotal.WithLabelValues(provider, endpoint, status).Inc()
	externalAPIDuration.WithLabelValues(provider, endpoint, status).Observe(time.Since(start).Seconds())
}
