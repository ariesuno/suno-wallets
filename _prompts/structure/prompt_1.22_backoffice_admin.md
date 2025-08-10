# Prompt 1.22 — Backoffice Admin (Painel Operacional) — **suno-wallets**

Crie o **Backoffice Admin** (somente APIs nesta fase) para **investigação operacional**, **ações pontuais** e **auditoria** da carteira do cliente, consolidando informações dos módulos 1.11/1.14 (ingest B3), 1.17 (inconsistências), 1.18 (system ops), 1.19 (manual ops + dedup), 1.20 (timeline/summary) e 1.21 (policy).

> Nomes de arquivos/classes/funções em **inglês**; comentários no código em **pt‑BR**. **Sem dados fake**. Multi‑tenant estrito, LGPD e RBAC rígidos. Observabilidade completa.  
> **Escopo é API** (REST) — UI web fica para fase posterior.


---

## 🎯 Objetivos
1) Fornecer **endpoints administrativos** para **buscar CPF**, ver **saúde** e **histórico** (ingest, reconciliação, dedup, policy), e **acionar** rotinas com **dry‑run** e **dupla confirmação**.  
2) Prover **auditoria** das ações administrativas e **export** de relatórios operacionais (CSV/JSON).  
3) Incluir **métricas** e **logs** específicos do backoffice.  
4) Testes completos (unit/integration) e documentação (Swagger/Collections).


---

## 🧱 Arquitetura (DDD)
Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Application/Admin/backoffice_query_service.go`  
  Consulta agregada de **profile operacional** do cliente (status ingest, inconsistências, ledger mix, policy, tempos médios, últimas execuções).

- `src/Application/Admin/backoffice_actions_service.go`  
  Orquestra **ações** (dry‑run/execução real) com **locks** por `(tenantId, cpf)`, **idempotência** e **dupla confirmação**.

- `src/Infrastructure/Admin/backoffice_repository.go`  
  Leitura agregada (CTEs/views) e persistência de **auditoria** e **job triggers**.

- `src/Api/Controllers/Admin/backoffice_controller.go`  
  Endpoints REST (search, profile, actions, export, audit).

- (Reuso) Services/Repos de 1.11/1.14/1.17/1.18/1.19/1.20/1.21 através de **ports**.

Pastas de testes:
- `src/Tests/Unit/Admin/...`
- `src/Tests/Integration/Admin/...`


---

## 🗃️ Banco de Dados (migrations)

### a) Auditoria de ações administrativas
```sql
CREATE TYPE admin_action_type AS ENUM (
  'B3_FULL_FETCH','B3_INCREMENTAL_FETCH','RECON_SCAN','AUTO_FIX','DEDUPE_SCAN','DEDUPE_RESOLVE',
  'POLICY_UPDATE','LEDGER_EXPORT','CACHE_INVALIDATE','CLIENT_RESET','CLIENT_ZERO_AND_REFETCH'
);

CREATE TYPE admin_action_status AS ENUM ('REQUESTED','CONFIRMED','RUNNING','SUCCESS','ERROR','SKIPPED');

CREATE TABLE IF NOT EXISTS admin_action_audit (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  cpf VARCHAR(11) NOT NULL,
  action admin_action_type NOT NULL,
  status admin_action_status NOT NULL,
  requested_by TEXT NOT NULL,
  confirmed_by TEXT,                     -- dupla confirmação (se aplicável)
  confirm_token TEXT,                    -- token curto emitido no REQUESTED
  confirm_deadline TIMESTAMPTZ,          -- TTL para confirmação
  request_payload JSONB NOT NULL,        -- parâmetros informados
  result_payload JSONB,                  -- resumo de resultados, ids gerados, contagens
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  error_message TEXT
);

CREATE INDEX IF NOT EXISTS ix_admin_action_lookup
  ON admin_action_audit (tenant_id, cpf, action, status, created_at DESC);
```

### b) Checkpoints operacionais (opcional)
Se ainda não houver métricas persistidas por CPF (p.ex., 1.11/1.14), criar tabela leve:
```sql
CREATE TABLE IF NOT EXISTS client_operational_checkpoints (
  tenant_id UUID NOT NULL,
  cpf VARCHAR(11) NOT NULL,
  last_full_fetch_at TIMESTAMPTZ,
  last_incremental_at TIMESTAMPTZ,
  avg_full_fetch_seconds NUMERIC(10,2),
  avg_incremental_seconds NUMERIC(10,2),
  last_recon_scan_at TIMESTAMPTZ,
  last_auto_fix_at TIMESTAMPTZ,
  last_dedup_scan_at TIMESTAMPTZ,
  PRIMARY KEY (tenant_id, cpf)
);
```


---

## 📡 Endpoints (Swagger tag: `Backoffice Admin`)

### 1) **Search & Profile**
- `GET /admin/backoffice/search?query=`  
  Busca por CPF (mascarado na resposta), nome (se disponível), e metadados essenciais (modo de policy, datas de última ingestão).

- `GET /admin/backoffice/profile?cpf=`  
  Retorna **visão 360** do cliente:
  ```json
  {
    "cpfMasked":"***1234",
    "dataSourceMode":"B3_ONLY|MANUAL_ONLY|HYBRID",
    "b3":{
      "lastFullFetchAt":"ISO",
      "lastIncrementalAt":"ISO",
      "latestWindow":{"from":"YYYY-MM-DD","to":"YYYY-MM-DD"}
    },
    "reconciliation":{
      "inconsistencies":{"open":3,"byType":{"OPENING_BALANCE_MISSING":2}},
      "systemOps":{"created":8,"byReason":{"OPENING_BALANCE_PRE_API":5}},
      "manualOps":6,
      "dedup":{"open":1,"autoMerged":2,"overridden":3,"ignored":4}
    },
    "performance":{"avgFullFetchSec":12.3,"avgIncrementalSec":1.9},
    "lastActions":[{"action":"B3_INCREMENTAL_FETCH","at":"ISO","status":"SUCCESS"}]
  }
  ```

### 2) **Actions (com dry‑run e dupla confirmação)**

> **Padrão**: `POST /admin/backoffice/actions/<action>` → cria **REQUESTED** e retorna `confirmToken`. Depois:  
> `POST /admin/backoffice/actions/<action>/confirm` com `confirmToken` para **CONFIRMED** e execução.  
> Logs e métricas em todas as etapas.

- `POST /admin/backoffice/actions/b3-full-fetch`  
  Body: `{ "cpf":"000...","from":"YYYY-MM","to":"YYYY-MM","dryRun":false }`  
  Gatilho do 1.11 (full histórico, por janelas mensais).

- `POST /admin/backoffice/actions/b3-incremental`  
  Body: `{ "cpf":"000...","date":"YYYY-MM-DD?","dryRun":false }`  
  Gatilho do 1.14 (janela do dia ou última janela aberta).

- `POST /admin/backoffice/actions/recon-scan`  
  Body: `{ "cpf":"000...","tickers":null,"types":["OPENING_BALANCE_MISSING","SELL_WITHOUT_BUY"],"dryRun":true }`  
  Gatilho do 1.17 (detector de inconsistências).

- `POST /admin/backoffice/actions/auto-fix`  
  Body: `{ "cpf":"000...","types":["OPENING_BALANCE_MISSING","SELL_WITHOUT_BUY"],"dryRun":false }`  
  Gatilho do 1.18.

- `POST /admin/backoffice/actions/dedup-scan`  
  Body: `{ "cpf":"000...","scanMode":"BATCH","dryRun":false }`  
  Gatilho do 1.19 (gerar candidatos).

- `POST /admin/backoffice/actions/dedup-resolve`  
  Body: `{ "action":"MERGE|OVERRIDE|IGNORE","candidateIds":["..."] }`  
  Gatilho do 1.19 (resolver).

- `POST /admin/backoffice/actions/policy-set`  
  Body: `{ "cpf":"000...","mode":"B3_ONLY|MANUAL_ONLY|HYBRID","reason":"..." }`  
  Atalho para 1.21 (com audit integrado aqui).

- `POST /admin/backoffice/actions/client-reset`  
  Body: `{ "cpf":"000...","what":"LEDGER_ONLY|ALL_NORMALIZED|ALL_RAW|ALL","dryRun":true }`  
  **Perigoso** — exige **dupla confirmação** SEMPRE. Gera auditoria e checkpoint pós‑execução.

- `POST /admin/backoffice/actions/zero-and-refetch`  
  Body: `{ "cpf":"000...","from":"2019-10-01","dryRun":false }`  
  Zera **ledger normalizado** do CPF e reexecuta 1.11 (full). **Dupla confirmação obrigatória**.

### 3) **Audit & Exports**
- `GET /admin/backoffice/actions` — lista (paginada) as últimas ações com filtros.  
- `GET /admin/backoffice/actions/{id}` — detalhe da auditoria (payloads, tempos, erro).  
- `GET /admin/backoffice/export/ledger?cpf=&format=csv|json&limit=50000` — export administrável do ledger (respeitando policy).


---

## 🔐 Segurança, RBAC & LGPD
- Roles mínimas: `ADMIN`, `SUPPORT`, `AUDITOR`.
  - `ADMIN`: pode **executar ações**, alterar **policy** e exportar.
  - `SUPPORT`: pode **consultar** e executar **recon-scan** (dry‑run), mas **não** pode resetar/zero‑refetch/policy‑set.
  - `AUDITOR`: somente **leitura** e **audit**.
- **`X-Tenant-Id` obrigatório**; todas as queries filtradas por tenant.  
- **CPF mascarado** nos logs e respostas de **search** (`***1234`); CPF completo apenas em payloads **administrativos** (jamais em métricas).  
- **Dupla confirmação** (confirmToken + TTL) para ações **perigosas**.  
- **Rate‑limit** por ação e **cotas** por ator (ex.: no máximo N `zero-and-refetch` por hora).


---

## 📊 Observabilidade
- **Logs** (JSON): `tenantId`, `actor`, `cpfMasked`, `action`, `status`, `durationMs`, `dryRun`, `resultCounts`.  
- **Métricas Prometheus**:
  - `admin_actions_total{action,status}`
  - `admin_actions_duration_seconds{action}` (histogram)
  - `admin_exports_total{format}`
  - `admin_profile_requests_total{result}`
- **Alertas**: alta taxa de `ERROR` em `admin_actions_total` ou explosão de `zero-and-refetch`.


---

## 🚦 Performance & Concorrência
- Locks por `(tenantId, cpf)` evitam corrida com jobs automáticos.  
- Ações **idempotentes** (reexecutar atualização de audit sem duplicar efeitos).  
- Exports **streaming** com **cap** de linhas e tempo.  
- Índices garantidos em tabelas consultadas (ledger, inconsistências, audit).


---

## 🧪 Testes (obrigatório)

**Integration — `src/Tests/Integration/Admin/backoffice_admin_test.go`**
- `search` retorna resultados com CPF mascarado e metadados.  
- `profile` agrega dados coerentes com ledger/inconsistências/policy/checkpoints.  
- Ações com **dry‑run** não alteram estado; com **confirm** executam e auditam.  
- **Dupla confirmação** exigida em `client-reset` e `zero-and-refetch`; sem confirm → não executa.  
- Idempotência: repetir **confirm** não duplica efeitos.  
- RBAC: `SUPPORT` não consegue executar ações perigosas.

**Unit**
- Geração/verificação de `confirmToken` + TTL, serialização de payloads, máscaras de CPF, merge de dados para `profile`.


---

## 📄 Documentação
- Swagger (tag `Backoffice Admin`) com exemplos realistas (sanitizados) e códigos 200/201/202/204 e 400/401/403/404/409/429/5xx.  
- Coleções Postman/Bruno em `/docs/admin/` com **folders por ação** (dry‑run e confirm).  
- `docs/admin/operations_guide.md`: **o que olhar** no Grafana/Prometheus para cada ação e **interpretação** dos campos do `profile`.


---

## 🔧 Configurações (.env)
- `BACKOFFICE_CONFIRM_TTL_SECONDS=600`  
- `BACKOFFICE_RATE_LIMIT_WINDOW=60s`  
- `BACKOFFICE_RATE_LIMIT_MAX_ACTIONS=30`  
- `OBS_MASK_CPF=true`


---

## ✅ Entregáveis
- Services, Repositories, Controllers, migrations.  
- Testes unitários e de integração.  
- Logs, métricas, RBAC, Swagger e collections.  
- **Aceite**: executar uma sequência real (dry‑run + confirm) para `b3-full-fetch` e `zero-and-refetch`, ver auditoria completa, métricas atualizadas e sem efeitos colaterais fora do tenant/CPF informados.
