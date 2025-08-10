# Prompt 1.23 — Corporate Actions Canonicalization (Ticker mapping & histórico) — **suno-wallets**

Implemente a **camada de canonicalização de ativos** (mapeamento de tickers e histórico de eventos corporativos **sem** aplicar ajustes na quantidade/preço).  
Este módulo fornece **identidade canônica** e **histórico de mudanças** (rename, incorporação, migração de ticker) para ser usado por leitura (timeline/summary), dedup e backoffice.  
**Não** executar projeções/ajustes de frações/quantidades aqui — isso será feito no **1.24 (CA Projection & Fraction Rounding)**.

> Nomes de arquivos/classes/funções em **inglês**; comentários no código em **pt‑BR**. **Sem dados fake**. Multi‑tenant estrito. Observabilidade e testes obrigatórios.

---

## 🎯 Objetivos
1) Persistir **catálogo canônico** de instrumentos e **aliases** (tickers históricos/sinônimos) por **tenant**.  
2) Registrar **Corporate Action Events** relevantes para **identidade** (rename/merge/spin‑off/ticker-move), **sem** alterar ledger.  
3) Expor **serviços/endpoint** para: **resolver ticker → canonical**, listar **histórico** e **backfill canônico** nas operações do ledger (campo `canonical_ticker`).  
4) Garantir **desempenho** para consultas de timeline (1.20) e backoffice (1.22).

---

## 🧱 Arquitetura (DDD)
Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Domain/Catalog/instrument.go`  
  Entidades: `Instrument` (canônico), `TickerAlias`, `CorporateActionEvent`. Regras de validade.

- `src/Application/Catalog/canonicalization_service.go`  
  Resolve/normaliza ticker para `canonical_ticker`; backfill no ledger; APIs de consulta.

- `src/Infrastructure/Catalog/catalog_repository.go`  
  CRUD e consultas (por tenant); leitura do histórico; operações de backfill com lotes.

- `src/Api/Controllers/Catalog/catalog_controller.go`  
  Rotas públicas (read‑only) e administrativas (CRUD + backfill).

Pastas de testes:
- `src/Tests/Unit/Catalog/...`
- `src/Tests/Integration/Catalog/...`

---

## 🗃️ Banco de Dados (migrations)

### a) Catálogo Canônico
```sql
CREATE TABLE IF NOT EXISTS instruments (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  canonical_ticker TEXT NOT NULL,
  asset_type TEXT NOT NULL,             -- ex.: EQUITY, FII, ETF, BDR...
  isin TEXT,                            -- opcional se disponível
  status TEXT NOT NULL DEFAULT 'ACTIVE',-- ACTIVE, INACTIVE, DELISTED
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, canonical_ticker)
);

CREATE TABLE IF NOT EXISTS ticker_aliases (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  instrument_id UUID NOT NULL,
  alias_ticker TEXT NOT NULL,           -- ticker histórico ou sinônimo
  valid_from DATE,                      -- datas efetivas são opcionais
  valid_to DATE,
  source TEXT,                          -- origem do cadastro (ADMIN, IMPORT, B3_REF, etc.)
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, alias_ticker)
);
```

### b) Corporate Actions (identidade)
```sql
CREATE TYPE ca_identity_type AS ENUM ('RENAME','MERGE','SPIN_OFF','TICKER_MOVE');

CREATE TABLE IF NOT EXISTS corporate_action_events (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  event_type ca_identity_type NOT NULL,
  effective_date DATE NOT NULL,
  -- participantes / resultado
  from_ticker TEXT,             -- pode ser NULL em TICKER_MOVE destino
  to_ticker TEXT,               -- pode ser NULL em SPIN_OFF múltiplos
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_ca_events_lookup
  ON corporate_action_events (tenant_id, effective_date DESC, event_type);
```

### c) Backfill no Ledger (coluna)
Se ainda não existir, **adicionar** coluna `canonical_ticker` no ledger:
```sql
ALTER TABLE b3_operations_ledger
  ADD COLUMN IF NOT EXISTS canonical_ticker TEXT;
CREATE INDEX IF NOT EXISTS ix_ops_canonical_lookup
  ON b3_operations_ledger (tenant_id, cpf, canonical_ticker, operation_date DESC, id DESC);
```

> Observação: **não** altere `ticker` original no ledger; `canonical_ticker` é um **campo paralelo** para leitura/relato.

---

## 🧠 Regras de Canonicalização

- `ResolveTicker(tenant, rawTicker, onDate?) -> canonicalTicker`  
  1) Se `alias_ticker == rawTicker` e `valid_from/valid_to` compatível com `onDate` → retorna `canonical_ticker` do `instrument`.  
  2) Se não achar alias, procurar `canonical_ticker == rawTicker` em `instruments`.  
  3) Se ainda não encontrado e houver `corporate_action_events` que indiquem **RENAME/TICKER_MOVE**, usar mapeamento por data.  
  4) Caso não resolva, **retornar o próprio `rawTicker`** e `resolution="UNKNOWN"` (não falhar).

- **Backfill**:  
  Popular `canonical_ticker` em `b3_operations_ledger` **em lote**, por `(tenant, cpf, ticker)` usando `ResolveTicker` com `operation_date`.  
  Idempotente e retomável (processar por janelas).

- **Sem projeções**: não ajustar `quantity/price` nem criar operações aqui (isso no 1.24).

---

## 📡 Endpoints (Swagger tags: `Catalog`, `Catalog Admin`)

### Público (read‑only)
- `GET /catalog/tickers/resolve?symbol=&date=`  
  Retorna `canonicalTicker`, `resolution` (`ALIAS|SELF|CA_EVENT|UNKNOWN`), `instrumentMeta`.

- `GET /catalog/ca/history?ticker=`  
  Lista eventos de identidade envolvendo o ticker (como `from` ou `to`).

### Admin
- `POST /admin/catalog/instruments`  
  Cria/atualiza `Instrument` (canônico).

- `POST /admin/catalog/aliases`  
  Cria **alias** (`aliasTicker`, `instrumentId`, `validFrom?`, `validTo?`, `source?`).

- `POST /admin/catalog/ca-events`  
  Registra **CA** de identidade (`eventType`, `effectiveDate`, `fromTicker?`, `toTicker?`, `notes?`).

- `POST /admin/catalog/backfill/ledger-canonical`  
  Body:
  ```json
  {
    "cpf": "00000000000",
    "tickers": null,
    "from": null,
    "to": null,
    "batchSize": 5000,
    "dryRun": false
  }
  ```
  Executa backfill de `canonical_ticker` no ledger para o CPF (ou escopo informado).

**Segurança**: `X-Tenant-Id` obrigatório; público read‑only pode ser restrito a suporte/admin inicialmente.  
**LGPD**: payloads públicos **não** expõem CPF; mascarar em logs.

---

## 📊 Observabilidade
- **Logs** (JSON): `tenantId`, `action`, `symbol`, `resolution`, `durationMs`, `affectedRows`.  
- **Métricas Prometheus**:
  - `catalog_resolve_requests_total{result}`
  - `catalog_backfill_runs_total{result}`
  - `catalog_backfill_duration_seconds` (histogram)
  - `catalog_items_total{type="instrument|alias|ca_event"}`

Evitar alta cardinalidade; símbolos em métricas são aceitáveis (limitados).

---

## 🚦 Performance
- Índices: `instruments(tenant_id, canonical_ticker)` e `ticker_aliases(tenant_id, alias_ticker)` garantidos.  
- **Cache local** LRU para `ResolveTicker` (chave: `tenant|symbol|dateBucket`).  
- Backfill em **lotes** com `batchSize` configurável, janelas por `cpf`/`date`.  
- `EXPLAIN ANALYZE` nas queries críticas.

---

## 🧪 Testes (obrigatório)

**Integration — `src/Tests/Integration/Catalog/canonicalization_test.go`**
- `resolve` retorna **ALIAS** quando `alias_ticker` mapeia para canônico no período.  
- **SELF** quando já é canônico; **CA_EVENT** quando depender de evento; **UNKNOWN** caso sem mapeamento.  
- Backfill popula `canonical_ticker` em lote e é idempotente (reexecutar não duplica nem altera sem razão).

**Unit**
- Regras de prioridade (alias > self > ca_event > unknown); cache LRU; filtros de data.

---

## 📄 Documentação
- Swagger (tags `Catalog`, `Catalog Admin`) com exemplos sanitizados.  
- Postman/Bruno em `/docs/catalog/` (resolve, history, CRUD, backfill).  
- `docs/catalog/canonicalization.md` explicando como manter o catálogo e impactos no 1.24.

---

## 🔧 Configurações (.env)
- `CATALOG_RESOLVE_CACHE_TTL_SECONDS=300`  
- `CATALOG_BACKFILL_BATCH_SIZE=5000`  
- `OBS_MASK_CPF=true`

---

## ✅ Entregáveis
- Domain + Service + Repository + Controllers + migrations.  
- Testes unitários e de integração.  
- Logs, métricas, Swagger e collections.  
- **Aceite**: `GET /catalog/tickers/resolve` retorna canônico consistente; `backfill/ledger-canonical` atualiza `canonical_ticker` de um CPF sem efeitos colaterais, com execução idempotente e observável.
