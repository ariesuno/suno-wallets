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

	b3RetriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_retries_total",
			Help: "Total de retries efetuados ao chamar B3",
		},
		[]string{"endpoint"},
	)

	// Métricas específicas do preview de transações
	b3TransactionsPagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_transactions_pages_processed_total",
			Help: "Total de páginas processadas no preview de transações",
		},
		[]string{"asset_type"},
	)
	b3TransactionsRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_transactions_preview_requests_total",
			Help: "Total de requisições de preview de transações",
		},
		[]string{"asset_type", "result"},
	)
)

// ObserveExternalAPI registra duração e contagem de chamadas externas
func ObserveExternalAPI(provider, endpoint, status string, start time.Time) {
	externalAPITotal.WithLabelValues(provider, endpoint, status).Inc()
	externalAPIDuration.WithLabelValues(provider, endpoint, status).Observe(time.Since(start).Seconds())
}

func IncB3Retries(endpoint string) {
	b3RetriesTotal.WithLabelValues(endpoint).Inc()
}

// ObserveTransactionsPreview registra métricas do preview
func ObserveTransactionsPreview(assetType, result string, pages int) {
	b3TransactionsRequestsTotal.WithLabelValues(assetType, result).Inc()
	if pages > 0 {
		b3TransactionsPagesTotal.WithLabelValues(assetType).Add(float64(pages))
	}
}
