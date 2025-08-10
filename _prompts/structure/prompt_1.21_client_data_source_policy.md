# Prompt 1.21 — Client Data Source Policy (B3_ONLY | MANUAL_ONLY | HYBRID) — **suno-wallets**

Implemente **política de fonte de dados por cliente** para controlar ingestão, escrita e leitura de operações, cobrindo os modos:  
**`B3_ONLY`**, **`MANUAL_ONLY`** e **`HYBRID`**.  
A política deve ser **centralizada**, **cacheável**, **auditável**, **respeitada** por *jobs* (1.11/1.14), Manual Ops/Dedup (1.19) e pelas APIs.

> Nomes de arquivos/classes/funções em **inglês**; comentários **pt‑BR**. **Sem dados fake**. Multi‑tenant estrito. Observabilidade obrigatória.

---

## 🎯 Objetivos
1) Persistir e expor **policy** por `{tenant_id, cpf}` com auditoria e histórico simples.  
2) Enforce consistente em **ingest B3**, **Manual Ops CRUD** e **leitura** (quando aplicável).  
3) **Cache** (in‑memory + adapter opcional) com **invalidação** on‑write.  
4) Endpoints admin para **get/set**, *dry‑run impact* e **listagem**.  
5) Métricas, logs e testes (unit/integration).

---

## 🧱 Arquitetura (DDD)
Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Domain/ClientPolicy/client_policy.go`  
  Enum `DataSourceMode`, entidade `ClientPolicy`, regras de validação.

- `src/Application/ClientPolicy/client_policy_service.go`  
  Upsert, leitura com cache, invalidation; *dry‑run* de impacto.

- `src/Infrastructure/ClientPolicy/client_policy_repository.go`  
  Persistência e queries; histórico simples de alterações.

- `src/Infrastructure/ClientPolicy/client_policy_cache.go`  
  Cache em memória (LRU/TTL) + porta opcional para Redis (interface).

- `src/Api/Controllers/ClientPolicy/client_policy_controller.go`  
  Rotas admin de CRUD/lista e *dry‑run impact*.

- **Enforcers (hooks)**  
  - `src/Application/B3/ingest/ingest_enforcer.go` (1.11/1.14 leem policy antes de executar)  
  - `src/Application/Ops/manual_ops_enforcer.go` (1.19 bloqueia POST/PUT/DELETE quando `B3_ONLY`)  
  - `src/Application/Ops/read_enforcer.go` (p.ex., ignora B3_RAW quando `MANUAL_ONLY` em consultas *read‑side*)

Pastas de testes:
- `src/Tests/Unit/ClientPolicy/...`
- `src/Tests/Integration/ClientPolicy/...`

---

## 🗃️ Banco de Dados (migrations)
```sql
CREATE TYPE data_source_mode AS ENUM ('B3_ONLY','MANUAL_ONLY','HYBRID');

CREATE TABLE IF NOT EXISTS client_data_source_policy (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  cpf VARCHAR(11) NOT NULL,
  mode data_source_mode NOT NULL DEFAULT 'HYBRID',
  reason TEXT,                                -- justificativa da mudança
  effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  effective_to   TIMESTAMPTZ,                 -- opcional (nulo = vigente)
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by TEXT,
  UNIQUE (tenant_id, cpf)                      -- uma vigente por cliente
);

-- Histórico minimalista (apenas *append*)
CREATE TABLE IF NOT EXISTS client_data_source_policy_audit (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  cpf VARCHAR(11) NOT NULL,
  old_mode data_source_mode,
  new_mode data_source_mode,
  changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  changed_by TEXT,
  reason TEXT
);

CREATE INDEX IF NOT EXISTS ix_client_policy_lookup ON client_data_source_policy (tenant_id, cpf, mode);
CREATE INDEX IF NOT EXISTS ix_client_policy_audit ON client_data_source_policy_audit (tenant_id, cpf, changed_at DESC);
```

---

## 🧠 Regras de Enforce (modos)

### `B3_ONLY`
- **Permite**: ingestão B3 (full/incremental), leitura de B3_RAW/System no ledger.
- **Bloqueia**: CRUD de `USER_MANUAL` (retornar 409 `POLICY_VIOLATION_MANUAL_DISABLED`).
- **Dedup**: desabilitar varredura contra USER_MANUAL (não há base manual).

### `MANUAL_ONLY`
- **Permite**: CRUD de `USER_MANUAL`, leitura apenas de USER_MANUAL/System (ignorar B3_RAW em consultas/relatórios).
- **Bloqueia**: ingestão B3 (retornar 409 `POLICY_VIOLATION_B3_DISABLED`).
- **Jobs 1.11/1.14**: **não** executam para este CPF (registrar skip com métrica).

### `HYBRID`
- **Permite** ambos + **Dedup** ativo (1.19).

> Observação: *System Operations* (1.18) sempre podem existir (derivam de reconciliação), porém honram a leitura conforme o modo.

---

## 📡 Endpoints (Swagger tag: `Client Policy`)

1) `GET /admin/client/policy?cpf=`  
   Retorna a policy atual e metadados (`effective_from`, `updated_by`, `reason`).

2) `POST /admin/client/policy` *(admin‑only)*  
Body:
```json
{
  "cpf": "00000000000",
  "mode": "B3_ONLY|MANUAL_ONLY|HYBRID",
  "reason": "motivo opcional"
}
```
Semântica:
- Upsert da policy (cria se não existir, atualiza se existir).
- Salva entrada no **audit log**.
- **Invalida cache** imediatamente.

3) `POST /admin/client/policy/dry-run` *(admin‑only)*  
Body:
```json
{
  "cpf": "00000000000",
  "newMode": "B3_ONLY|MANUAL_ONLY|HYBRID"
}
```
Resposta:
```json
{
  "affectedJobs": ["B3_FullFetch","B3_Incremental","DedupScan","ManualOpsCRUD"],
  "blockedEndpoints": ["/ops/manual (POST, PUT, DELETE)","/b3/fetch/*"],
  "readSideChanges": ["ignore B3_RAW in queries"]
}
```

4) `GET /admin/client/policy/audit?cpf=`  
   Lista mudanças com data, usuário e reason.

**Segurança**: `X-Tenant-Id` obrigatório; RBAC admin para writes/dry‑run; leitura para admin/suporte.

---

## 🧩 Integrações obrigatórias

- **Jobs B3** (1.11/1.14): checar policy **antes** de iniciar; se `MANUAL_ONLY` → **skip** e log métrica `policy_skipped_total{job="B3_*"}`.  
- **Manual Ops CRUD** (1.19): checar policy; se `B3_ONLY` → **409**.  
- **Read‑side**: consultas de timeline/summary (1.20) devem oferecer `respectPolicy=true` (default) para **ignorar B3_RAW** quando `MANUAL_ONLY`.  
- **Dedup** (1.19): ativo apenas no `HYBRID` (e quando existirem bases comparáveis).

---

## 🧠 Cache & Invalidação

- **In‑memory LRU** com TTL (`CLIENT_POLICY_TTL_SECONDS`, default 300s).  
- Interface opcional para Redis:
```go
type ClientPolicyCache interface {
    Get(ctx context.Context, tenantID string, cpf string) (ClientPolicy, bool)
    Set(ctx context.Context, tenantID string, cpf string, policy ClientPolicy, ttl time.Duration) error
    Invalidate(ctx context.Context, tenantID string, cpf string) error
}
```
- **On‑write**: invalidar cache local e publicar *event* opcional (no futuro).  
- Fallback seguro: em caso de erro no cache, buscar direto no repositório.

---

## 📊 Observabilidade

- **Logs** (JSON): `tenantId`, `cpfMasked`, `mode`, `actor`, `action`, `durationMs`, `result`.  
- **Métricas** (Prometheus):
  - `client_policy_reads_total{source="cache|db", result}`
  - `client_policy_writes_total{result}`
  - `client_policy_cache_invalidations_total`
  - `policy_skipped_total{job="B3_FullFetch|B3_Incremental"}`
- **Alertas**: taxa alta de `policy_skipped_total` por CPF pode indicar configuração errada.

---

## 🔐 Segurança & LGPD

- Header **`X-Tenant-Id` obrigatório**; RBAC: admin para writes, admin/suporte para leitura.  
- **Mascarar CPF** em logs (`***1234`); **não** logar tokens/segredos.  
- Validação de entrada: `cpf` 11 dígitos; `mode` válido; razão ≤ 500 chars.

---

## 🚦 Performance

- Índices de busca por `(tenant_id, cpf)` (já criados).  
- Cache com TTL reduz latência em *hot paths*.  
- Upsert transacional com gravação em **audit**.

---

## 🧪 Testes (obrigatório)

**Integration — `src/Tests/Integration/ClientPolicy/client_policy_test.go`**
- `POST /admin/client/policy` cria/atualiza com audit + invalidação de cache.  
- `B3_ONLY` → `POST /ops/manual` retorna **409**; `HYBRID` permite.  
- `MANUAL_ONLY` → jobs 1.11/1.14 retornam **skip** com métrica `policy_skipped_total` incrementada.  
- `GET /admin/client/policy` e `.../audit` retornam dados corretos.

**Unit**
- Lógica de cache (hit/miss/ttl/invalidate), *dry‑run impact*, validações de entrada.

---

## 📄 Documentação
- **Swagger** (tag `Client Policy`) com exemplos sanitizados e códigos 200/201/400/401/403/409/5xx.  
- **Postman/Bruno** em `/docs/client-policy/` com requests prontos.  
- `docs/client_policy/overview.md` explicando os modos, enforce e integrações.

---

## 🔧 Configurações (.env)
- `CLIENT_POLICY_TTL_SECONDS=300`  
- `OBS_MASK_CPF=true`

---

## ✅ Entregáveis
- Domain + Service + Repository + Cache + Controllers + migrations.  
- Integrações de *enforce* nos módulos 1.11/1.14/1.19/1.20.  
- Testes (unit/integration), logs, métricas, Swagger e collections.  
- **Aceite**: alterar o modo de um CPF e observar imediatamente o efeito nas rotas/jobs (enforce), com cache invalidado, auditoria registrada e métricas atualizadas.
