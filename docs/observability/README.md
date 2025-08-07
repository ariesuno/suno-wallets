# Observabilidade

Este diretório contém instruções para executar Prometheus, Grafana, Loki e Tempo localmente.

## Endpoints
- Métricas: `http://localhost:8080/metrics`
- Health: `http://localhost:8080/health`

## Executando stack (docker-compose)
Adicionaremos serviços no `docker-compose.yml`:
- Prometheus (aponta para `app:8080/metrics`)
- Grafana (dashboards)
- Loki e Promtail (coleta de logs da app)
- Tempo (rastreamento; opcional)

## Testes
Rode:
```
go test ./tests/observability/... -v
```
