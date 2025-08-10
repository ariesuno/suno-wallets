# Prompt 1.25 — Health & Observability Finale (Fase 1) — **suno-wallets**

Consolide a **observabilidade de ponta a ponta** da Fase 1: *health checks*, métricas Prometheus, logs estruturados (Loki), *traces* OpenTelemetry, dashboards Grafana e alertas. Inclua **endpoints padrões** (`/health/*`, `/metrics`) e **status operacional** voltado a B3, jobs e ledger.  
**Meta‑regras**: nomes em **inglês**, comentários em **pt‑BR**, **sem PII em métricas/labels**, CPF apenas **mascarado** em logs, **multi‑tenant estrito**.

---

## 🎯 Objetivos
1) Expor **health liveness/readiness/startup** e **status operacional** (B3, DB, cache, jobs).  
2) Publicar **métricas Prometheus** com naming consistente e baixa cardinalidade.  
3) Padronizar **logs JSON** (Loki) com correlação (trace/span/requestId) e mascaramento.  
4) Habilitar **OpenTelemetry** (HTTP + DB + jobs), amostragem e *exporters*.  
5) Entregar **dashboards Grafana** (JSON), **alertas** (YAML) e **documentação** de operação.  
6) Testes (unit/integration) cobrindo endpoints, métricas, logs e *alerts sanity*.

---

## 🧱 Arquitetura (DDD + Infra)
Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Api/Controllers/Observability/health_controller.go`  
  Endpoints `/health/live`, `/health/ready`, `/health/startup`, `/observability/status`.

- `src/Infrastructure/Observability/metrics_registry.go`  
  Registro único de métricas, *helpers* e middlewares HTTP.

- `src/Infrastructure/Observability/logging.go`  
  Logger JSON (campos padrão), mascaramento, *request context*, *trace correlation*.

- `src/Infrastructure/Observability/tracing.go`  
  Config OpenTelemetry (OTLP/HTTP ou gRPC), amostragem e instrumentação básica.

- `src/Infrastructure/Observability/rate_limit_probe.go`  
  Coleta *rate/quotas* da B3 quando disponíveis (headers/erros) e mede *backoff/retries*.

- `src/Infrastructure/Observability/status_aggregator.go`  
  Agrega saúde de DB, cache, B3 client, filas (se houver), *workers* e *last runs*.

Pastas de testes:  
- `src/Tests/Unit/Observability/...`  
- `src/Tests/Integration/Observability/...`

---

## 📡 Endpoints (Swagger tags: `Health`, `Observability`)

### 1) `GET /health/live`
- Retorna 200 se o processo está vivo (sem IO externo).

### 2) `GET /health/ready`
- Verifica **PostgreSQL**, cache (se usado), *migrations ok*, *seed mínima* e **B3 client** inicializável.  
- Retorna 200 apenas se o serviço puder atender **tráfego real**.

### 3) `GET /health/startup`
- Retorna 200 após *bootstrap* (migrations + cargas essenciais).

### 4) `GET /observability/status`
- Resumo operacional (sem PII):  
```json
{
  "version":"1.0.0+build",
  "uptimeSec": 12345,
  "tenantContext": "required",
  "db": {"status":"OK","pool":{"inUse":3,"idle":10,"waitCount":0}},
  "b3": {"client":"OK","lastQuota":{"remaining":123,"resetAt":"ISO"},"lastError":null},
  "jobs": {"b3FullFetch":{"lastRun":"ISO","durationSec":30,"result":"SUCCESS"},"b3Incremental":{"lastRun":"ISO"}},
  "ledger": {"opsTotal": 1234567, "lastOpAt":"ISO"},
  "cache": {"status":"OK","hitRatio":0.83},
  "traceSampleRate": 0.1
}
```

**Segurança**: `X-Tenant-Id` obrigatório em `/observability/status` (e qualquer rota que mostre dados de tenant).

---

## 📊 Métricas Prometheus (exemplos)

**HTTP (middleware)**  
- `http_requests_total{route,method,status}`  
- `http_request_duration_seconds{route,method}` (histogram)  
- `http_response_size_bytes{route,method}`

**B3 Client**  
- `b3_requests_total{endpoint,result}`  
- `b3_request_duration_seconds{endpoint}` (histogram)  
- `b3_backoff_retries_total{endpoint,reason}`  
- `b3_rate_limit_remaining{endpoint}` (gauge)  
- `b3_window_resets_at_timestamp{endpoint}` (gauge)

**Jobs**  
- `job_runs_total{job,result}`  
- `job_duration_seconds{job}` (histogram)  
- `policy_skipped_total{job}` *(1.21)*

**Ledger / Ops**  
- `ledger_ops_ingested_total{source}` *(B3_RAW|SYSTEM_SYNTHETIC|USER_MANUAL)*  
- `ledger_ops_active_gauge{source}`  
- `inconsistency_open_gauge{type}` *(1.17)*  
- `ca_projection_runs_total{status}` *(1.24)*

**DB/Cache**  
- `db_pool_in_use_gauge`, `db_pool_idle_gauge`, `db_pool_wait_count_total`  
- `cache_hits_total{name}`, `cache_misses_total{name}`

> ⚠️ **Nunca** use CPF/tickers específicos em labels. **Apenas agregados.**

---

## 🧾 Logs (Loki) — formato e campos
- Formato **JSON**; *logger* central. Campos obrigatórios:  
  `timestamp, level, message, tenantId, requestId, traceId, spanId, route, method, status, durationMs, actorRole, cpfMasked?`  
- **Mascaramento**: `cpfMasked="***1234"` (últimos 4). Nunca logar tokens/segredos.  
- **Contexto**: propagar `requestId` + `traceId/spanId` do OpenTelemetry.  
- **Erros**: incluir `error.kind`, `error.code`, `stack` quando disponível.  
- **Jobs**: logar início/fim, contagens e tempo.

---

## 🔭 Tracing (OpenTelemetry)
- *SDK* com **amostragem** (`OTEL_TRACES_SAMPLER=traceidratio`, `OTEL_TRACES_SAMPLER_ARG=0.1`).  
- Export via **OTLP** (HTTP/gRPC).  
- *Spans* relevantes: HTTP server, HTTP client (B3), DB (queries sensíveis), jobs.  
- Atribuir *attributes* com **baixa cardinalidade** (ex.: `b3.endpoint`, `job.name`).

---

## 📈 Grafana — dashboards (JSON no repo)
Adicione arquivos JSON em `deploy/grafana/dashboards/phase1/`:

1) **B3 Ingestion & Health**  
   - Painéis: taxa/sucesso por endpoint, latência p50/p95, *rate limit remaining*, retriable vs non‑retriable, erros 4xx/5xx.

2) **Jobs & Backfills**  
   - Execuções/duração, skips por policy, frações removidas (1.24), tempo por CPF top‑N (sem exibir CPF).

3) **API Performance**  
   - Requests por rota, latência p50/p95/p99, taxa de erro, throughput.

4) **Ledger & Recon Overview**  
   - `ledger_ops_active_gauge` por source, inconsistências abertas (1.17), system ops criadas (1.18).

> Inclua **templating** por `tenantId` e **links rápidos** para `/observability/status`.

---

## 🚨 Alertas (PrometheusRule YAML)
Adicione exemplos em `deploy/prometheus/rules/phase1/`:

```yaml
groups:
- name: phase1-b3.yml
  rules:
  - alert: B3ErrorSpike
    expr: increase(b3_requests_total{result="ERROR"}[5m]) > 50
    for: 10m
    labels: { severity: page }
    annotations:
      summary: "Erro B3 elevado"
      description: "Aumentaram as falhas na B3 em 10m. Verificar rate limit e credenciais."
  - alert: B3RateLimitLow
    expr: min_over_time(b3_rate_limit_remaining[15m]) < 100
    for: 5m
    labels: { severity: warn }
    annotations:
      summary: "Quota B3 baixa"
      description: "Rate limit próximo do fim. Considerar reduzir paralelismo."
- name: phase1-api.yml
  rules:
  - alert: ApiHighLatencyP95
    expr: histogram_quantile(0.95, sum by (le,route,method) (rate(http_request_duration_seconds_bucket[5m]))) > 1.5
    for: 10m
    labels: { severity: page }
    annotations:
      summary: "P95 alto"
      description: "Rotas com p95 > 1.5s por 10m."
```

---

## 🔧 Configurações (.env)
- `OBS_ENABLE_METRICS=true`
- `OBS_ENABLE_LOGS=true`
- `OBS_ENABLE_TRACING=true`
- `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318`
- `OTEL_TRACES_SAMPLER=traceidratio`
- `OTEL_TRACES_SAMPLER_ARG=0.1`
- `OBS_MASK_CPF=true`
- `OBS_TENANT_HEADER=X-Tenant-Id`

---

## 🧪 Testes (obrigatório)

**Integration — `src/Tests/Integration/Observability/health_and_metrics_test.go`**
- `/health/live` 200; `/health/ready` 200 só com DB/cache/B3 ok; `/health/startup` após bootstrap.  
- `/observability/status` exige `X-Tenant-Id` e **não** vaza PII.  
- `/metrics` expõe *families* esperadas; **nenhum CPF** no *payload*.  
- Métricas B3 incrementam em chamadas reais/fakes controladas (sem dados sensíveis).

**Unit**  
- Formatação de logs com mascaramento; *middleware* HTTP mede duração/bytes; *rate_limit_probe* parseia headers.  
- *Tracing*: propagação de `traceId`/`spanId` via *context*.

---

## 📄 Documentação
- **Swagger** (tags `Health`, `Observability`) com exemplos.  
- **Postman/Bruno**: rotas health/metrics/status.  
- `docs/observability/runbook_phase1.md`: como ler dashboards, responder a alertas (passo a passo), e checagens rápidas (checklist).

---

## ✅ Entregáveis
- Controllers/Infra de health/metrics/logs/tracing + dashboards + alertas.  
- Testes unitários e de integração.  
- Documentação operacional e coleções.  
- **Aceite**: em ambiente local, dashboards mostram métricas em tempo real; `/health/*` e `/observability/status` funcionam; alertas podem ser simulados com *load* e regras disparam conforme esperado.
