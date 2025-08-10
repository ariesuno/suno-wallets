# Prompt 1.19 — Manual Ops + Dedup & Alertas — **suno-wallets**

Implemente **operações manuais do cliente** no **ledger unificado** e um **mecanismo de deduplicação** entre **B3_RAW × USER_MANUAL** com **alertas e resolução segura**.  
O objetivo é suportar **MANUAL_ONLY**, **B3_ONLY** e **HYBRID**, evitando **duplicidades** e permitindo **override consciente** pelo usuário/operador.

> Nomes de arquivos/classes/funções em **inglês**; comentários no código em **pt‑BR**. **Sem dados fake**. Se faltar insumo, **abortar** e informar claramente.

## 🎯 Objetivos
1) CRUD seguro de **USER_MANUAL** em `b3_operations_ledger`.  
2) **Dedup** B3×Manual com **heurística escorável**, **idempotência** e **explicabilidade**.  
3) **Alertas** e **fluxos de resolução**: *merge*, *override* (manual prevalece) ou *ignore*.  
4) Respeitar a **política por cliente** (1.21): `B3_ONLY | MANUAL_ONLY | HYBRID`.  
5) Observabilidade completa, performance e testes (unit/integration).

---

## 🧱 Arquitetura (DDD)
Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Application/Ops/manual_operations_service.go`  
  CRUD e validações de USER_MANUAL; aplica política do cliente; chama dedup on‑write (opcional).

- `src/Application/Ops/dedup_service.go`  
  Heurística, scoring, geração/atualização de candidatos, e ações (*merge/override/ignore*).

- `src/Infrastructure/Ops/operations_repository.go`  
  Acesso ao `b3_operations_ledger` (USER_MANUAL/B3_RAW/System) e *soft delete* (`is_active=false`).

- `src/Infrastructure/Ops/dedup_repository.go`  
  Persistência de candidatos e decisões; *upsert* por `dedupe_key` e `pair_key`.

- `src/Api/Controllers/Ops/manual_operations_controller.go`  
  Rotas CRUD para USER_MANUAL.

- `src/Api/Controllers/Ops/dedup_controller.go`  
  Rotas de scan, listagem e resolução.

Pastas de testes:
- `src/Tests/Unit/Ops/...`
- `src/Tests/Integration/Ops/...`

---

## 🗃️ Banco de Dados (migrations)

### a) Ledger (reuso do 1.18)
- Reutilize `b3_operations_ledger` com colunas e índices já criados (1.18).  
- **USER_MANUAL** deve ser criado **apenas** via API; **nunca** alterar B3_RAW.

### b) Candidatos de dedup
```sql
CREATE TYPE dedup_status AS ENUM ('OPEN','AUTO_MERGED','OVERRIDDEN','IGNORED','CONFIRMED_MERGE');

CREATE TABLE IF NOT EXISTS dedup_candidates (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  cpf VARCHAR(11) NOT NULL,
  primary_operation_id UUID NOT NULL,   -- geralmente a operação B3_RAW
  candidate_operation_id UUID NOT NULL, -- geralmente a operação USER_MANUAL
  score NUMERIC(6,3) NOT NULL,          -- 0..1
  status dedup_status NOT NULL DEFAULT 'OPEN',
  rationale JSONB NOT NULL,             -- explicabilidade: features, pesos, tolerâncias usadas
  pair_key TEXT NOT NULL,               -- chave natural do par (para idempotência de geração)
  dedupe_key TEXT NOT NULL,             -- hash estável do conteúdo relevante
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT,
  resolved_at TIMESTAMPTZ,
  resolved_by TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_dedup_pair
  ON dedup_candidates(tenant_id, cpf, primary_operation_id, candidate_operation_id);

CREATE INDEX IF NOT EXISTS ix_dedup_lookup
  ON dedup_candidates(tenant_id, cpf, status, score);
```

### c) Política de preferência (opcional, por tenant/cliente)
```sql
CREATE TABLE IF NOT EXISTS dedup_policies (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  cpf VARCHAR(11) NOT NULL,
  prefer_source TEXT NOT NULL DEFAULT 'B3_RAW', -- ou 'USER_MANUAL'
  auto_merge_threshold NUMERIC(6,3) NOT NULL DEFAULT 0.92,
  alert_threshold NUMERIC(6,3) NOT NULL DEFAULT 0.70,
  date_tolerance_days SMALLINT NOT NULL DEFAULT 2,
  quantity_tolerance_ratio NUMERIC(6,4) NOT NULL DEFAULT 0.005, -- 0.5%
  gross_tolerance_ratio NUMERIC(6,4) NOT NULL DEFAULT 0.005,    -- 0.5%
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by TEXT,
  UNIQUE (tenant_id, cpf)
);
```

---

## 🤖 Heurística de Dedup (scoring)
- **Pré‑filtros (mesma classe e ticker canônico)**: `asset_type` igual, `canonical_ticker` igual.  
- **Janela de data**: `|date(manual) - date(b3)| ≤ date_tolerance_days` (úteis quando possível).  
- **Sinais**:
  - `quantity_match` → 1.0 se `|Δq|/q ≤ quantity_tolerance_ratio`; senão decai com função linear.  
  - `gross_match` → 1.0 se `|Δ(gross)|/gross ≤ gross_tolerance_ratio`.  
  - `side_match` → 1.0 se `operation_type` igual (`BUY`/`SELL`).  
  - `broker_hint` → +0.05 se corretora/canal coincidir (se existir).  
  - `order_id_hint` → +0.1 se existir `order_id` igual (se existir).  
- **Score final**: média ponderada; pesos default: `quantity 0.45`, `gross 0.35`, `side 0.15`, `extras 0.05`.  
- **Decisão**:  
  - `score ≥ auto_merge_threshold` → **auto‑merge**.  
  - `alert_threshold ≤ score < auto_merge_threshold` → **alerta pendente** (revisão humana).  
  - `score < alert_threshold` → ignorar (sem candidato).

**Explicabilidade**: salvar as *features*, pesos e tolerâncias em `rationale`.

**Idempotência**: `pair_key = hash(primary_id|candidate_id)`; `dedupe_key = hash(campos_relevantes)`.

---

## 📡 Endpoints (Swagger tags: `Manual Ops`, `Ops Dedup`)

### Manual Ops (USER_MANUAL)
- `POST /ops/manual` — cria operação manual (valida política do cliente).  
  Body mínimo: `{ cpf, ticker, assetType, operationDate, operationType, quantity, unitPrice?, currency? }`
- `PUT /ops/manual/{id}` — atualiza **apenas** USER_MANUAL **ativa** (não permite alterar `source`).  
- `DELETE /ops/manual/{id}` — **soft delete** (`is_active=false`).  
- `GET /ops/manual` — lista paginada/filtrada por `cpf`, `ticker`, `dateRange`, `type`, `isActive`.

### Dedup
- `POST /ops/dedup/scan` *(admin‑only)* — gera candidatos (opcional `dryRun`).  
  Body: `{ cpf, since?, to?, tickers?, scanMode: "ON_WRITE|BATCH", dryRun?: false }`
- `GET /ops/dedup/candidates` — lista candidatos por `status`, `scoreRange`, `ticker`, `cpf`.  
- `POST /ops/dedup/resolve` — aplica ação em lote:  
  Body:
  ```json
  {
    "action": "MERGE|OVERRIDE|IGNORE",
    "candidateIds": ["..."],
    "prefer": "B3_RAW|USER_MANUAL"  // opcional, para forçar preferência no momento
  }
  ```

**Semântica de resolução**:
- **MERGE**: mantém **uma** operação **ativa** conforme preferência (padrão: `prefer_source` da policy ou `B3_RAW`); a outra recebe `is_active=false` e `superseded_by_operation_id`.  
- **OVERRIDE**: força prevalência do **USER_MANUAL** (marca `is_active=false` na B3_RAW **no ledger**; não toca no RAW da B3).  
- **IGNORE**: mantém ambas ativas e marca o candidato como `IGNORED` (não gerar novamente).

**Segurança**: `X-Tenant-Id` obrigatório; RBAC (admin para scan/resolve, user para CRUD próprio se aplicável).  
**LGPD**: mascarar CPF nos logs; não logar tokens/segredos.

---

## 🔐 Política por Cliente (integração 1.21)
- Respeitar `client_data_source_policy.mode` (`B3_ONLY | MANUAL_ONLY | HYBRID`):  
  - `B3_ONLY` → **bloquear** criação de USER_MANUAL (retornar 409 com hint).  
  - `MANUAL_ONLY` → **não importar** B3_RAW no ledger (ou desativar leitura para cálculo).  
  - `HYBRID` → permitir ambos + dedup.

---

## 📊 Observabilidade
- **Logs** (JSON): `tenantId`, `cpfMasked`, `opId`, `action`, `score`, `decision`, `durationMs`.  
- **Métricas Prometheus**:
  - `ops_manual_writes_total{result}`
  - `ops_dedup_scan_runs_total{result}`
  - `ops_dedup_candidates_total{status}`
  - `ops_dedup_resolutions_total{action}`
  - `ops_dedup_duration_seconds` (histogram)

Evitar alta cardinalidade (CPF apenas nos **logs**).

---

## 🚦 Performance & Índices
- Índices já existentes no ledger: `(tenant_id, cpf, ticker, operation_date)`; adicionar `(tenant_id, cpf, source, is_active)`.
- Dedup por **lotes** (janelas mensais) e *set‑based* quando possível.
- `EXPLAIN ANALYZE` nas principais consultas; paginação consistente.

---

## 🧪 Testes (obrigatório)

**Integration** — `src/Tests/Integration/Ops/manual_and_dedup_test.go`
- CRUD USER_MANUAL: criar/editar/soft‑delete e leitura paginada.  
- `HYBRID`: criar USER_MANUAL semelhante a B3_RAW → `scan` gera candidato com `score` alto; `MERGE` inativa uma; `OVERRIDE` prevalece manual; `IGNORE` não reaparece.  
- Idempotência: reexecutar `scan` **não** duplica `dedup_candidates`; `resolve` repetido não quebra.  
- Política `B3_ONLY` bloqueia `POST /ops/manual` com 409.

**Unit**
- Cálculo de *score* (features/pesos/tolerâncias); geração de `pair_key`/`dedupe_key`.  
- Regras de resolução (MERGE/OVERRIDE/IGNORE) e auditoria.

---

## 📄 Documentação
- Swagger: tags **`Manual Ops`** e **`Ops Dedup`**, exemplos de 200/400/401/403/409/5xx.  
- Coleções Postman/Bruno em `/docs/ops/` com rotas CRUD, scan e resolve.  
- `docs/ops/dedup.md` explicando a heurística, thresholds e como interpretar `rationale`.

---

## 🔧 Configurações (.env)
- `DEDUP_AUTO_MERGE_THRESHOLD=0.92`  
- `DEDUP_ALERT_THRESHOLD=0.70`  
- `DEDUP_DATE_TOLERANCE_DAYS=2`  
- `DEDUP_QTY_TOLERANCE_RATIO=0.005`  
- `DEDUP_GROSS_TOLERANCE_RATIO=0.005`  
- `OBS_MASK_CPF=true`

---

## ✅ Entregáveis
- Services, repositories, controllers, migrations.  
- Testes unitários e de integração.  
- Logs, métricas, RBAC, Swagger e collections.  
- **Aceite**: rodar `POST /ops/dedup/scan` e `POST /ops/dedup/resolve` em ambiente com HYBRID e obter **decisões corretas**, sem duplicações, com explicabilidade e auditoria.
