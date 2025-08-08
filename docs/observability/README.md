# Observabilidade

Este diretório contém instruções para executar Prometheus, Grafana, Loki e Tempo localmente.

## Endpoints
- Métricas: `http://localhost:8080/metrics`
- Health: `http://localhost:8080/health`

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
