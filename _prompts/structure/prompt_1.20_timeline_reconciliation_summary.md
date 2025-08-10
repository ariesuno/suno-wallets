# Prompt 1.20 — Timeline & Reconciliation Summary — **suno-wallets**

Construa **APIs de timeline e resumo de reconciliação** para inspecionar, auditar e operar a carteira do cliente **antes dos cálculos da Fase 2**. A timeline consolida **todas as operações** (B3_RAW, SYSTEM_SYNTHETIC e USER_MANUAL) do **ledger unificado**, com filtros, paginação estável e export. O resumo apresenta **status da reconciliação**, contagens por tipo e *health* do fluxo.

> Nomes de arquivos/classes/funções em **inglês**; comentários no código em **pt‑BR**. **Sem dados fake**. Multi‑tenant estrito. Observabilidade obrigatória.


---

## 🎯 Objetivos
1) **Timeline unificada** de operações por CPF/Ticker, ordenada e paginável, com camadas de origem visíveis.  
2) **Resumo de reconciliação** por CPF: inconsistências (1.17), system ops geradas (1.18), overrides manuais (1.19), pendências e últimas execuções.  
3) **Export** administrável (CSV/JSON) e **keyset pagination** para alto volume.  
4) Segurança (RBAC), auditoria, métricas e testes.

---

## 🧱 Arquitetura (DDD)
Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Application/Ops/timeline_query_service.go`  
  Monta consultas paginadas e aplica filtros; agrega campos de origem e CA (quando disponíveis).

- `src/Application/Ops/reconciliation_summary_service.go`  
  Calcula o snapshot de reconciliação (inconsistências, ops de sistema, overrides, últimas janelas de B3).

- `src/Infrastructure/Ops/timeline_repository.go`  
  Consultas set‑based no `b3_operations_ledger` + *views* auxiliares; estratégia de **keyset pagination**.

- `src/Infrastructure/Ops/reconciliation_repository.go`  
  Consolida contagens a partir de `b3_inconsistencies`, `b3_operations_ledger` e checkpoints (1.11/1.14).

- `src/Api/Controllers/Ops/timeline_controller.go`  
  Endpoints REST de timeline e export.

- `src/Api/Controllers/Ops/reconciliation_controller.go`  
  Endpoints REST de resumo e detalhes.

Pastas de testes:
- `src/Tests/Unit/Ops/...`
- `src/Tests/Integration/Ops/...`

---

## 🗃️ Banco de Dados (views/índices)

Reutilize **`b3_operations_ledger`** (1.18) e **`b3_inconsistencies`** (1.17).  
Crie *views* auxiliares **opcionais** para simplificar queries de leitura:

```sql
-- View simplificada para leitura da timeline (mantém original + canônico)
CREATE OR REPLACE VIEW v_ops_timeline AS
SELECT
  id,
  tenant_id,
  cpf,
  ticker AS original_ticker,
  COALESCE(canonical_ticker, ticker) AS canonical_ticker,  -- se já houver coluna/camada de canonicalização
  asset_type,
  operation_date,
  operation_type,
  source,
  quantity,
  unit_price,
  currency,
  reason_code,
  price_confidence,
  generated_by_inconsistency_id,
  supersedes_operation_id,
  superseded_by_operation_id,
  is_active,
  created_at,
  updated_at,
  ca_id,                  -- se presente do 1.24
  ca_type,
  ca_version_applied
FROM b3_operations_ledger
WHERE is_active = TRUE;
```

**Índices recomendados** (crie se ausentes):
- `b3_operations_ledger(tenant_id, cpf, canonical_ticker, operation_date DESC, id DESC)`  
- `b3_operations_ledger(tenant_id, cpf, source, is_active)`  
- `b3_inconsistencies(tenant_id, cpf, status, type)`

---

## 📡 Endpoints (Swagger tags: `Timeline`, `Reconciliation`)

### 1) `GET /ops/timeline`
**Query params**:  
`cpf` (obrigatório), `tickers?` (lista), `from?`, `to?`, `sources?` (lista: `B3_RAW,SYSTEM_SYNTHETIC,USER_MANUAL`), `assetTypes?`, `canonicalize?` (bool, default: true), `pageSize?` (default: 100, max: 1000), `cursor?` (keyset).

**Ordem**: `operation_date DESC, id DESC` (keyset).

**Resposta (200)**:
```json
{
  "items": [
    {
      "id": "uuid",
      "canonicalTicker": "ITSA4",
      "originalTicker": "ITSA4",
      "assetType": "EQUITY",
      "operationDate": "2024-06-10",
      "operationType": "BUY",
      "source": "B3_RAW|SYSTEM_SYNTHETIC|USER_MANUAL",
      "quantity": "100.0000000000",
      "unitPrice": "10.00",
      "currency": "BRL",
      "reasonCode": "OPENING_BALANCE_PRE_API|ZERO_PNL_PRE_API|...",
      "priceConfidence": "MARKET_CLOSE|MIRRORED_SELL|DERIVED_POSITION|UNKNOWN",
      "links": {
        "inconsistencyId": "uuid|null",
        "supersedesOperationId": "uuid|null",
        "supersededByOperationId": "uuid|null",
        "caId": "uuid|null"
      },
      "meta": {
        "createdAt": "ISO-8601",
        "updatedAt": "ISO-8601",
        "caVersionApplied": 3
      }
    }
  ],
  "nextCursor": "opaque-key-or-null",
  "count": 100
}
```

**Erros**: 400 (param inválido), 401/403, 404 (cpf sem dados), 429/5xx.  
**Segurança**: `X-Tenant-Id` obrigatório; RBAC admin/suporte (e futuramente o próprio usuário).  
**LGPD**: CPF **não** retorna em payload; logs mascaram CPF.

---

### 2) `GET /ops/timeline/export`
**Query**: mesmos filtros do `/ops/timeline`, + `format=csv|json` (default: `csv`), `limit` (cap ex.: 50k).  
**Resposta**: `Content-Disposition: attachment` com arquivo.  
**Controles**: rate‑limit e auditoria de export.

---

### 3) `GET /ops/reconciliation/summary`
**Query**: `cpf` (obrigatório).

**Resposta (200)**:
```json
{
  "cpfMasked": "***1234",
  "period": { "from": "2019-10-01", "to": "2025-08-01" },
  "dataSourceMode": "B3_ONLY|MANUAL_ONLY|HYBRID",
  "b3": {
    "lastFullFetchAt": "ISO-8601|null",
    "lastIncrementalAt": "ISO-8601|null",
    "latestWindow": { "from": "YYYY-MM-DD", "to": "YYYY-MM-DD" }
  },
  "inconsistencies": {
    "open": { "total": 3, "byType": { "OPENING_BALANCE_MISSING": 2, "SELL_WITHOUT_BUY": 1 } },
    "resolved": 10,
    "overridden": 4,
    "lastScanAt": "ISO-8601|null"
  },
  "systemOps": {
    "created": 8,
    "byReason": { "OPENING_BALANCE_PRE_API": 5, "ZERO_PNL_PRE_API": 3 },
    "lastAutoFixAt": "ISO-8601|null"
  },
  "userOverrides": {
    "manualOps": 6,
    "dedup": { "autoMerged": 2, "overridden": 3, "ignored": 4 },
    "lastActionAt": "ISO-8601|null"
  },
  "tickers": [
    {
      "ticker": "ITSA4",
      "assetType": "EQUITY",
      "operations": { "B3_RAW": 12, "SYSTEM_SYNTHETIC": 1, "USER_MANUAL": 0 },
      "inconsistenciesOpen": 0,
      "lastOperationAt": "ISO-8601"
    }
  ],
  "notes": [
    "Sem divergências posição × transações no período amostrado.",
    "Existem 2 inconsistências aguardando preço (PENDING_PRICE)."
  ]
}
```

**Erros**: 400, 401/403, 404, 5xx.  
**Observação**: valores são contagens/tempos derivados das tabelas base e checkpoints (1.11/1.14).

---

## 🧠 Implementação (pontos‑chave)

- **Keyset pagination**: use `cursor = base64(operation_date|id)`; queries com `WHERE (operation_date, id) < (:date, :id)` para ordem DESC.  
- **Canonicalização de ticker**: se disponível (1.23/1.24), inclua `canonical_ticker`; senão retorne `original_ticker` = `ticker`.  
- **Filtros**: múltiplos tickers, `sources`, `assetTypes`, janela `from/to` (inclusive).  
- **Export**: fluxo *streaming* (CSV) para grandes volumes; cap e RBAC admin.  
- **Recon Summary**: consultas agregadas (CTEs) para **inconsistências**, **ops de sistema**, **dedup decisions** e **últimos marcos B3**.  
- **Sem side-effects**: somente leitura.

---

## 📊 Observabilidade
- **Logs** (JSON) com: `tenantId`, `cpfMasked`, `filters`, `items`, `durationMs`, `sourceMix`, `nextCursor`.  
- **Métricas Prometheus**:
  - `ops_timeline_requests_total{result}`
  - `ops_timeline_duration_seconds` (histogram)
  - `ops_timeline_export_requests_total{format}`
  - `recon_summary_requests_total{result}`
  - `recon_summary_duration_seconds` (histogram)

Evitar alta cardinalidade; CPF apenas nos logs (mascarado).

---

## 🔐 Segurança & Multi‑tenant
- Header **`X-Tenant-Id` obrigatório**.  
- RBAC: `timeline` leitura; `export` apenas **admin**; `summary` leitura para admin/suporte.  
- **Rate‑limit** em export.  
- **Sem** tokens/segredos em logs; **mascarar** CPF.

---

## 🚦 Performance
- **Índices** citados ativos; **EXPLAIN ANALYZE** em queries de timeline e resumo.  
- Keyset por padrão; `offset/limit` apenas para pequenos volumes (parâmetro `useOffset=false` por default).  
- Responses com paginação consistente e campos mínimos necessários.

---

## 🧪 Testes (obrigatório)

**Integration — `src/Tests/Integration/Ops/timeline_and_summary_test.go`**
- Timeline mistura B3_RAW + SYSTEM_SYNTHETIC + USER_MANUAL corretamente, ordenado por data/id.  
- Filtros por `tickers`, `sources`, `assetTypes`, `from/to` funcionam; `canonicalize=true` respeitado.  
- `nextCursor` válido; repetição retorna a próxima página sem duplicação.  
- Export CSV/JSON respeita filtros e limites; audita a operação.  
- Summary retorna contagens consistentes com o ledger e inconsistências para o CPF.

**Unit**
- Serialização do cursor (encode/decode), validação de filtros, agregadores do resumo.

---

## 📄 Documentação
- **Swagger** (tags `Timeline` e `Reconciliation`) com exemplos realistas.  
- **Postman/Bruno** em `/docs/ops/` com rotas prontas (timeline, export, summary).  
- `docs/ops/timeline_summary.md` com dicas de uso, paginação e troubleshooting.

---

## 🔧 Configurações (.env)
- `TIMELINE_DEFAULT_PAGE_SIZE=100`  
- `TIMELINE_MAX_PAGE_SIZE=1000`  
- `TIMELINE_EXPORT_MAX_ROWS=50000`  
- `OBS_MASK_CPF=true`

---

## ✅ Entregáveis
- Services + Repositories + Controllers + views/índices auxiliares (se necessário).  
- Testes unitários e de integração.  
- Logs, métricas, RBAC, Swagger e collections.  
- **Aceite**: chamadas de timeline e summary retornam dados consistentes, com **keyset pagination**, filtros funcionais, export seguro e agregados coerentes com o estado do ledger e das inconsistências.
