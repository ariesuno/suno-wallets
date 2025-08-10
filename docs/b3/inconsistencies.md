# Inconsistencies (1.17)

- Endpoints:
  - POST `/api/v1/b3/reconciliation/scan` (admin: `X-Admin-Secret`) — body com `cpf`, `tickers?`, `from?`, `to?`, `dryRun?`, `maxSamplesPerType?`, `concurrency?`
  - GET `/api/v1/b3/reconciliation/inconsistencies` — filtros `cpf`, `status?`, `type?`, `ticker?`
  - GET `/api/v1/b3/reconciliation/inconsistencies/{id}`

- Métricas Prometheus:
  - `b3_recon_scan_runs_total{result}`
  - `b3_recon_inconsistencies_found_total{type}`
  - `b3_recon_scan_duration_seconds_bucket`

- Observações: somente leitura de RAW/Normalized; grava achados em `b3_inconsistencies` com upsert por `dedupe_hash`.


