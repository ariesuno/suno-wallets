# Prompt de Validação 1.23v — Corporate Actions Canonicalization (Ticker mapping & histórico) — **suno-wallets**

## 🎯 Objetivo
Confirmar que a **camada de canonicalização** (catálogo canônico + aliases + eventos de identidade) está funcional, **performática** e **auditável**, e que o **backfill** de `canonical_ticker` no ledger ocorre de forma **idempotente**, **multi‑tenant** e **segura** — **sem** alterar quantidade/preço/RAW.

> Escopo: migrations (`instruments`, `ticker_aliases`, `corporate_action_events` e coluna `canonical_ticker` no ledger), services/repos/controllers (`resolve`, `history`, CRUD admin, backfill`), cache LRU, observabilidade, Swagger/Collections, e testes (unit/integration).


---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos presentes com **nomes em inglês** e **comentários pt‑BR**:
  - `src/Domain/Catalog/instrument.go`
  - `src/Application/Catalog/canonicalization_service.go`
  - `src/Infrastructure/Catalog/catalog_repository.go`
  - `src/Api/Controllers/Catalog/catalog_controller.go`
  - Testes: `src/Tests/Unit/Catalog/...`, `src/Tests/Integration/Catalog/...`
- [ ] Services não acessam BD direto; usam Repositories.
- [ ] **Sem projeções financeiras** aqui (nada de mudar `quantity/price`/criar operações).

### 2) Banco & Migrations
- [ ] Tabelas criadas conforme o prompt 1.23:
  - `instruments (tenant_id, canonical_ticker, asset_type, status, isin?)` + **UNIQUE (tenant_id, canonical_ticker)**
  - `ticker_aliases (tenant_id, instrument_id, alias_ticker, valid_from?, valid_to?, source?)` + **UNIQUE (tenant_id, alias_ticker)**
  - `corporate_action_events (tenant_id, event_type, effective_date, from_ticker?, to_ticker?, notes?)`
- [ ] Ledger possui coluna `canonical_ticker` e índices:
  - `ix_ops_canonical_lookup (tenant_id, cpf, canonical_ticker, operation_date DESC, id DESC)`
- [ ] Demais índices:
  - `instruments(tenant_id, canonical_ticker)`
  - `ticker_aliases(tenant_id, alias_ticker)`
  - `ix_ca_events_lookup (tenant_id, effective_date DESC, event_type)`

### 3) Regras de Resolução (ResolveTicker)
- [ ] Prioridade: **ALIAS** > **SELF** > **CA_EVENT** > **UNKNOWN**.
- [ ] `valid_from/valid_to` respeitados quando fornecidos.
- [ ] `CA_EVENT` usa `effective_date` para mapear `from_ticker → to_ticker`/migrações.
- [ ] Multi‑tenant estrito: consultas sempre filtram por `tenant_id`.
- [ ] Em caso de não resolução: retornar `UNKNOWN` e **não** falhar.

### 4) Backfill do Ledger
- [ ] Endpoint/serviço **admin** `backfill/ledger-canonical` implementado com:
  - Filtros por `cpf`, `tickers?`, `from?`, `to?`, `batchSize`, `dryRun`.
  - Execução **idempotente**: reexecutar não altera quando já preenchido corretamente.
  - Processamento em **lotes**; lock por `(tenantId, cpf)`.
  - **Somente leitura** do *ticker* original; **somente escrita** em `canonical_ticker`.
- [ ] Não tocar em RAW/Normalized; não criar/alterar operações.

### 5) Endpoints & Contratos (Swagger tags `Catalog`, `Catalog Admin`)
- [ ] `GET /catalog/tickers/resolve?symbol=&date=` retorna:
  ```json
  { "canonicalTicker":"ITSA4", "resolution":"ALIAS|SELF|CA_EVENT|UNKNOWN", "instrumentMeta":{...} }
  ```
- [ ] `GET /catalog/ca/history?ticker=` lista eventos de identidade para o símbolo.
- [ ] `POST /admin/catalog/instruments` cria/atualiza um canônico.
- [ ] `POST /admin/catalog/aliases` cria alias (com período opcional).
- [ ] `POST /admin/catalog/ca-events` cria evento (RENAME|MERGE|SPIN_OFF|TICKER_MOVE).
- [ ] `POST /admin/catalog/backfill/ledger-canonical` executa backfill (dryRun/real).
- [ ] Códigos: 200/201/204/400/401/403/404/409/5xx com mensagens claras.  
- [ ] **Postman/Bruno** e **Swagger** atualizados em `/docs/catalog/`.

### 6) Observabilidade
- [ ] Logs estruturados (JSON): `tenantId`, `action`, `symbol`, `resolution`, `durationMs`, `affectedRows` (no backfill).
- [ ] Métricas Prometheus:
  - `catalog_resolve_requests_total{result}`
  - `catalog_backfill_runs_total{result}`
  - `catalog_backfill_duration_seconds` (histogram)
  - `catalog_items_total{type}`
- [ ] Evitar alta cardinalidade; **CPF nunca** em labels.

### 7) Cache
- [ ] Cache LRU com TTL (`CATALOG_RESOLVE_CACHE_TTL_SECONDS`); *metrics* de hit/miss (opcional).  
- [ ] Invalidação quando criar/alterar **instrument/alias/ca_event**.

### 8) Performance
- [ ] Índices citados existem e são utilizados (`EXPLAIN ANALYZE`).  
- [ ] Backfill em lotes respeita `batchSize`; sem *seq scans* desnecessários.  
- [ ] *Keyset pagination* (se aplicável) no histórico.

### 9) Segurança & LGPD
- [ ] **`X-Tenant-Id` obrigatório** em tudo que toca dados de cliente/ledger.  
- [ ] Rotas `Admin` com RBAC **admin** apenas; rotas públicas podem ser restritas a suporte/admin inicialmente.  
- [ ] Logs **sem PII sensível**; mascarar quando necessário.

### 10) Testes Automatizados
**Integration — `src/Tests/Integration/Catalog/canonicalization_test.go`**
- [ ] `resolve` retorna **ALIAS/SELF/CA_EVENT/UNKNOWN** conforme cada cenário e data.  
- [ ] Backfill popula `canonical_ticker` corretamente (e é idempotente).  
- [ ] Multi‑tenant: resolução/backfill restritos ao `tenant_id` do header.

**Unit**
- [ ] Prioridade de resolução e filtros de período; cache LRU; fallback `UNKNOWN`.

### 11) Documentação
- [ ] Swagger e coleções Postman/Bruno alinhadas e versionadas.  
- [ ] `docs/catalog/canonicalization.md` descreve manutenção do catálogo e interação com 1.24.

---

## 🧪 Passos de Validação Manual (rápidos)

1) **Criar instrumento e alias**  
```bash
curl -X POST "$BASE_URL/admin/catalog/instruments" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"canonicalTicker":"ITSA4","assetType":"EQUITY","status":"ACTIVE"}'

curl -X POST "$BASE_URL/admin/catalog/aliases" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"instrumentId":"<uuid>","aliasTicker":"ITSA3","validFrom":"2015-01-01"}'
```

2) **Resolver**  
```bash
curl "$BASE_URL/catalog/tickers/resolve?symbol=ITSA3&date=2020-06-01" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```
**Esperado**: `ALIAS` → `canonicalTicker="ITSA4"`.

3) **Evento de identidade e histórico**  
```bash
curl -X POST "$BASE_URL/admin/catalog/ca-events" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"eventType":"TICKER_MOVE","effectiveDate":"2021-01-01","fromTicker":"XPTO3","toTicker":"XPTO4"}'

curl "$BASE_URL/catalog/ca/history?ticker=XPTO3" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

4) **Backfill (dry‑run e execução)**  
```bash
curl -X POST "$BASE_URL/admin/catalog/backfill/ledger-canonical" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","dryRun":true,"batchSize":2000}'

curl -X POST "$BASE_URL/admin/catalog/backfill/ledger-canonical" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","dryRun":false,"batchSize":2000}'
```

5) **Idempotência**  
Repetir o backfill **não** deve alterar linhas já corretas.

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.23 (Corporate Actions — Canonicalization) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
