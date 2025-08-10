# Prompt de Validação 1.18v — System Operations (auto‑reconciliação) — **suno-wallets**

## 🎯 Objetivo
Validar que o **auto‑fix** (1.18) transforma inconsistências do 1.17 em **operações de sistema** no **ledger unificado** com **idempotência, auditoria, segurança** e **observabilidade**, **sem** tocar em RAW/Normalized, e respeitando as regras de preço/quantidade configuradas.

> Escopo: regras **OPENING_BALANCE_MISSING → OPENING_BALANCE** e **SELL_WITHOUT_BUY → BUY (espelho P&L=0)**; endpoints `POST /reconciliation/auto-fix` e `POST /reconciliation/auto-fix/{id}`; ledger `b3_operations_ledger`; integração de preço; testes unitários e de integração.

---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos presentes com **nomes em inglês** e **comentários pt‑BR**:
  - `src/Application/B3/reconciliation/system_operations_service.go`
  - `src/Infrastructure/B3/reconciliation/system_operations_repository.go`
  - `src/Api/Controllers/B3/reconciliation_controller.go` (rotas de auto‑fix)
  - Tests: `src/Tests/Unit/B3/reconciliation/...` e `src/Tests/Integration/B3/reconciliation/...`
- [ ] Fluxo: **Controller → Service → Repository**; **sem** uso direto de DB no controller.
- [ ] **RAW/Normalized** permanecem **somente leitura** (confirmar via testes/grep).

### 2) Ledger & Migrations
- [ ] Tabela `b3_operations_ledger` criada com colunas exigidas:
  - `tenant_id, cpf, ticker, asset_type, operation_date, operation_type, source, quantity, unit_price, currency`
  - rastros: `reason_code, price_confidence, generated_by_inconsistency_id, supersedes_operation_id, superseded_by_operation_id, is_active, created_at/by, updated_at/by`
- [ ] Enums: `operation_source (B3_RAW|SYSTEM_SYNTHETIC|USER_MANUAL)`, `operation_type`, `price_confidence (MARKET_CLOSE|MIRRORED_SELL|DERIVED_POSITION|UNKNOWN)`
- [ ] Índices:
  - [ ] `ix_ops_ledger_lookup (tenant_id, cpf, ticker, operation_date)`
  - [ ] `uq_ops_ledger_system_dedupe` (idempotência de system ops)
- [ ] **Nenhuma** coluna PII sensível extra além de `cpf` (11 dígitos).

### 3) Regras de Negócio (auto‑fix)
- **OPENING_BALANCE_MISSING → OPENING_BALANCE**
  - [ ] `operation_date = B3_API_EARLIEST_DATE` **ou** a primeira `reference_date` com posição (conforme implementação documentada).
  - [ ] `quantity = saldo_posicao_na_data - net_tx_ate_data` (nunca negativa).
  - [ ] `unit_price`:
    - [ ] Busca via `PriceLookupPort` → `price_confidence='MARKET_CLOSE'` quando disponível.
    - [ ] Fallback derivado de posição (`value/qty`) → `DERIVED_POSITION`.
    - [ ] Se `AUTO_FIX_REQUIRE_PRICE=true` e sem preço: **não** gerar; marcar inconsistência **OPEN** com `details.reason='PENDING_PRICE'`.
    - [ ] Se `AUTO_FIX_REQUIRE_PRICE=false`: permitir `unit_price=NULL` com `price_confidence='UNKNOWN'`.
  - [ ] `reason_code='OPENING_BALANCE_PRE_API'`.

- **SELL_WITHOUT_BUY → BUY (espelho P&L=0)**
  - [ ] `operation_date = data_da_primeira_venda_inconsistente`.
  - [ ] `quantity` cobre totalmente o negativo (ou iguala a venda).
  - [ ] `unit_price` **igual ao da venda** → `price_confidence='MIRRORED_SELL'`.
  - [ ] `reason_code='ZERO_PNL_PRE_API'`.

- [ ] Após criar operações: atualizar `b3_inconsistencies.status='RESOLVED'` e anexar `generated_op_ids` nos `details`.
- [ ] **Idempotência**: reexecutar auto‑fix não duplica (checar `uq_ops_ledger_system_dedupe`).

### 4) Endpoints & Contratos (Swagger: `B3 Reconciliation`)
- [ ] `POST /reconciliation/auto-fix` *(admin‑only)* aceita body:
  ```json
  {
    "cpf": "00000000000",
    "types": ["OPENING_BALANCE_MISSING","SELL_WITHOUT_BUY"],
    "tickers": null,
    "dryRun": false,
    "force": false,
    "concurrency": 2
  }
  ```
  - [ ] `dryRun=true` só simula; `dryRun=false` persiste.
  - [ ] `force=true` permite reavaliar inconsistências **RESOLVED** (idempotente).
  - [ ] Resposta sumariza `{created, skipped, pending, resolvedIds, pendingIds}` + prévias/IDs.
- [ ] `POST /reconciliation/auto-fix/{id}` executa apenas para a inconsistência informada.
- [ ] **Códigos**: 200/400/401/403/409/5xx com mensagens claras.
- [ ] **Swagger**, **Postman/Bruno** atualizados em `/docs/reconciliation/`.

### 5) Integração de Preço (PriceLookupPort)
- [ ] Porta `GetClosingPrice(ctx, ticker, date)` implementada com **adapter** inicial (ex.: derivação via posição normalizada ou stub que retorna “não disponível” de forma segura).
- [ ] Tratamento de erro e *timeouts*; **nunca** logar credenciais.
- [ ] Switch de comportamento por `AUTO_FIX_REQUIRE_PRICE` documentado e coberto em testes.

### 6) Segurança & Multi‑tenant
- [ ] **`X-Tenant-Id` obrigatório**; RBAC admin nas rotas; logs com **CPF mascarado** (`***1234`).
- [ ] **Sem** escrita fora do tenant do header.
- [ ] Rate‑limit defensivo opcional no auto‑fix.

### 7) Observabilidade
- [ ] **Logs estruturados** (JSON): `tenantId`, `cpfMasked`, `types`, `tickers`, `created/skipped/pending`, `durationMs`, `priceConfidenceMix`.
- [ ] **Métricas Prometheus** incrementadas:
  - `b3_autofix_runs_total{result}`
  - `b3_autofix_ops_created_total{reason_code}`
  - `b3_autofix_pending_total{reason}`
  - `b3_autofix_duration_seconds` (histogram)
- [ ] Evitar **alta cardinalidade** (CPF só nos logs).

### 8) Performance & Concorrência
- [ ] Lock por `(tenantId, cpf)` impede corrida com jobs 1.13/1.14.
- [ ] Upserts em lote; respeito a `concurrency` para consultas de preço/posição.
- [ ] Índices presentes e usados (`EXPLAIN ANALYZE`) nas consultas ao ledger e inconsistências.

### 9) Testes Automatizados
- **Integration** — `src/Tests/Integration/B3/reconciliation/system_operations_test.go`
  - [ ] `OPENING_BALANCE_MISSING` gera `OPENING_BALANCE` correto (datas/qty/preço).  
  - [ ] `SELL_WITHOUT_BUY` gera `BUY` espelho no mesmo dia/preço, zerando negativo.  
  - [ ] `AUTO_FIX_REQUIRE_PRICE=true` sem preço → **PENDING** (não cria op).  
  - [ ] `AUTO_FIX_REQUIRE_PRICE=false` sem preço → cria com `unit_price=NULL` e `price_confidence='UNKNOWN'`.  
  - [ ] **Idempotência**: reexecutar não duplica; `force=true` reavalia sem duplicar.  
  - [ ] **Dry‑run** não persiste.
- **Unit**
  - [ ] Cálculo de `quantity`, `reason_code`, `price_confidence`, e composição da resposta `{created, skipped, pending…}`.

### 10) Documentação
- [ ] `docs/b3/auto_fix.md` descreve regras, price fallback, PENDING_PRICE e cenários típicos.
- [ ] Swagger e coleções atualizadas e versionadas.

---

## 🧪 Passos de Validação Manual (rápido)

1) **Dry‑run (CPF inteiro)**  
```bash
curl -X POST "$BASE_URL/reconciliation/auto-fix" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","types":["OPENING_BALANCE_MISSING","SELL_WITHOUT_BUY"],"dryRun":true}'
```
**Esperado**: 200 com plano de criação; **BD inalterado**.

2) **Execução real (CPF inteiro)**  
```bash
curl -X POST "$BASE_URL/reconciliation/auto-fix" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","types":["OPENING_BALANCE_MISSING","SELL_WITHOUT_BUY"],"dryRun":false}'
```
**Esperado**: 200 com `{created>0}` quando houver inconsistências; registros no `b3_operations_ledger`; inconsistências `RESOLVED`.

3) **Por ID**  
```bash
curl -X POST "$BASE_URL/reconciliation/auto-fix/<inconsistencyId>" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

4) **Idempotência** (rodar o passo 2 novamente)  
**Esperado**: **nenhuma duplicação** (ver constraint única).

5) **Com `AUTO_FIX_REQUIRE_PRICE=true`** (sem fonte de preço)  
**Esperado**: inconsistências permanecem **OPEN** com `details.reason='PENDING_PRICE'` (sem operações criadas).

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.18 (System Operations) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
