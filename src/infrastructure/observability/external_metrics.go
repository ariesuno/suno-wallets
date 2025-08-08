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

	// Métricas específicas do preview de posições
	b3PositionsPagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_positions_pages_processed_total",
			Help: "Total de páginas processadas no preview de posições",
		},
		[]string{"asset_type"},
	)
	b3PositionsRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_positions_preview_requests_total",
			Help: "Total de requisições de preview de posições",
		},
		[]string{"asset_type", "result"},
	)

	// Métricas do módulo de persistência RAW (1.9)
	b3RawSavedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "b3_raw_saved_total",
			Help: "Total de registros RAW salvos",
		},
	)
	b3RawSkippedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "b3_raw_skipped_total",
			Help: "Total de registros/mês pulados por cobertura",
		},
	)
	b3RawErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "b3_raw_errors_total",
			Help: "Total de erros durante ingestão RAW",
		},
	)
	b3RawPagesProcessedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "b3_raw_pages_processed_total",
			Help: "Total de páginas processadas na ingestão RAW",
		},
	)
	b3RawMonthsCompletedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "b3_raw_months_completed_total",
			Help: "Total de meses concluídos na ingestão RAW",
		},
	)

  // Métricas do Sync diário (1.11)
  b3SyncClientsTotal = promauto.NewCounter(
    prometheus.CounterOpts{
      Name: "b3_sync_clients_total",
      Help: "Total de clientes avaliados no sync",
    },
  )
  b3SyncSuccessTotal = promauto.NewCounter(
    prometheus.CounterOpts{
      Name: "b3_sync_success_total",
      Help: "Total de clientes sincronizados com sucesso",
    },
  )
  b3SyncFailedTotal = promauto.NewCounter(
    prometheus.CounterOpts{
      Name: "b3_sync_failed_total",
      Help: "Total de clientes com falha no sync",
    },
  )
  b3SyncNewRawTotal = promauto.NewCounter(
    prometheus.CounterOpts{
      Name: "b3_sync_new_raw_total",
      Help: "Total de novos registros RAW detectados no sync",
    },
  )
  b3SyncDuration = promauto.NewHistogram(
    prometheus.HistogramOpts{
      Name:    "b3_sync_duration_seconds",
      Help:    "Duração do sync por execução (segundos)",
      Buckets: prometheus.DefBuckets,
    },
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

// ObservePositionsPreview registra métricas do preview de posições
func ObservePositionsPreview(assetType, result string, pages int) {
	b3PositionsRequestsTotal.WithLabelValues(assetType, result).Inc()
	if pages > 0 {
		b3PositionsPagesTotal.WithLabelValues(assetType).Add(float64(pages))
	}
}

// Funções para o módulo RAW
func IncRawSaved(n int) {
	if n > 0 {
		b3RawSavedTotal.Add(float64(n))
	}
}
func IncRawSkipped(n int) {
	if n > 0 {
		b3RawSkippedTotal.Add(float64(n))
	}
}
func IncRawErrors(n int) {
	if n > 0 {
		b3RawErrorsTotal.Add(float64(n))
	}
}
func IncRawPagesProcessed(n int) {
	if n > 0 {
		b3RawPagesProcessedTotal.Add(float64(n))
	}
}
func IncRawMonthsCompleted(n int) {
	if n > 0 {
		b3RawMonthsCompletedTotal.Add(float64(n))
	}
}

// Funções do Sync diário
func ObserveSync(clients, success, failed, newRaw int, startedAt time.Time) {
  if clients > 0 { b3SyncClientsTotal.Add(float64(clients)) }
  if success > 0 { b3SyncSuccessTotal.Add(float64(success)) }
  if failed > 0 { b3SyncFailedTotal.Add(float64(failed)) }
  if newRaw > 0 { b3SyncNewRawTotal.Add(float64(newRaw)) }
  b3SyncDuration.Observe(time.Since(startedAt).Seconds())
}
