# Prompt de Validação 1.29v — Smoke Reconciliação & Corporate Actions (E2E) — **suno-wallets**

## 🎯 Objetivo
Validar a **suite E2E Smoke** ponta‑a‑ponta da Fase 1: ingestão B3 (full/incremental) → inconsistências → auto‑fix (system ops) → override manual → projeção de CA com *comicota* (floor) → timeline/export → observabilidade.  
Checar **idempotência**, **gates de performance/erros**, **multi‑tenant**, **LGPD** (CPF mascarado), **métricas/traços/logs** e geração de **artifacts** de sumário.

> Nomes de arquivos/funções em **inglês**; comentários em **pt‑BR**. **Sem PII** em métricas/labels. CPF apenas **mascarado** nos logs e no sumário.


---

## ✅ Checklist de Validação

### 1) Estrutura & Artefatos
- [ ] Teste E2E criado em `src/Tests/E2E/smoke/smoke_e2e_test.go` (testify) com **comentários pt‑BR**.
- [ ] Script `scripts/e2e_smoke.sh` presente e executável.
- [ ] `docs/e2e/smoke_playbook.md` com setup, variáveis e troubleshooting.
- [ ] Coleções: `docs/postman/e2e_smoke_collection.json`, `docs/bruno/e2e_smoke.bru`.
- [ ] Job de CI: `deploy/ci/e2e-smoke.yml` executa a suite e **publica artifacts**.

### 2) Variáveis & Gates (.env)
- [ ] `E2E_TEST_CPF`, `E2E_TEST_TENANT_ID`, `E2E_TEST_FROM`, `E2E_TEST_TO`, `BASE_URL`, `AUTH_TOKEN` definidos.
- [ ] `OBS_REQUIRE_MIN_SAMPLE_RATE` configurado (ex.: `0.05`).
- [ ] **Gates** carregados: `GATE_MAX_P95_API`, `GATE_MAX_B3_ERROR_5M`, `GATE_MAX_INCONSISTENCIES_AFTER_AUTOFIX`.

### 3) Preflight
- [ ] `GET /health/ready` → **200**.
- [ ] `GET /observability/status` → DB/B3 client/jobs/cache/ledger OK (sem expor PII).
- [ ] `X-Tenant-Id` obrigatório em chamadas de status quando exigido.

### 4) Reset Seguro (opcional)
- [ ] Endpoint administrativo “zero‑and‑refetch” existe e **não apaga RAW**; timeline fica vazia antes do *refetch*.

### 5) Ingestão B3 — Full
- [ ] `POST /b3/fetch/full` com `from..to` parametrizados.
- [ ] **RAW** persistido; **normalized/ledger** populado.
- [ ] Métricas: `b3_requests_total` ↑ e `ledger_ops_ingested_total{source="B3_RAW"}` ↑.
- [ ] Logs com `traceId/requestId/tenantId` e `cpfMasked`.

### 6) Ingestão B3 — Incremental
- [ ] `POST /b3/fetch/incremental` para o **dia seguinte**.
- [ ] **Sem duplicações**; idempotência preservada.
- [ ] `b3_requests_total` ↑ moderado; erros B3 em 5m ≤ `GATE_MAX_B3_ERROR_5M`.

### 7) Inconsistências (1.17)
- [ ] `POST /ops/inconsistencies/scan?cpf=` executa análise.
- [ ] `GET /ops/inconsistencies?cpf=` retorna itens esperados (ex.: *posição sem compra*, *venda sem compra*).
- [ ] Métrica `inconsistency_open_gauge` > 0 **antes** do auto‑fix.

### 8) Auto‑fix — System Ops (1.18)
- [ ] `POST /ops/system/autofix?cpf=` cria operações **sintéticas/neutras**.
- [ ] `GET /ops/inconsistencies?cpf=` → **0** (ou ≤ `GATE_MAX_INCONSISTENCIES_AFTER_AUTOFIX`).
- [ ] Ledger mostra `source="SYSTEM_SYNTHETIC"`; métrica `ledger_ops_ingested_total{source="SYSTEM_SYNTHETIC"}` ↑.
- [ ] Idempotência: reexecutar não duplica system ops (marca obsoletas / *upsert*).

### 9) Override Manual (1.19)
- [ ] `POST /ops/manual` realiza substituição **controlada** de uma sintética.
- [ ] Ledger reflete a **manual** ativa; sintética relacionada marcada como **inativa**.
- [ ] Métricas/Logs registram o evento (sem PII).

### 10) Corporate Actions — Projeção com *comicota* (1.24)
- [ ] `POST /catalog/ca/projection/preview` retorna proposta coerente.
- [ ] `POST /catalog/ca/projection/apply` aplica **versão N** com **arredondamento para baixo** (sem fração).
- [ ] Resto contabilizado como `FRACTION_ADJUSTMENT` (neutralidade econômica).
- [ ] Idempotência por **versão**: re‑`apply(N)` não duplica; `status` reflete a versão ativa.

### 11) Timeline & Export (1.20)
- [ ] `GET /ops/timeline?cpf=&from=&to=&respectPolicy=true` consolida B3 + system + manual + CA.
- [ ] `GET /ops/timeline/export?format=csv` baixa arquivo válido (≥1 linha; cabeçalhos corretos).

### 12) Observabilidade
- [ ] **Traces** visíveis para as etapas (HTTP server, HTTP client B3, DB, jobs).
- [ ] **Métricas** esperadas durante a suite:  
  - `b3_requests_total` ↑  
  - `ledger_ops_ingested_total{source="B3_RAW|SYSTEM_SYNTHETIC|USER_MANUAL"}` ↑  
  - `inconsistency_open_gauge` pico e queda após auto‑fix  
  - `ca_projection_runs_total{status}` ↑ quando aplicar  
  - `http_request_duration_seconds` p95 ≤ `GATE_MAX_P95_API`
- [ ] **Logs** com `traceId/spanId/requestId/tenantId` e `cpfMasked` (`***1234`). **Sem PII** em métricas.

### 13) Idempotência e Sumário
- [ ] Reexecutar **4→11** produz **mesmos totais** (sem duplicações).
- [ ] `artifacts/e2e_smoke_summary.json` e `artifacts/e2e_smoke_summary.md` gerados com:  
  `result`, `cpfMasked`, `totals` (rawSaved, ledgerTotal, inconsistenciesBefore/After, systemOpsCreated, manualOverrides, caOpsCreated), `limits` (p95, b3ErrorsLast5m).
- [ ] Suite retorna **código 0** em caso de PASS.

### 14) Segurança & Tenant
- [ ] `X-Tenant-Id` em todas as chamadas; RBAC respeitado.
- [ ] CPF **nunca** aparece completo em logs, métricas ou artifacts (apenas `***1234`).


---

## 🧪 Passos de Validação Manual (rápidos)

1) **Rodar local (shell)**  
```bash
export BASE_URL=http://localhost:8080
export AUTH_TOKEN="bearer-token"
export E2E_TEST_TENANT_ID="tenant-uuid"
export E2E_TEST_CPF="00000000000"
export E2E_TEST_FROM="2019-12-01"
export E2E_TEST_TO="$(date +%F)"
export GATE_MAX_P95_API=1.5
export GATE_MAX_B3_ERROR_5M=50
export GATE_MAX_INCONSISTENCIES_AFTER_AUTOFIX=0

bash scripts/e2e_smoke.sh
```

2) **Checar artifacts**  
```bash
cat artifacts/e2e_smoke_summary.json | jq
cat artifacts/e2e_smoke_summary.md
```

3) **Idempotência**  
Execute o `scripts/e2e_smoke.sh` novamente e confirme **mesmos totais**.

4) **Observabilidade**  
Abra Grafana/Tempo e confirme métricas/traços das etapas principais; verifique que **nenhuma label** contém CPF/ticker específico.

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:

```
Validação concluída: Etapa 1.29 (Smoke Reconciliação & CA — E2E) está 100% conforme.
```

Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
