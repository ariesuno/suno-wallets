# Prompt de Validação 1.20v — Timeline & Reconciliation Summary — **suno-wallets**

## 🎯 Objetivo
Validar que as **APIs de Timeline** e **Resumo de Reconciliação** expõem visão unificada (B3_RAW, SYSTEM_SYNTHETIC, USER_MANUAL) a partir do **ledger**, com **keyset pagination**, filtros, export seguro e snapshot de reconciliação coerente, mantendo **segurança, observabilidade, performance** e **documentação**.

> Escopo: serviços/queries de timeline e summary, views/índices auxiliares (se usados), endpoints `GET /ops/timeline`, `GET /ops/timeline/export`, `GET /ops/reconciliation/summary`, Swagger/Collections, testes unitários e de integração.


---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos presentes (nomes em inglês, comentários pt‑BR):
  - `src/Application/Ops/timeline_query_service.go`
  - `src/Application/Ops/reconciliation_summary_service.go`
  - `src/Infrastructure/Ops/timeline_repository.go`
  - `src/Infrastructure/Ops/reconciliation_repository.go`
  - `src/Api/Controllers/Ops/timeline_controller.go`
  - `src/Api/Controllers/Ops/reconciliation_controller.go`
  - Testes: `src/Tests/Unit/Ops/...`, `src/Tests/Integration/Ops/...`
- [ ] Controllers apenas orquestram; regras em Services; acesso a BD via Repositories.
- [ ] Somente leitura (não altera ledger/inconsistências).

### 2) Banco & Índices (e views opcionais)
- [ ] Reuso do `b3_operations_ledger` (1.18) e `b3_inconsistencies` (1.17).
- [ ] Índices ativos e utilizados (confirmar com `EXPLAIN ANALYZE`):
  - `b3_operations_ledger(tenant_id, cpf, canonical_ticker, operation_date DESC, id DESC)`
  - `b3_operations_ledger(tenant_id, cpf, source, is_active)`
  - `b3_inconsistencies(tenant_id, cpf, status, type)`
- [ ] Se criada a view `v_ops_timeline`, conferir colunas e filtro `is_active=TRUE`.

### 3) `GET /ops/timeline` (Swagger tag `Timeline`)
- [ ] Query params suportados: `cpf` (obrigatório), `tickers?`, `from?`, `to?`, `sources?`, `assetTypes?`, `canonicalize?`, `pageSize?` (≤ `TIMELINE_MAX_PAGE_SIZE`), `cursor?`.
- [ ] Ordenação **DESC** por `operation_date, id` com **keyset pagination** (`cursor` opaco base64).
- [ ] Resposta contém `items[]` + `nextCursor` + `count` (nº de itens retornados).
- [ ] Campos por item: `canonicalTicker/originalTicker`, `assetType`, `operationDate`, `operationType`, `source`, `quantity`, `unitPrice`, `currency`, `reasonCode`, `priceConfidence`, `links{...}`, `meta{createdAt,updatedAt,caVersionApplied?}`.
- [ ] Erros: 400/401/403/404/429/5xx com mensagens claras.
- [ ] LGPD: **CPF não aparece** no payload; logs mascaram CPF.

### 4) `GET /ops/timeline/export`
- [ ] Mesmos filtros do timeline + `format=csv|json` (default: `csv`) e `limit` (≤ `TIMELINE_EXPORT_MAX_ROWS`).
- [ ] Resposta com `Content-Disposition: attachment`.
- [ ] Rate‑limit e auditoria de export (logs com quem, quando, filtros, qtde).

### 5) `GET /ops/reconciliation/summary` (Swagger tag `Reconciliation`)
- [ ] Query `cpf` obrigatória.
- [ ] Campos na resposta:
  - [ ] `cpfMasked`, `period{from,to}`, `dataSourceMode`
  - [ ] Bloco **b3**: `lastFullFetchAt`, `lastIncrementalAt`, `latestWindow{from,to}`
  - [ ] **inconsistencies**: `open{total,byType}`, `resolved`, `overridden`, `lastScanAt`
  - [ ] **systemOps**: `created`, `byReason`, `lastAutoFixAt`
  - [ ] **userOverrides**: `manualOps`, `dedup{autoMerged,overridden,ignored}`, `lastActionAt`
  - [ ] `tickers[]` com contagens por origem e `lastOperationAt`
  - [ ] `notes[]`
- [ ] Códigos: 200/400/401/403/404/5xx.

### 6) Segurança & Multi‑tenant
- [ ] **Header `X-Tenant-Id` obrigatório**.
- [ ] RBAC: timeline/summary leitura; export somente **admin**.
- [ ] Logs: CPF **mascarado**; sem tokens/segredos.
- [ ] Todas as consultas filtram por `tenant_id` do header.

### 7) Observabilidade
- [ ] Logs estruturados (JSON): `tenantId`, `cpfMasked`, `filters`, `items/rows`, `durationMs`, `sourceMix`, `nextCursor?`.
- [ ] Métricas Prometheus:
  - `ops_timeline_requests_total{result}`
  - `ops_timeline_duration_seconds` (histogram)
  - `ops_timeline_export_requests_total{format}`
  - `recon_summary_requests_total{result}`
  - `recon_summary_duration_seconds` (histogram)
- [ ] Baixa cardinalidade — sem labels com CPF.

### 8) Performance
- [ ] **Keyset pagination** por padrão; `offset/limit` apenas quando explicitamente habilitado (dev).
- [ ] CTEs/aggregates eficientes; `EXPLAIN ANALYZE` sem *seq scan* desnecessário.
- [ ] Export em **streaming** para CSV; cap de linhas e tempo.

### 9) Testes Automatizados
**Integration — `src/Tests/Integration/Ops/timeline_and_summary_test.go`**
- [ ] Timeline mistura B3_RAW + SYSTEM_SYNTHETIC + USER_MANUAL corretamente (ordem e filtros).  
- [ ] `nextCursor` funciona (segunda página não duplica itens).  
- [ ] Export obedece filtros e limites; gera arquivo válido.  
- [ ] Summary reflete contagens coerentes com ledger + inconsistências para um CPF real.

**Unit**
- [ ] Encoder/decoder do `cursor`, validação de filtros, agregadores do resumo.

### 10) Documentação
- [ ] Swagger atualizado (duas tags) com exemplos realistas (sanitizados).  
- [ ] Coleções **Postman/Bruno** em `/docs/ops/` (timeline, export, summary).  
- [ ] `docs/ops/timeline_summary.md` com dicas de paginação e troubleshooting.

---

## 🧪 Passos de Validação Manual (rápidos)

1) **Timeline (1ª página)**  
```bash
curl "$BASE_URL/ops/timeline?cpf=00000000000&pageSize=50" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```
**Esperado**: 200, `items` ≤ 50, `nextCursor` presente quando houver próxima página.

2) **Timeline (2ª página com cursor)**  
```bash
curl "$BASE_URL/ops/timeline?cpf=00000000000&pageSize=50&cursor=<nextCursor_da_1a>" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

3) **Export CSV**  
```bash
curl -D headers.txt -o export.csv "$BASE_URL/ops/timeline/export?cpf=00000000000&format=csv&limit=10000" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```
**Esperado**: `Content-Disposition` com filename; tamanho ≤ limite configurado.

4) **Resumo**  
```bash
curl "$BASE_URL/ops/reconciliation/summary?cpf=00000000000" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.20 (Timeline & Reconciliation Summary) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
