# Observabilidade

Este diretório contém instruções para executar Prometheus, Grafana, Loki e Tempo localmente.

## Endpoints
- Métricas: `http://localhost:8080/metrics`
- Health (legado): `http://localhost:8080/health`
- Liveness: `http://localhost:8080/health/live`
- Readiness: `http://localhost:8080/health/ready`
- Details: `http://localhost:8080/health/details`
- B3 Auth Health (flag): `http://localhost:8080/api/v1/b3/health/auth` (`OBS_ALLOW_B3_AUTH_HEALTH=true`)

## Stack (docker-compose)
Serviços no `docker-compose.yml`:
- Prometheus (config em `misc/prometheus.yml`, alertas em `misc/alert.rules.yml`)
- Grafana (acesso: `http://localhost:3000`)
- Loki + Promtail (coleta de logs em `./logs` -> Loki)
- Node Exporter e cAdvisor (métricas de host/containers)
  
Prometheus scrapa `app:8080/metrics`.

## Testes
```
go test ./tests/integration/observability/... -v
```

## Dashboards

- Importar `docs/observability/grafana_b3_reports_dashboard.json` para acompanhar os endpoints de Reports (1.15): throughput, erros e latência (p95/p99) por `endpoint`.
- Importar `docs/observability/grafana_phase1_smoke_dashboard.json` para visão geral de HTTP + build info.
- Importar `docs/observability/grafana_e2e_incremental_dashboard.json` para E2E (1.13) e Incremental (1.14): runs por `result/mode/dataType/assetType`, erros por `stage`, latência p95/p99, e `b3_retries_total` por endpoint.
