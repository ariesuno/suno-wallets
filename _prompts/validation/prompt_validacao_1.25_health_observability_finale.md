# Prompt de Validação 1.25v — Health & Observability Finale (Fase 1) — **suno-wallets**

## 🎯 Objetivo
Validar que a **observabilidade de ponta a ponta** foi consolidada: endpoints de **health** (`/health/*`), **status operacional**, **métricas Prometheus**, **logs estruturados (Loki)**, **tracing OpenTelemetry**, **dashboards Grafana** e **alertas Prometheus**, tudo **multi‑tenant**, com **LGPD**, **baixa cardinalidade** e **testes automatizados**.

> Escopo: controllers/infra de health, metrics, logging, tracing; status aggregator; rate‑limit probe B3; dashboards JSON; regras de alertas; Swagger/Collections; testes unitários e de integração.


---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos criados com **nomes em inglês** e **comentários pt‑BR**:
  - `src/Api/Controllers/Observability/health_controller.go`
  - `src/Infrastructure/Observability/metrics_registry.go`
  - `src/Infrastructure/Observability/logging.go`
  - `src/Infrastructure/Observability/tracing.go`
  - `src/Infrastructure/Observability/rate_limit_probe.go`
  - `src/Infrastructure/Observability/status_aggregator.go`
  - Testes: `src/Tests/Unit/Observability/...`, `src/Tests/Integration/Observability/...`
- [ ] Controllers **não** contêm lógica de métrica/trace — usam middlewares/infra.

### 2) Endpoints
- [ ] `GET /health/live` retorna 200 sem dependências externas.
- [ ] `GET /health/ready` verifica DB, cache (se houver), migrações, e **B3 client inicializável**; só retorna 200 quando **apto** a tráfego.
- [ ] `GET /health/startup` retorna 200 após bootstrap.
- [ ] `GET /observability/status` agregado operacional sem PII; requer **`X-Tenant-Id`**.
- [ ] Swagger contém as quatro rotas nas tags `Health` e `Observability` com exemplos sanitizados.
- [ ] Coleções Postman/Bruno exportadas em `/docs/observability/`.

### 3) Métricas Prometheus
- [ ] **HTTP**: `http_requests_total{route,method,status}`, `http_request_duration_seconds{route,method}` (histogram), `http_response_size_bytes{route,method}`.
- [ ] **B3 client**: `b3_requests_total{endpoint,result}`, `b3_request_duration_seconds{endpoint}`, `b3_backoff_retries_total{endpoint,reason}`, `b3_rate_limit_remaining{endpoint}` (gauge), `b3_window_resets_at_timestamp{endpoint}` (gauge).
- [ ] **Jobs**: `job_runs_total{job,result}`, `job_duration_seconds{job}`, `policy_skipped_total{job}` (1.21).
- [ ] **Ledger & Recon**: `ledger_ops_ingested_total{source}`, `ledger_ops_active_gauge{source}`, `inconsistency_open_gauge{type}` (1.17), `ca_projection_runs_total{status}` (1.24).
- [ ] **DB/Cache**: `db_pool_in_use_gauge`, `db_pool_idle_gauge`, `db_pool_wait_count_total`, `cache_hits_total{name}`, `cache_misses_total{name}`.
- [ ] **Baixa cardinalidade** (sem CPF/ticker em labels); validação de *help/type/unit* nos *descriptors*.
- [ ] `/metrics` expõe os *families* acima.

### 4) Logs (Loki)
- [ ] Formato **JSON** unificado: `timestamp, level, message, tenantId, requestId, traceId, spanId, route, method, status, durationMs, actorRole, cpfMasked?`.
- [ ] **Mascaramento** de CPF (`***1234`) aplicado; **nenhum** token/segredo em logs.
- [ ] Middlewares adicionam `requestId` e **correlação com trace** (`traceId`, `spanId`).
- [ ] Logs de **jobs** incluem contagens e duração; erros contêm `error.kind`, `error.code`, `stack`.

### 5) Tracing (OpenTelemetry)
- [ ] SDK configurado com **amostragem** (ex.: `traceidratio 0.1`).
- [ ] Export via **OTLP** (HTTP/gRPC) funcional.
- [ ] *Spans* para: HTTP server, HTTP client (B3), DB (queries relevantes), jobs.
- [ ] Atributos de **baixa cardinalidade** (ex.: `b3.endpoint`, `job.name`) e propagação de contexto OK.

### 6) Dashboards Grafana
- [ ] JSONs em `deploy/grafana/dashboards/phase1/` para:
  - **B3 Ingestion & Health** (taxa/sucesso, latências, rate limit, erros 4xx/5xx, retries/backoff).
  - **Jobs & Backfills** (execuções/duração, skips por policy, frações removidas 1.24, top‑N por consumo de tempo sem expor CPF).
  - **API Performance** (requests por rota, p50/p95/p99, taxa de erro, throughput).
  - **Ledger & Recon** (ops ativas por source, inconsistências abertas 1.17, system ops 1.18).
- [ ] Dashboards com **templating por `tenantId`** e atalhos para `/observability/status`.

### 7) Alertas Prometheus
- [ ] YAMLs em `deploy/prometheus/rules/phase1/` com regras exemplo:
  - `B3ErrorSpike`, `B3RateLimitLow`
  - `ApiHighLatencyP95`
- [ ] Validação de sintaxe (*promtool*), severidade (`page`/`warn`) e *for* adequados.
- [ ] Alertas **não** incluem PII e usam labels/annotations claras.

### 8) Segurança, LGPD & Tenant
- [ ] `X-Tenant-Id` **obrigatório** em `/observability/status` e qualquer rota que exponha dados do tenant.
- [ ] Logs e métricas **sem PII**; CPF só **mascarado** nos logs (nunca em métricas).
- [ ] RBAC aplicado para leitura de status detalhado (admin/suporte).

### 9) Performance & Confiabilidade
- [ ] Overhead do middleware de métricas/logs **baixo** (p95 < 5ms por request em ambiente local).
- [ ] *Sampling* de tracing configurado para não degradar throughput.
- [ ] `/health/ready` **não** realiza operações pesadas (no máximo *ping* de dependências).

### 10) Testes Automatizados
**Integration — `src/Tests/Integration/Observability/health_and_metrics_test.go`**
- [ ] `/health/live` 200 sempre que o processo estiver vivo.
- [ ] `/health/ready` 503 quando DB/cache/B3 indisponíveis; 200 quando OK.
- [ ] `/health/startup` 200 após bootstrap.
- [ ] `/observability/status` exige `X-Tenant-Id`, retorna sem PII e com campos coerentes.
- [ ] `/metrics` contém *families* esperadas (amostra).
- [ ] B3 client: chamadas simuladas elevam corretamente `b3_requests_total`, `b3_request_duration_seconds` e `b3_backoff_retries_total`.

**Unit**
- [ ] Logger mascara CPF e agrega `traceId`/`spanId`; *rate_limit_probe* parseia headers; *middlewares* HTTP registram duração/bytes.

### 11) Documentação
- [ ] Swagger atualizado (tags `Health`, `Observability`) e coleções **Postman/Bruno**.
- [ ] `docs/observability/runbook_phase1.md` explica leitura dos dashboards, ações em alertas e *troubleshooting*.

---

## 🧪 Passos de Validação Manual (rápidos)

1) **Health**
```bash
curl "$BASE_URL/health/live"
curl "$BASE_URL/health/ready"
curl "$BASE_URL/health/startup"
```

2) **Status Operacional**
```bash
curl "$BASE_URL/observability/status" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

3) **Métricas**
```bash
curl "$BASE_URL/metrics"
```

4) **Tracing**
- Ative uma requisição conhecida (ex.: `/observability/status`); verifique no Grafana Tempo/Jaeger traces com `route=/observability/status` e spans HTTP→DB→B3.

5) **Alertas**
- Em dev, force erro B3 (credencial inválida ou endpoint simulado) e verifique se **B3ErrorSpike** dispara no Prometheus/Grafana.

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.25 (Health & Observability Finale) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
