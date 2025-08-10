# Prompt 1.16 — Phase 1 **Smoke & Integrity Suite** (E2E Runner + CI) — **suno-wallets**

Implemente uma **bateria de verificação ponta‑a‑ponta** para a **Fase 1** que execute, de forma **sequencial e automatizada**, os fluxos críticos com **dados reais** (sem mocks), consolidando um **relatório único** (JSON + Markdown) e **falhando** se qualquer etapa obrigatória não cumprir os critérios definidos.

> **Cobertura:** 1.5 (Observabilidade base), 1.6 (B3 client), 1.7/1.8 (Preview), 1.9 (RAW), 1.10 (Normalization), 1.11 (Daily Sync), 1.12 (Health/metrics), 1.13 (Reset & Full Re‑Fetch), 1.14 (Incremental), 1.15 (Reports).  
> **Sem dados fake.** Se faltar insumo, **abortar** e **pedir instrução** (não inventar).  
> **Multi‑tenant** e **segurança** estritas (mascarar CPF nos logs; nunca logar tokens/segredos).

---

## 🎯 Objetivos
1. Criar um **runner E2E** invocável via **CLI** e **Make/GitHub Actions**, que:
   - Valide **health** (liveness/readiness), **/metrics**, e **B3 auth** (quando habilitado).
   - Execute **reset & full re‑fetch (dry run → real)** para um CPF/tenant informados.
   - Execute **incremental on‑demand** (1.14) a partir do último marco ou `since`.
   - Gere **relatórios** (1.15) e **cruze** com expectativas mínimas (contagens > 0, coerência de intervalo).
   - **Consolide** métricas/tempos/erros num **relatório final** (JSON + MD).
2. Disponibilizar **gatilhos de segurança** para etapas destrutivas (reset).  
3. Integrar **observabilidade** (logs estruturados, métricas de duração/resultado) e **saídas versionadas** em `docs/e2e/`.

---

## 🧱 Arquitetura (DDD + CLI)
Crie/atualize os artefatos **(nomes em inglês, comentários pt‑BR)**:

- `cmd/e2e/phase1_smoke_runner.go`  
  CLI que orquestra as etapas; lê env/flags; imprime progressos e escreve relatórios.
- `src/Application/E2E/phase1/phase1_smoke_orchestrator.go`  
  Orquestrador com funções de etapa (health, preview, reset, refetch, incremental, reports).
- `src/Infrastructure/E2E/http_client.go`  
  Cliente HTTP reutilizável (timeout, retry defensivo, headers padrão `X-Tenant-Id` e `Authorization`).
- **Reuso** dos endpoints existentes (não reimplementar regras):  
  - Health: `/health/live`, `/health/ready`, `/health/details`, `/metrics`, `/b3/health/auth` (se flag permitir).
  - Preview: `/b3/preview/transactions`, `/b3/preview/positions` (1.7/1.8).
  - RAW ingest/reset: `/b3/admin/reset-and-refetch` (1.13).
  - Incremental: `/b3/admin/incremental-from-last`, `/b3/client/sync-window` (1.14).
  - Reports: `/b3/client/raw-date-range`, `/b3/client/summary`, `/b3/client/tickers` (1.15).

**Testes**  
- `src/Tests/E2E/phase1/phase1_smoke_test.go` (chama o orquestrador diretamente com envs reais).

---

## 🧩 Etapas do Runner (sequência e critérios)
1. **Preflight**
   - Verificar env obrigatórios: `TEST_TENANT_ID`, `TEST_CPF`, `API_BASE_URL`, `AUTH_BEARER`.
   - Se `ALLOW_DESTRUCTIVE_RESET != true`, **pular** etapas destrutivas (reset real) e executar **somente** dry‑run.
2. **Health & Metrics**
   - `GET /health/live` → 200; `GET /health/ready` → 200; `GET /health/details` → `UP` ou `DEGRADED` aceitável sem dependência B3.
   - `GET /metrics` → conter famílias HTTP e build_info.
   - (Opcional) `GET /b3/health/auth` **apenas** se `OBS_ALLOW_B3_AUTH_HEALTH=true`.
3. **Preview (sanidade B3) — 1.7/1.8**
   - Rodar **preview** de `transactions` e `positions` para uma janela curta (`last 7 days`).
   - Critério: HTTP 200, páginas ≥ 1 **ou** vazio com mensagem coerente; **sem** persistência.
4. **Reset & Full Re‑Fetch — 1.13**
   - **Dry run**: `POST /b3/admin/reset-and-refetch` com `dryRun=true` e `confirm` — retornar plano (meses/páginas estimadas).
   - **Execução real** *(somente se `ALLOW_DESTRUCTIVE_RESET=true`)*: mesmo endpoint `dryRun=false` `mode=archive`.
   - Critérios: plano coerente; execução retorna **sumário** com `raw.saved > 0` **ou** justificativa coerente quando intervalo não tem dados.
5. **Normalization Check — 1.10**
   - Verificar que os registros **normalized** foram produzidos após o full re‑fetch (via contagens das APIs de Reports).
6. **Incremental — 1.14**
   - `GET /b3/client/sync-window` → obter janela calculada;  
   - `POST /b3/admin/incremental-from-last` (dryRun → real).  
   - Critérios: janela coerente; segundo run sem `force` **não duplica**.
7. **Reports — 1.15**
   - `GET /b3/client/raw-date-range` → `from/to` coerentes (cobertura ≥ 1 mês quando há histórico).  
   - `GET /b3/client/summary` → `monthsWithTransactions ≥ 1` e `totalTransactions ≥ 1` (quando aplicável).  
   - `GET /b3/client/tickers` → lista **paginada**; validar estrutura e tipos numéricos.
8. **Consolidação de Resultados**
   - Montar **Phase1SmokeReport.json** + **Phase1SmokeReport.md** com tempos, contagens, status e **veredito final**.

---

## 📦 Saídas & Artefatos
- `docs/e2e/Phase1SmokeReport.json` (estrutura sugerida):
```json
{
  "tenantId": "uuid",
  "cpfMasked": "***1234",
  "startedAt": "ISO-8601",
  "finishedAt": "ISO-8601",
  "durationMs": 0,
  "stages": {
    "preflight": {"ok": true, "details": "...", "durationMs": 0},
    "health":    {"ok": true, "details": {...}, "durationMs": 0},
    "preview":   {"ok": true, "details": {...}, "durationMs": 0},
    "resetDry":  {"ok": true, "summary": {...}},
    "resetReal": {"ok": true, "summary": {...}},
    "normalize": {"ok": true, "counts": {...}},
    "incrementalDry":  {"ok": true, "summary": {...}},
    "incrementalReal": {"ok": true, "summary": {...}},
    "reports":   {"ok": true, "summary": {...}}
  },
  "verdict": "PASS|FAIL",
  "failures": [{"stage": "incrementalReal", "reason": "..." }]
}
```
- `docs/e2e/Phase1SmokeReport.md` (mesmo conteúdo em formato humano).
- Logs estruturados (JSON) com **correlação** por `x-correlation-id` (gerado no runner).

---

## ⚙️ Configuração (ENV/flags)
- `TEST_TENANT_ID` (uuid) — **obrigatório**
- `TEST_CPF` (string 11 dígitos) — **obrigatório**
- `API_BASE_URL` (ex.: `http://localhost:8080`)
- `AUTH_BEARER` (token para chamadas)
- `SINCE_OVERRIDE` (opcional, `YYYY-MM-DD`) — usado no incremental.
- `ALLOW_DESTRUCTIVE_RESET` (default: `false`) — exigido **true** para reset real.
- `OBS_ALLOW_B3_AUTH_HEALTH` (de 1.12) — se `true`, habilita `/b3/health/auth`.
- `E2E_REQUEST_TIMEOUT_MS` (default: `10000`)

**Flags CLI** (curtas): `--tenant`, `--cpf`, `--since`, `--base-url`, `--token`, `--allow-reset`.

---

## 🔐 Segurança & Multi‑tenant
- Incluir `X-Tenant-Id` e `Authorization` em **todas** as chamadas.  
- **Mascarar CPF** nos logs/relatórios (exibir apenas os 4 últimos).  
- **Nunca** incluir tokens nos logs/relatórios.  
- Desabilitar etapas destrutivas se `--allow-reset` **não** for passado (ou env).

---

## 📊 Observabilidade
- Métricas Prometheus adicionais (emissão pelo runner via `/metrics` interno do app se aplicável ou counters do próprio runner):  
  - `phase1_smoke_runs_total{result}`  
  - `phase1_smoke_stage_duration_seconds{stage}` (histogram)  
  - `phase1_smoke_failures_total{stage}`
- Logs do runner com campos: `stage`, `status`, `durationMs`, `httpStatus`, `pages`, `records`, `errorCode`.

---

## 🧪 Testes (obrigatório)
- **E2E** (`src/Tests/E2E/phase1/phase1_smoke_test.go`):  
  - Invoca o orquestrador com env válidos; valida `verdict=PASS` em ambiente com dados.  
  - Cenários negativos (sem `ALLOW_DESTRUCTIVE_RESET`) → pular reset real e **PASS** com dry‑run.  
  - Cenários com rate‑limit simulado (429) → runner reintenta e prossegue.
- **Unit** (mínimo): validação de configs, composição do relatório, máscara de CPF, cálculo de verdict.

---

## 🛠️ DevEx (Make/CI/Docs)
- **Makefile**: alvos `smoke`, `smoke-dry`, `smoke-reset`, `smoke-inc`.
- **GitHub Actions** (`.github/workflows/phase1_smoke.yml`): job manual (`workflow_dispatch`) com `secrets` para `AUTH_BEARER`, `TEST_TENANT_ID`, `TEST_CPF`; artefatos do relatório anexados ao job.
- **Docs**: `docs/e2e/phase1_smoke.md` com exemplos de uso, requisitos, troubleshooting e explicação dos critérios.

---

## 🧾 Critérios de Aceite
- Runner produz **Phase1SmokeReport.json** e **.md** em `docs/e2e/`.
- Em ambiente com dados: **todas** as etapas retornam **200** e **contagens > 0** onde aplicável.
- **Idempotência**: reexecutar incremental sem `force` **não** duplica RAW/Normalized.
- Reset **real** somente quando `ALLOW_DESTRUCTIVE_RESET=true`, com `mode=archive` e sumário coerente.
- Logs/métricas gerados; **nenhum** vazamento de PII/segredos.

---

## 🔚 Entregáveis
- CLI + orquestrador + HTTP client + testes + Make + GitHub Actions + docs.  
- Atualização do **Swagger** (tag `E2E/Smoke` opcional) e das collections do Postman/Bruno (roteiro de execução).

> **Observação final**: Todos os nomes **em inglês**; comentários **pt‑BR**. Nunca criar/usar dados fake. Caso o runner necessite de um insumo ausente (ex.: token expirado, cert mTLS), **abortar com erro claro** e **não** prosseguir as etapas seguintes.
