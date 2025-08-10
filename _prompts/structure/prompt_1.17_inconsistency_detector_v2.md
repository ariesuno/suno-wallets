# Prompt 1.17 — Inconsistency Detector (read-only) — **suno-wallets**

Varre **RAW/Normalized** e identifica casos como **“posição sem compra”** e **“venda sem compra”**. Registra em **`b3_inconsistencies`** com **tipo, evidências e status**. *Não* altera ledger, *não* cria operações — apenas detecção (a reconciliação automática acontece no 1.18).

> Nomes de arquivos/classes/funções em **inglês**; comentários em **pt‑BR**; sem dados fake. Se faltar insumo, **abortar** e **solicitar instruções**.

---

## 🎯 Objetivos
1) Detectar inconsistências por **CPF/Ticker** (pré‑API e divergências RAW/Normalized).  
2) Cobrir tipos:  
   - **OPENING_BALANCE_MISSING** — posição existe mas não há compras suficientes anteriores.  
   - **SELL_WITHOUT_BUY** — primeira transação é venda ou cumulativo negativo.  
   - **POSITION_TX_DIVERGENCE** — foto de posição v3 difere do acumulado líquido de transações.  
3) Persistir achados em `b3_inconsistencies` com **status**, **severidade**, **amostras**, **detalhes** e **idempotência** por `dedupe_hash`.  
4) Expor endpoints **admin/read** para **scan**, **listagem** e **detalhe**.  
5) Garantir **observabilidade**, **multi‑tenant**, **segurança** e **testes**.

---

## 🧱 Arquitetura (DDD)
Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Application/B3/reconciliation/inconsistency_detector_service.go`  
  Orquestra o escaneamento por janela/CPF/Ticker, aplica regras e monta achados.

- `src/Infrastructure/B3/reconciliation/inconsistency_repository.go`  
  Queries set‑based para obter evidências e persistir achados; **upsert** por `dedupe_hash`.

- `src/Api/Controllers/B3/reconciliation_controller.go`  
  Endpoints: `POST /reconciliation/scan` (admin), `GET /reconciliation/inconsistencies`, `GET /reconciliation/inconsistencies/{id}`.

Pastas de testes:
- `src/Tests/Unit/B3/reconciliation/...`
- `src/Tests/Integration/B3/reconciliation/...`

---

## 🗃️ Banco de Dados (migrations)
```sql
CREATE TYPE inconsistency_type AS ENUM (
  'OPENING_BALANCE_MISSING',
  'SELL_WITHOUT_BUY',
  'POSITION_TX_DIVERGENCE',
  'OTHER'
);

CREATE TYPE inconsistency_status AS ENUM ('OPEN','RESOLVED','OVERRIDDEN');

CREATE TABLE b3_inconsistencies (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  cpf VARCHAR(11) NOT NULL,
  ticker TEXT NOT NULL,
  type inconsistency_type NOT NULL,
  status inconsistency_status NOT NULL DEFAULT 'OPEN',
  severity SMALLINT NOT NULL DEFAULT 2, -- 1=low,2=medium,3=high
  affected_period_start DATE,
  affected_period_end DATE,
  first_detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_detected_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  sample_dates JSONB,             -- ex.: {"first_sell":"2020-01-10","first_position":"2020-01-02"}
  details JSONB,                  -- evidências, diferenças, contagens
  dedupe_hash TEXT NOT NULL,      -- hash(cpf,ticker,type,period,signature)
  created_by_version TEXT,
  created_by TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_b3_inconsistencies_dedupe
  ON b3_inconsistencies(tenant_id, cpf, ticker, type, dedupe_hash);

CREATE INDEX ix_b3_inconsistencies_lookup
  ON b3_inconsistencies(tenant_id, cpf, status, type);
```

**Observações**
- Não tocar em RAW/Normalized; só leitura + gravação em `b3_inconsistencies`.
- `dedupe_hash` garante idempotência do escaneamento.

---

## 🔎 Regras de detecção
Use `B3_API_EARLIEST_DATE` (ex.: `2019-10-01`, configurável). Sempre filtrar por `tenant_id` e `cpf`.

1) **OPENING_BALANCE_MISSING**  
   - `first_position_date = MIN(reference_date)` em `b3_normalized_positions`  
   - `position_qty_at_first_date`  
   - `cum_net_qty_from_tx_since_earliest` = Σ(buys - sells) de `B3_API_EARLIEST_DATE` até `first_position_date`  
   - Se `position_qty_at_first_date > cum_net_qty_from_tx_since_earliest` → provável saldo pré‑API.

2) **SELL_WITHOUT_BUY**  
   - `first_tx_side = SELL` **ou** `min(cumulative_net_qty) < 0` ao ordenar transações por `trade_date`.

3) **POSITION_TX_DIVERGENCE**  
   - Para cada `reference_date` disponível, comparar:  
     `position_qty(reference_date)` **vs** `Σ(buys - sells)` do início até a data.  
   - Registrar top‑N datas divergentes (amostragem).

**Details JSON** — exemplos:
```json
{ "first_position_date":"YYYY-MM-DD","position_qty":"100","net_tx_until_first_position":"0" }
{ "first_tx_date":"YYYY-MM-DD","first_tx_side":"SELL","min_cum_qty":"-100" }
{ "samples":[{"date":"YYYY-MM-DD","pos_qty":"150","net_tx_qty":"120"}], "sample_count":1 }
```

---

## 📡 Endpoints (Swagger tag: `B3 Reconciliation`)
1) `POST /reconciliation/scan` *(admin‑only)*  
Body:
```json
{
  "cpf": "00000000000",
  "tickers": null,
  "from": null,
  "to": null,
  "dryRun": false,
  "maxSamplesPerType": 5,
  "concurrency": 2
}
```
Semântica: `dryRun=true` só simula; `dryRun=false` persiste (upsert).

2) `GET /reconciliation/inconsistencies` — filtros `cpf`, `status?`, `type?`, `ticker?`, `from?`, `to?`, `page?`, `pageSize?`.

3) `GET /reconciliation/inconsistencies/{id}` — retorna registro completo.

**Segurança**: `X-Tenant-Id` obrigatório; RBAC admin para `POST`; leitura para admin/suporte.  
**LGPD**: mascarar CPF nos logs; nunca logar tokens/segredos.

---

## 📊 Observabilidade
- **Logs** (JSON): `tenantId`, `cpfMasked`, `tickers`, `from`, `to`, `found_by_type`, `durationMs`.  
- **Métricas** (Prometheus):  
  - `b3_recon_scan_runs_total{result}`  
  - `b3_recon_inconsistencies_found_total{type}`  
  - `b3_recon_scan_duration_seconds` (histogram)  
- Evitar alta cardinalidade (CPF só em logs).

---

## 🚦 Performance & Índices
- Índices recomendados:
  - `b3_normalized_positions(tenant_id, cpf, ticker, reference_date)`
  - `b3_normalized_transactions(tenant_id, cpf, ticker, trade_date, side)`
- Processar por **janelas mensais** e usar CTEs/aggregates (set‑based).  
- Limitar amostras nos `details` (`maxSamplesPerType`).

---

## 🧪 Testes (obrigatório)
**Integration** — `src/Tests/Integration/B3/reconciliation/inconsistency_detector_test.go`  
- Posição sem compras → detecta `OPENING_BALANCE_MISSING`.  
- Primeira transação `SELL` → detecta `SELL_WITHOUT_BUY`.  
- Divergência posição × transações → detecta `POSITION_TX_DIVERGENCE`.  
- `dryRun=true` não persiste; `dryRun=false` persiste com **idempotência**.

**Unit** — `src/Tests/Unit/B3/reconciliation/...`  
- Cálculo de `dedupe_hash`, paginação, filtros e composição de `details`.

---

## 📄 Documentação
- **Swagger** (tag `B3 Reconciliation`) com exemplos e erros 200/400/401/403/409/5xx.  
- **Postman/Bruno** em `/docs/reconciliation/` (requests sanitizados).  
- `docs/b3/inconsistencies.md` com explicações e troubleshooting.

---

## 🔧 Configurações (.env)
- `B3_API_EARLIEST_DATE=2019-10-01` *(confirmar data oficial)*  
- `RECON_SCAN_MAX_CONCURRENCY=2`  
- `OBS_MASK_CPF=true`

---

## ✅ Entregáveis
- Service + Repository + Controller + migrations.  
- Testes unitários/integrados.  
- Logs, métricas e RBAC integrados.  
- Swagger + Postman/Bruno + docs atualizados.  
- **Aceite**: rodar `/reconciliation/scan` (dry‑run e real) com achados coerentes e persistência idempotente, **sem** alterar o ledger.
