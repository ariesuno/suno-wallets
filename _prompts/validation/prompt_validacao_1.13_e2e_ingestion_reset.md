# Prompt de Validação 1.13v — E2E Ingestion Runner + Reset & Full Re‑Fetch (admin) — **suno-wallets**

## 🎯 Objetivo
Garantir que a orquestração **reset → ingest (1.9) → normalize (1.10)** para um **CPF** está correta, segura (LGPD), idempotente, observável e fiel ao contrato do endpoint admin.

> **Pré‑requisitos**: 1.6 (B3OfficialClient), 1.7/1.8 (preview), 1.9 (RAW Persistence), 1.10 (Normalization), 1.11 (Sync State), 1.12 (Health/Metrics) concluídos.

---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos presentes:
  - `src/Application/B3/e2e/e2e_orchestrator.go`
  - `src/Infrastructure/B3/e2e/reset_repository.go`
  - `src/Api/Controllers/B3/admin_controller.go` (rota `POST /b3/admin/reset-and-refetch`)
- [ ] **Reuso** de 1.9 (ingest) e 1.10 (normalize) — **sem** reimplementar lógicas de fetch/persist/parse.
- [ ] Nomes **em inglês** e comentários **pt‑BR**; arquivos coesos (≤ ~300 linhas).

### 2) Segurança, LGPD & Multi‑tenant
- [ ] Rota é **admin‑only** (RBAC) e exige **`X-Tenant-Id`**.
- [ ] Payload exige `"confirm": "RESET_AND_REFETCH"`; para `mode="hard-delete"` exige `"confirmHard": "YES_DELETE"`.
- [ ] **CPF mascarado** em logs; **nunca** logar tokens/segredos.
- [ ] **Sem** qualquer limpeza automática de RAW — somente via este endpoint (manual e auditado).

### 3) Reset Seguro (Archive/Hard‑delete)
- [ ] `mode="archive"` **move** dados do CPF de:
  - `b3_raw_data_client` → `b3_raw_data_client_archive`
  - `b3_normalized_transactions` → `_archive`
  - `b3_normalized_positions` → `_archive`
  - preenchendo `archived_at` e `archived_by`.
- [ ] `mode="hard-delete"` só executa com `"confirmHard": "YES_DELETE"`.
- [ ] `b3_fetched_periods` limpo para o CPF.
- [ ] `b3_sync_state` resetado (`last_*_sync_at=null`, `needs_reprocess=false`, `failure_count=0`).

### 4) Janela & Planejamento
- [ ] Cálculo de janela mensal `start→end` usando `startOverride`/`endOverride` **ou** `B3_API_EARLIEST_DATE` → `today()`.
- [ ] **`dryRun=true`** retorna **plano** completo (meses/páginas estimadas) **sem** mudanças no BD.
- [ ] Locks por `(tenantId, cpf)` durante todo o E2E (não permite execuções paralelas).

### 5) Ingest (1.9) & Normalize (1.10)
- [ ] Ingest chamado **mês a mês** por `dataType × assetType` com `force=true` no reset completo.
- [ ] Normalização executada após ingest para o mesmo escopo.
- [ ] **Idempotência** validada: executar 2x com `force=false` não duplica RAW/Normalized.
- [ ] Tratamento de `429/5xx` com retry/backoff+jitter (política 1.6) e timeouts.

### 6) Observabilidade
- [ ] **Logs estruturados (JSON)** contendo: `tenantId`, `cpfMasked`, `mode`, `force`, `dryRun`, `stage`, `durationMs`, totais por etapa.
- [ ] **Métricas Prometheus**:
  - `b3_e2e_runs_total{result="success|error", mode, dataType, assetType}`
  - `b3_e2e_duration_seconds` (histogram)
  - `b3_e2e_errors_total{stage="reset|ingest|normalize"}`

### 7) Migrations & Auditoria
- [ ] Tabelas `_archive` existem com mesmo schema + `archived_at/by`.
- [ ] (Opcional) `b3_admin_resets` registrando cada execução (tenant, cpf, modo, tempos, totais).
- [ ] Índices mínimos preservados para consultas/auditoria.

### 8) Contrato do Endpoint
- [ ] **Request body** aceito conforme especificação:
  ```json
  {
    "cpf":"00000000000",
    "assetTypes":["equity"],
    "dataTypes":["transactions","positions"],
    "startOverride":null,
    "endOverride":null,
    "force":true,
    "dryRun":false,
    "mode":"archive",
    "confirm":"RESET_AND_REFETCH",
    "confirmHard":null
  }
  ```
- [ ] **Response** retorna sumário:
  ```json
  {
    "raw":{"saved":0,"skipped":0,"errors":0,"monthsProcessed":0,"pagesProcessed":0},
    "normalized":{"inserted":0,"updated":0,"skipped":0,"errors":0},
    "startedAt":"ISO-8601",
    "finishedAt":"ISO-8601",
    "durationMs":0,
    "mode":"archive",
    "force":true,
    "dryRun":false
  }
  ```
- [ ] Códigos de retorno e erros padronizados (200/400/401/403/409/429/5xx) com mensagens claras.

### 9) Testes
- **Integration —** `src/Tests/Integration/B3/e2e_reset_refetch_test.go`
  - [ ] `dryRun=true` → **plano** sem alterar o BD.
  - [ ] Execução real (`mode=archive`) move para `_archive`, salva RAW (1.9) e gera Normalized (1.10); sumário coerente.
  - [ ] Segunda execução com `force=false` → **sem duplicidade**.
  - [ ] Simulação de `429/5xx` → observar retries e métricas de erro no **stage** correto.
- **Unit —** `src/Tests/Unit/B3/e2e/e2e_orchestrator_test.go`
  - [ ] Planejamento mensal, locks e composição de sumário.
  - [ ] Rollback por etapa e reporte de erros.

### 10) Documentação
- [ ] **Swagger** (tag `B3 Admin`) com contrato e exemplos.
- [ ] **Postman/Bruno** atualizados (exemplos `archive` e `hard-delete`).
- [ ] `docs/b3/e2e_reset_refetch.md` com cURLs e troubleshooting.

---

## 🧪 Passo‑a‑passo sugerido (execução manual)

1. **Dry run** (plano):
   ```bash
   curl -X POST "$BASE_URL/b3/admin/reset-and-refetch" \
     -H "Content-Type: application/json" \
     -H "X-Tenant-Id: $TENANT" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"cpf":"00000000000","assetTypes":["equity"],"dataTypes":["transactions","positions"],"force":true,"dryRun":true,"mode":"archive","confirm":"RESET_AND_REFETCH"}'
   ```
   - Esperado: **200** com **plano**; BD **inalterado**.

2. **Execução real** (archive):
   ```bash
   curl -X POST "$BASE_URL/b3/admin/reset-and-refetch" \
     -H "Content-Type: application/json" \
     -H "X-Tenant-Id: $TENANT" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"cpf":"00000000000","assetTypes":["equity"],"dataTypes":["transactions","positions"],"force":true,"dryRun":false,"mode":"archive","confirm":"RESET_AND_REFETCH"}'
   ```
   - Esperado: **200** com **sumário**; dados movidos para `_archive`; **RAW** e **Normalized** refeitos.

3. **Idempotência** (force=false):
   - Repetir a execução **sem** `force=true`.  
   - Esperado: **sem duplicidade** em RAW/Normalized.

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.13 (E2E Reset & Full Re‑Fetch) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens faltantes e a **correção objetiva** para cada um.
