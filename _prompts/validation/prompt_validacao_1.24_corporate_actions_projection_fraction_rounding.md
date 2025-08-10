# Prompt de Validação 1.24v — Corporate Actions Projection & Fraction Rounding (aplicação + *comicota*) — **suno-wallets**

## 🎯 Objetivo
Validar que a **projeção de eventos corporativos** gera **operações sintéticas** no ledger (**ADJUSTMENT**, **FRACTION_ADJUSTMENT** e opcional **CASH_ADJUSTMENT**) com **neutralidade econômica**, regra de **comicota (sempre floor)**, **idempotência**, **versionamento/rollback**, **segurança multi‑tenant**, **observabilidade** e **performance**.  
**RAW/Normalized não são alterados** nesta etapa.

> Escopo: migrations (`ca_projection_runs`, colunas/índices no ledger), services/repos/controllers (preview/apply/rollback/status), integração com `PriceLookupPort`, locks, métricas/logs, Swagger/Collections e testes (unit/integration).


---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos com **nomes em inglês** e **comentários pt‑BR**:
  - `src/Domain/CorporateActions/ca_types.go`
  - `src/Application/CorporateActions/ca_projection_service.go`
  - `src/Infrastructure/CorporateActions/ca_projection_repository.go`
  - `src/Api/Controllers/CorporateActions/ca_projection_controller.go`
  - Testes: `src/Tests/Unit/CorporateActions/...`, `src/Tests/Integration/CorporateActions/...`
- [ ] Controllers apenas **orquestram**; regras no **Service**; BD via **Repository**.
- [ ] Dependências externas via **ports** (ex.: `PriceLookupPort`).

### 2) Banco & Migrations
- [ ] Tabela `ca_projection_runs` criada:
  - Colunas: `id, tenant_id, cpf?, version, status(DRAFT|APPLIED|ROLLED_BACK|ERROR), effective_from, effective_to, params(JSONB), created_at/by, completed_at, notes`.
  - Índice único `uq_ca_runs_tenant_version`.
- [ ] Ledger `b3_operations_ledger` com colunas:
  - `generated_by_ca_event_id UUID`, `ca_version_applied INTEGER`.
  - Índice único `uq_ops_ca_dedupe(tenant_id, cpf, ticker, operation_date, operation_type, source, generated_by_ca_event_id, ca_version_applied)`.
- [ ] **Nenhuma** alteração em RAW/Normalized.

### 3) Regras & Matemática (por evento)
- **SPLIT / REVERSE_SPLIT** `ratio a:b`
  - [ ] `newQtyTheoretical = oldQty * (b/a)`; **`newQty = floor(newQtyTheoretical)`** (*comicota*).  
  - [ ] `fraction = newQtyTheoretical - newQty` registrada em **FRACTION_ADJUSTMENT** (negativa).
- **BONUS** `percent` (ex.: 0.10 = +10%)
  - [ ] `bonusQtyTheoretical = oldQty * percent`; **`bonusQty = floor(...)`**; fração registrada.
- **MERGER** `ratio a:b`, `toTicker`
  - [ ] `newQty(toTicker) = floor(oldQty(fromTicker) * (b/a))`;  
        gerar `ADJUSTMENT` negativo no `fromTicker` e positivo no `toTicker` e fração em **FRACTION_ADJUSTMENT**.
- **SPIN_OFF** `ratio a:b`, `childTicker`
  - [ ] `childQty = floor(oldQty(parent) * (b/a))`; fração registrada; não reduzir `parent` salvo regra explícita.
- **Neutralidade econômica**
  - [ ] `ADJUSTMENT`/`FRACTION_ADJUSTMENT` **não** afetam P&L/base de custo (unit_price `NULL`, `reason_code='CA_PROJECTION'`).  
  - [ ] `CASH_ADJUSTMENT` (opcional) apenas se houver preço (via `PriceLookupPort`) **e** `CA_PROJECTION_ENABLE_CASH_IN_LIEU=true`; caso `AUTO_FIX_REQUIRE_PRICE=true` e sem preço → marcar **pending** no run e **não** gerar cash.
- [ ] **Somente leitura** do *ticker*/posição; **somente escrita** no ledger sintético.

### 4) Idempotência, Versões & Concorrência
- [ ] `preview` **não** persiste em ledger; retorna plano.
- [ ] `apply` grava com `source='SYSTEM_SYNTHETIC'`, `operation_date=effective_date`, `ca_version_applied=version`.  
- [ ] Reexecução do mesmo `version` **não** duplica (garantia via `uq_ops_ca_dedupe`).  
- [ ] `rollback` desativa (`is_active=false`) todas as operações da `version` no escopo (`tenantId[, cpf]`).  
- [ ] **Locks** por `(tenantId, cpf)` evitam corrida com outros jobs (1.11/1.14).

### 5) Endpoints & Contratos (Swagger tag: `Corporate Actions — Projection`)
- [ ] `POST /catalog/ca/projection/preview` aceita:
  ```json
  {"cpf": "00000000000|null", "from":"YYYY-MM-DD","to":"YYYY-MM-DD","tickers":null,"includeKinds":["SPLIT","REVERSE_SPLIT","BONUS","MERGER","SPIN_OFF"],"simulateCashInLieu":true}
  ```
  **Resposta**: plano com contagens, frações totais, cash estimado, lista simulada.
- [ ] `POST /catalog/ca/projection/apply` aceita `version?`, `dryRun=false`; aloca versão quando ausente.  
  **Resposta**: `version`, `created/skipped/pendingCash`, `byKind`, `byTicker`, `runId`.
- [ ] `POST /catalog/ca/projection/rollback` aceita `{ "version": <int>, "cpf": "000...|null" }`; desativa registros da versão.  
- [ ] `GET /catalog/ca/projection/status?cpf=&version=` retorna runs e estatísticas.  
- [ ] Códigos: 200/201/202/204 e 400/401/403/404/409/5xx com mensagens claras.  
- [ ] **Swagger/Postman/Bruno** atualizados em `/docs/ca/`.

### 6) Integrações & Dependências
- [ ] Uso de `PriceLookupPort` com *timeouts* e cache (para cash‑in‑lieu).  
- [ ] Reuso de canonicalização (1.23) para `canonical_ticker` quando necessário.  
- [ ] Ledger/timeline (1.20) exibe novas operações com `ca_version_applied` e links/razões.

### 7) Observabilidade
- [ ] Logs (JSON): `tenantId`, `scope` (cpf|tenant), `version`, `kinds`, `created/skipped/pending`, `fractionsTotal`, `durationMs`.  
- [ ] Métricas Prometheus:
  - `ca_projection_runs_total{status}`
  - `ca_projection_ops_created_total{operation_type}`
  - `ca_projection_fractions_total`
  - `ca_projection_duration_seconds` (histogram)
- [ ] **Sem** CPF em labels; CPF **mascarado** nos logs.

### 8) Segurança & LGPD
- [ ] **`X-Tenant-Id` obrigatório** e RBAC **admin** em todas as rotas.
- [ ] Sem tokens/segredos em logs; mascarar CPF (`***1234`).
- [ ] Validações: datas (`from ≤ to`), lista de `includeKinds`, existência de eventos no período.

### 9) Performance
- [ ] Índices no ledger suportam leituras por `(tenant_id, cpf, ticker, operation_date)`; uso de `canonical_ticker` quando aplicável.  
- [ ] Processamento em lotes com `CA_PROJECTION_DEFAULT_BATCH_SIZE`; `EXPLAIN ANALYZE` sem *seq scans* pesados.  
- [ ] Acesso à fonte de preço **cacheado** quando habilitado.

### 10) Testes Automatizados
**Integration — `src/Tests/Integration/CorporateActions/ca_projection_test.go`**
- [ ] **SPLIT**/**REVERSE_SPLIT** criam `ADJUSTMENT` correto + `FRACTION_ADJUSTMENT`; reexecutar **não** duplica.  
- [ ] **BONUS** aplica *floor* e fração; neutralidade mantida.  
- [ ] **MERGER** zera `fromTicker` (ajuste negativo) e cria positivo no `toTicker`; *comicota* aplicada.  
- [ ] **SPIN_OFF** cria quantidade no `childTicker`; fração registrada.  
- [ ] **CASH_IN_LIEU**: com preço → `CASH_ADJUSTMENT` criado; sem preço + `AUTO_FIX_REQUIRE_PRICE=true` → `pendingCash>0`, sem criação.  
- [ ] **Rollback** da `version` desativa operações geradas.  
- [ ] **Status** lista runs e estatísticas.

**Unit**
- [ ] Cálculo de `newQty`, `fraction`, *floor*; geração de operações; chaves de idempotência; alocação de `version`.

### 11) Documentação
- [ ] Swagger e coleções Postman/Bruno versionadas.  
- [ ] `docs/ca/projection.md` explica *comicota*, neutralidade, versionamento, rollback e exemplos.

---

## 🧪 Passos de Validação Manual (rápidos)

1) **Preview**  
```bash
curl -X POST "$BASE_URL/catalog/ca/projection/preview" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","from":"2024-01-01","to":"2024-12-31","includeKinds":["SPLIT","BONUS"],"simulateCashInLieu":true}'
```
**Esperado**: 200, plano com `fractionsTotal` e lista de operações simuladas.

2) **Apply (versão automática)**  
```bash
curl -X POST "$BASE_URL/catalog/ca/projection/apply" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","from":"2024-01-01","to":"2024-12-31","includeKinds":["SPLIT","BONUS"],"simulateCashInLieu":true,"dryRun":false}'
```
**Esperado**: 201/202 com `version` retornada; `created>0` se houver eventos; sem duplicação em nova execução.

3) **Status**  
```bash
curl "$BASE_URL/catalog/ca/projection/status?cpf=00000000000" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

4) **Rollback**  
```bash
curl -X POST "$BASE_URL/catalog/ca/projection/rollback" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"version": <VERSION>, "cpf":"00000000000"}'
```
**Esperado**: 200, contagens por `operation_type`; itens da versão ficam `is_active=false`.

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.24 (CA Projection & Fraction Rounding) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
