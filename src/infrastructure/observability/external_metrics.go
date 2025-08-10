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

	// Métricas do orquestrador E2E (1.13)
	b3E2ERunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_e2e_runs_total",
			Help: "Total de execuções do orquestrador E2E por resultado, modo, dataType e assetType",
		},
		[]string{"result", "mode", "dataType", "assetType"},
	)
	b3E2EDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "b3_e2e_duration_seconds",
			Help:    "Duração de execuções E2E",
			Buckets: prometheus.DefBuckets,
		},
	)
	b3E2EErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_e2e_errors_total",
			Help: "Total de erros no E2E por etapa",
		},
		[]string{"stage"},
	)

	// Métricas do Incremental Runner (1.14)
	b3IncrementalRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_incremental_runs_total",
			Help: "Total de execuções do incremental por resultado e tipo",
		},
		[]string{"result", "type"},
	)
	b3IncrementalDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "b3_incremental_duration_seconds",
			Help:    "Duração do incremental",
			Buckets: prometheus.DefBuckets,
		},
	)
	b3IncrementalErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_incremental_errors_total",
			Help: "Erros no incremental por stage",
		},
		[]string{"stage"},
	)

	// Métricas dos relatórios (1.15)
	b3ReportsRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_reports_requests_total",
			Help: "Total de requisições de relatórios por endpoint",
		},
		[]string{"endpoint"},
	)
	b3ReportsErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_reports_errors_total",
			Help: "Total de erros nos relatórios por endpoint",
		},
		[]string{"endpoint"},
	)
	b3ReportsDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "b3_reports_duration_seconds",
			Help:    "Duração das requisições de relatórios",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// Métricas do detector de inconsistências (1.17)
	b3ReconScanRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_recon_scan_runs_total",
			Help: "Total de execuções do scan de inconsistências por resultado",
		},
		[]string{"result"},
	)
	b3ReconInconsistenciesFoundTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "b3_recon_inconsistencies_found_total",
			Help: "Total de inconsistências encontradas por tipo",
		},
		[]string{"type"},
	)
	b3ReconScanDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "b3_recon_scan_duration_seconds",
			Help:    "Duração das execuções do scan de inconsistências",
			Buckets: prometheus.DefBuckets,
		},
	)

  // Métricas do Auto-fix (1.18)
  b3AutofixRunsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
      Name: "b3_autofix_runs_total",
      Help: "Total de execuções do auto-fix por resultado",
    },
    []string{"result"},
  )
  b3AutofixOpsCreatedTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
      Name: "b3_autofix_ops_created_total",
      Help: "Total de operações criadas pelo auto-fix por reason_code",
    },
    []string{"reason_code"},
  )
  b3AutofixPendingTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
      Name: "b3_autofix_pending_total",
      Help: "Total de inconsistências pendentes no auto-fix por reason",
    },
    []string{"reason"},
  )
  b3AutofixDuration = promauto.NewHistogram(
    prometheus.HistogramOpts{
      Name:    "b3_autofix_duration_seconds",
      Help:    "Duração das execuções do auto-fix",
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
	if clients > 0 {
		b3SyncClientsTotal.Add(float64(clients))
	}
	if success > 0 {
		b3SyncSuccessTotal.Add(float64(success))
	}
	if failed > 0 {
		b3SyncFailedTotal.Add(float64(failed))
	}
	if newRaw > 0 {
		b3SyncNewRawTotal.Add(float64(newRaw))
	}
	b3SyncDuration.Observe(time.Since(startedAt).Seconds())
}

// ObserveE2ERun registra agregados de execução do E2E
func ObserveE2ERun(result, mode, dataType, assetType string, startedAt time.Time) {
	b3E2ERunsTotal.WithLabelValues(result, mode, dataType, assetType).Inc()
	b3E2EDuration.Observe(time.Since(startedAt).Seconds())
}

// IncE2EError incrementa contador de erros por etapa do E2E
func IncE2EError(stage string) { b3E2EErrorsTotal.WithLabelValues(stage).Inc() }

// Incremental metrics
func ObserveIncrementalRun(result, typ string, startedAt time.Time) {
	b3IncrementalRunsTotal.WithLabelValues(result, typ).Inc()
	b3IncrementalDuration.Observe(time.Since(startedAt).Seconds())
}
func IncIncrementalError(stage string) { b3IncrementalErrorsTotal.WithLabelValues(stage).Inc() }

// Reports metrics
func ObserveReport(endpoint string, startedAt time.Time) {
	b3ReportsRequestsTotal.WithLabelValues(endpoint).Inc()
	b3ReportsDuration.WithLabelValues(endpoint).Observe(time.Since(startedAt).Seconds())
}
func IncReportError(endpoint string) { b3ReportsErrorsTotal.WithLabelValues(endpoint).Inc() }

// Recon metrics
func ObserveReconScanRun(result string, startedAt time.Time) {
	b3ReconScanRunsTotal.WithLabelValues(result).Inc()
	b3ReconScanDuration.Observe(time.Since(startedAt).Seconds())
}
func IncReconFound(typ string, n int) {
	if n <= 0 {
		return
	}
	b3ReconInconsistenciesFoundTotal.WithLabelValues(typ).Add(float64(n))
}

// Auto-fix metrics
func ObserveAutofixRun(result string, startedAt time.Time) {
  b3AutofixRunsTotal.WithLabelValues(result).Inc()
  b3AutofixDuration.Observe(time.Since(startedAt).Seconds())
}
func IncAutofixOpsCreated(reasonCode string, n int) {
  if n <= 0 { return }
  b3AutofixOpsCreatedTotal.WithLabelValues(reasonCode).Add(float64(n))
}
func IncAutofixPending(reason string, n int) {
  if n <= 0 { return }
  b3AutofixPendingTotal.WithLabelValues(reason).Add(float64(n))
}
