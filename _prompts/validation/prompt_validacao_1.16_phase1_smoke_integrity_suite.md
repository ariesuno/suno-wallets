# Prompt de Validação 1.16v — Phase 1 **Smoke & Integrity Suite** (E2E Runner + CI) — **suno-wallets**

## 🎯 Objetivo
Validar que o **runner E2E de smoke/integridade da Fase 1** executa o roteiro completo com **dados reais**, produz **relatórios consolidados** (JSON + Markdown), aplica **critérios de aceite objetivos** e **falha corretamente** quando qualquer etapa não atende.

> Cobertura esperada: 1.5, 1.6, 1.7/1.8, 1.9, 1.10, 1.11, 1.12, 1.13, 1.14, 1.15.

---

## ✅ Checklist de Validação

### 1) Arquitetura & Artefatos
- [ ] `cmd/e2e/phase1_smoke_runner.go` (CLI) existe e compila.
- [ ] `src/Application/E2E/phase1/phase1_smoke_orchestrator.go` implementa a orquestração por **estágios**.
- [ ] `src/Infrastructure/E2E/http_client.go` centraliza timeouts, retries, headers (`X-Tenant-Id`, `Authorization`) e **correlation-id**.
- [ ] **Reuso** de endpoints existentes (não reimplementar regras de negócio).

### 2) Pré‑flight & Configs
- [ ] Runner **falha** com mensagem clara se faltar qualquer env **obrigatório**: `TEST_TENANT_ID`, `TEST_CPF`, `API_BASE_URL`, `AUTH_BEARER`.
- [ ] Flag/env **`ALLOW_DESTRUCTIVE_RESET`** controla etapas destrutivas (reset real). **Sem** ela, apenas **dry‑run**.
- [ ] Suporte a flags CLI: `--tenant`, `--cpf`, `--since`, `--base-url`, `--token`, `--allow-reset`.

### 3) Roteiro de Estágios & Critérios
- **Health & Metrics**
  - [ ] `/health/live` → 200.
  - [ ] `/health/ready` → 200 (DB/Redis OK, migrations aplicadas).
  - [ ] `/health/details` → `UP` (ou `DEGRADED` aceitável quando dependência opcional estiver down) com componentes listados.
  - [ ] `/metrics` expõe famílias HTTP e `build_info`.
  - [ ] `/b3/health/auth` executado **apenas** se `OBS_ALLOW_B3_AUTH_HEALTH=true` (sem vazar tokens).
- **Preview (1.7/1.8)**  
  - [ ] Transações e Posições para janela curta (`last 7 days`): 200 + páginas ≥ 1 **ou** vazio justificado; **sem** persistir.
- **Reset & Full Re‑Fetch (1.13)**
  - [ ] **Dry‑run** com plano coerente (meses/páginas estimadas).
  - [ ] **Execução real** somente se `ALLOW_DESTRUCTIVE_RESET=true`, com `mode=archive`.
  - [ ] Sumário retorna `raw.saved > 0` **ou** justificativa coerente de zero (janela sem dados).
- **Normalization (1.10) Check**
  - [ ] Após re‑fetch, existem registros **normalized** correspondentes (verificado via endpoints de reports).
- **Incremental (1.14)**
  - [ ] `GET /b3/client/sync-window` retorna janela coerente.
  - [ ] `POST /b3/admin/incremental-from-last` (dry‑run → real) **não duplica** em execução subsequente sem `force`.
- **Reports (1.15)**
  - [ ] `raw-date-range` com `from`/`to` coerentes.
  - [ ] `summary` com `monthsWithTransactions ≥ 1` e `totalTransactions ≥ 1` (quando aplicável).
  - [ ] `tickers` paginado, tipos numéricos corretos.

### 4) Segurança & Multi‑tenant
- [ ] **Sempre** enviar `X-Tenant-Id` e `Authorization`.
- [ ] **CPF mascarado** em logs/relatórios (ex.: `***1234`); **zero** vazamento de tokens.
- [ ] Reset real **exige** confirmação no body (`confirm`) e `--allow-reset`/env ativo.

### 5) Observabilidade
- [ ] Logs estruturados do runner: `stage`, `status`, `durationMs`, `httpStatus`, `pages`, `records`, `errorCode`, `correlationId`.
- [ ] (Opcional) Métricas adicionais do runner:  
  `phase1_smoke_runs_total{result}`, `phase1_smoke_stage_duration_seconds{stage}`, `phase1_smoke_failures_total{stage}`.

### 6) Saídas & Estrutura de Relatórios
- [ ] Gera **`docs/e2e/Phase1SmokeReport.json`** e **`docs/e2e/Phase1SmokeReport.md`** na raiz do repo.
- [ ] JSON contém ao menos:
  ```json
  {
    "tenantId": "uuid",
    "cpfMasked": "***1234",
    "startedAt": "ISO-8601",
    "finishedAt": "ISO-8601",
    "durationMs": 0,
    "stages": {
      "preflight": {"ok": true, "details": "...", "durationMs": 0},
      "health": {"ok": true, "details": {...}, "durationMs": 0},
      "preview": {"ok": true, "details": {...}, "durationMs": 0},
      "resetDry": {"ok": true, "summary": {...}},
      "resetReal": {"ok": true, "summary": {...}},
      "normalize": {"ok": true, "counts": {...}},
      "incrementalDry": {"ok": true, "summary": {...}},
      "incrementalReal": {"ok": true, "summary": {...}},
      "reports": {"ok": true, "summary": {...}}
    },
    "verdict": "PASS|FAIL",
    "failures": [{"stage": "X", "reason": "mensagem objetiva"}]
  }
  ```
- [ ] **`verdict`** calculado objetivamente: se **qualquer** estágio obrigatório falhar → `FAIL`.

### 7) DevEx (Make/CI)
- [ ] **Makefile** com alvos: `smoke`, `smoke-dry`, `smoke-reset`, `smoke-inc` (passando envs).
- [ ] **GitHub Actions** (`.github/workflows/phase1_smoke.yml`): `workflow_dispatch` com inputs/`secrets`, anexando os relatórios como artefatos do job.

### 8) Testes Automatizados
- **E2E** (`src/Tests/E2E/phase1/phase1_smoke_test.go`):  
  - [ ] Em ambiente com dados, runner retorna `verdict=PASS`.
  - [ ] Sem `ALLOW_DESTRUCTIVE_RESET`, runner executa até dry‑run e retorna `PASS` (se critérios restantes OK).
  - [ ] Simulação de `429` em B3 → runner reintenta (backoff+jitter) e prossegue.
- **Unit**:  
  - [ ] Validação de configs/flags.
  - [ ] Composição do relatório e máscara de CPF.
  - [ ] Cálculo do `verdict`.

---

## 🧪 Execução Manual (exemplos)

### Via Make
```bash
make smoke \
  TEST_TENANT_ID="..." \
  TEST_CPF="00000000000" \
  API_BASE_URL="http://localhost:8080" \
  AUTH_BEARER="..." \
  ALLOW_DESTRUCTIVE_RESET=true
```

### Via CLI
```bash
go run ./cmd/e2e/phase1_smoke_runner.go \
  --tenant="..." \
  --cpf="00000000000" \
  --base-url="http://localhost:8080" \
  --token="..." \
  --allow-reset
```

> Esperado: geração de `docs/e2e/Phase1SmokeReport.json/.md` com **veredito `PASS`** em ambiente válido.

---

## 📌 Resultado Esperado
Se **todos** os itens estiverem OK, responda **exatamente**:
```
Validação concluída: Etapa 1.16 (Phase 1 Smoke & Integrity Suite) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
