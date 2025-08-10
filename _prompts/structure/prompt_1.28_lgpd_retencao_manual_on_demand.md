# Prompt 1.28 — LGPD & Retenção Manual (on‑demand) — **suno-wallets**

Implemente ferramentas **administrativas, manuais e auditáveis** para **exportação e eliminação de dados pessoais** por **CPF** sob solicitação do titular ou por regra **manual** de inatividade.  
**Nunca** realizar purga **automática** do RAW. Toda execução deve ser **on‑demand** com **dry‑run** e **dupla confirmação**. Produzir **recibos de eliminação** (hash) e manter **tombstone/blocklist** para evitar reprocessamento sem novo consentimento.

> **Aviso**: Este prompt não é aconselhamento jurídico. Aplique‑o segundo sua política interna e orientação legal.  
> Nomes de arquivos/classes/funções em **inglês**; comentários em **pt‑BR**. **Sem dados fake**. Multi‑tenant estrito. Logs/métricas **sem PII**.


---

## 🎯 Objetivos
1) **Exportabilidade** (portabilidade): empacotar todos os dados do sujeito (CPF) em **ZIP** (JSON/CSV + manifesto).  
2) **Eliminação manual on‑demand**: pipeline com **dry‑run → request → confirm → apply**, **nunca automática**.  
3) **Auditoria**: recibo com **hash salgado** do CPF, contagens por tabela, operador, horários, versão e política aplicada.  
4) **Tombstone/Blocklist**: impedir re‑ingestão acidental do CPF sem novo consentimento (por tenant).  
5) **Observabilidade** e **testes**: métricas Prometheus, logs, Swagger/Collections, unitários e integração.


---

## 🧱 Arquitetura (DDD)

Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Domain/LGPD/lgpd_models.go`  
  Entidades/DTOs: `ErasureRequest`, `ErasurePlan`, `ErasureReceipt`, `SubjectExport`, `RetentionRule`.

- `src/Application/LGPD/lgpd_service.go`  
  Orquestra: `preview`, `request`, `confirm`, `apply`, `export`, `listRuns`, `lookup`, `dryRunRetention`.

- `src/Infrastructure/LGPD/lgpd_repository.go`  
  Consultas por CPF/tabelas, criação das *runs*, persistência de recibos, tombstones e blocklist; estratégia de *batch delete* idempotente.

- `src/Api/Controllers/LGPD/lgpd_admin_controller.go`  
  Endpoints admin (dry‑run/export/request/confirm/apply/status/runs).

- **Reuso/Integração**: política do cliente (1.21), backoffice (1.22), ledger/RAW/normalized (1.11–1.20), observabilidade (1.25).

Pastas de testes:  
- `src/Tests/Unit/LGPD/...`  
- `src/Tests/Integration/LGPD/...`


---

## 🗃️ Banco de Dados (migrations)

```sql
-- Registro das solicitações/execuções
CREATE TABLE IF NOT EXISTS lgpd_erasure_runs (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  cpf_hash TEXT NOT NULL,                 -- hash(salt|cpf), nunca CPF puro
  subject_id TEXT,                        -- opcional, se existir outro identificador interno
  reason TEXT NOT NULL,                   -- USER_REQUEST | ADMIN_POLICY
  mode TEXT NOT NULL,                     -- DRY_RUN | REQUESTED | CONFIRMED | RUNNING | SUCCESS | ERROR | CANCELED
  include_raw BOOLEAN NOT NULL DEFAULT TRUE,
  export_path TEXT,                       -- caminho do pacote de export (quando solicitado)
  confirm_token TEXT,                     -- token de dupla confirmação
  created_by TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  confirmed_at TIMESTAMPTZ,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  error TEXT
);

CREATE INDEX IF NOT EXISTS ix_lgpd_runs_tenant_created ON lgpd_erasure_runs (tenant_id, created_at DESC);

-- Contagens por tabela para auditoria
CREATE TABLE IF NOT EXISTS lgpd_erasure_counts (
  run_id UUID REFERENCES lgpd_erasure_runs(id) ON DELETE CASCADE,
  table_name TEXT NOT NULL,
  matched BIGINT NOT NULL DEFAULT 0,
  deleted BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (run_id, table_name)
);

-- Recibos/provas de eliminação (sem PII reversível)
CREATE TABLE IF NOT EXISTS lgpd_erasure_receipts (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  cpf_hash TEXT NOT NULL,
  run_id UUID NOT NULL REFERENCES lgpd_erasure_runs(id) ON DELETE CASCADE,
  summary JSONB NOT NULL,                 -- contagens, tabelas afetadas, versão do sistema, horário
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tombstones: previnem reprocessamento até novo consentimento
CREATE TABLE IF NOT EXISTS lgpd_subject_tombstones (
  tenant_id UUID NOT NULL,
  cpf_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  notes TEXT,
  PRIMARY KEY (tenant_id, cpf_hash)
);

-- Blocklist para ingestão/B3/manual até remover o tombstone explicitamente
CREATE TABLE IF NOT EXISTS lgpd_ingestion_blocklist (
  tenant_id UUID NOT NULL,
  cpf_hash TEXT NOT NULL,
  source TEXT NOT NULL,                   -- B3 | MANUAL | BOTH
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, cpf_hash, source)
);
```

> **Nunca** armazenar CPF puro nessas tabelas. Use `cpf_hash = SHA256(tenant_salt || cpf)`. `tenant_salt` permanece fora do código (ex.: variável de ambiente).


---

## 🔌 Escopo de Dados (tabelas alvo)

- **RAW**: `b3_raw_payloads_*` (todas), **não** apagar automaticamente em políticas; apagar **apenas** quando `include_raw=true` em pedido do titular.  
- **Normalized/Ledger**: `b3_operations_ledger`, `normalized_positions`, `normalized_transactions`, `manual_operations`, `system_synthetic_operations` etc.  
- **Auxiliares**: inconsistências (1.17), projeções CA (1.24), policies (1.21) por CPF, checkpoints e audits do backoffice (1.22).  
- **Observabilidade**: **não** apagar logs/métricas; se necessário, persistir *pseudônimos* (hash) em registros futuros.


---

## 🧠 Fluxo & Regras

1) **Lookup** (`GET /admin/lgpd/subject/lookup?cpf=`)  
   - Indica se CPF está **ativo/inativo**, datas da primeira/última atividade, volumes por domínio (RAW/ledger/manual/system), se está em **tombstone/blocklist**.

2) **Dry‑run** (`POST /admin/lgpd/erasure/preview`)  
   - Gera **plano** com contagens por tabela; **nunca** apaga. Persistir *run* `mode=DRY_RUN` e `lgpd_erasure_counts`.

3) **Export** (`POST /admin/lgpd/export`)  
   - Empacota dados do sujeito: JSON/CSV por tabela + `manifest.json` (versões, horários, tenant, hash).  
   - Armazena em `export_path` (ex.: `/var/exports/lgpd/<runId>.zip`) e retorna link local/assinado (ou caminho).

4) **Request** (`POST /admin/lgpd/erasure/request`)  
   - Cria *run* `mode=REQUESTED` com `confirm_token`. Requer `reason` e `include_raw` (padrão `true` em USER_REQUEST).

5) **Confirm** (`POST /admin/lgpd/erasure/confirm`)  
   - Exige `confirm_token`. Transiciona para `CONFIRMED`.

6) **Apply** (`POST /admin/lgpd/erasure/apply`)  
   - Executa **batch deletes** por tabela (ordem segura), com **idempotência** por `(run_id, table)` e *checkpoint* por *cursor*.  
   - Insere **tombstone** e **blocklist** (`BOTH`), gera **receipt** e finaliza `SUCCESS`.  
   - Em erro, marca `ERROR` com *retry* possível por tabela restante.

7) **Runs/Status** (`GET /admin/lgpd/erasure/runs`, `GET /admin/lgpd/erasure/status/{id}`)  
   - Lista/consulta runs, contagens, *export_path*, horários, operador, status e erros.

**Política Manual de Inatividade**  
- `POST /admin/lgpd/retention/dry-run` → recebe regra (ex.: `inactiveMonths>=24`, `includeRaw=false`). **Não aplica**.  
- Para aplicar, criar **uma *run* por CPF** via `erasure/request` (lista explicitamente os CPFs a partir do dry‑run). **Nada em lote automático**.


---

## 📡 Endpoints (Swagger tag: `LGPD — Admin`)

- `GET /admin/lgpd/subject/lookup?cpf=`  
- `POST /admin/lgpd/erasure/preview` *(dry‑run c/ contagens por tabela)*  
- `POST /admin/lgpd/export` *(gera ZIP + manifesto; retorna caminho/URL)*  
- `POST /admin/lgpd/erasure/request` *(gera confirmToken)*  
- `POST /admin/lgpd/erasure/confirm` *(dupla confirmação)*  
- `POST /admin/lgpd/erasure/apply` *(executa pipeline por runId)*  
- `GET /admin/lgpd/erasure/status/{id}`  
- `GET /admin/lgpd/erasure/runs` *(paginado)*  
- `POST /admin/lgpd/retention/dry-run` *(descobrir candidatos por regra manual, **sem** aplicação)*

**Segurança**: `X-Tenant-Id` obrigatório; RBAC: `ADMIN` (request/confirm/apply/export), `SUPPORT` (lookup/preview/status/runs).  
**LGPD**: sem CPF em logs/métricas; CPF sempre mascarado nas respostas (quando inevitável).


---

## 🧾 Logs & 📊 Métricas

**Logs (JSON)**  
- Campos: `tenantId`, `runId`, `actor`, `mode`, `reason`, `includeRaw`, `tables`, `counts`, `durationMs`, `error?` e `cpfMasked`.  
- **Nunca** logar CPF puro; usar `cpfMasked="***1234"` + `cpf_hash` somente quando estritamente necessário.

**Prometheus**  
- `lgpd_erasure_runs_total{status}`  
- `lgpd_erasure_duration_seconds` (histogram)  
- `lgpd_erasure_deleted_rows_total{table}`  
- `lgpd_exports_total{format="zip"}`  
- `lgpd_blocklist_gauge{source}`  

**Alertas (sugestões)**  
- `lgpd_erasure_error_rate` alto em 30m.  
- `lgpd_erasure_duration_seconds` p95 acima do limite durante 15m.


---

## 🔐 Segurança & Conformidade

- **Dupla confirmação**: `request → confirm (token) → apply`.  
- **Hash do CPF**: `SHA256(tenant_salt || cpf)`. `tenant_salt` em `.env` seguro e rotacionável.  
- **Backups**: registrar que **backups legados** podem conter o dado; política de retenção deve ser documentada fora do escopo (não restaurar sujeitos apagados).  
- **Blocklist/Tombstone**: evita reprocesso. Para remover, exigir **novo consentimento** explícito.  
- **Idempotência**: repetir `apply` não duplica; *checkpoints* por tabela com cursores primários.  
- **Transparência**: gerar **receipt** com sumário e caminho da export quando aplicável.


---

## 🚦 Performance

- *Batch delete* por chaves (`cpf`, `tenant_id`) com **lotes** (ex.: 5k linhas).  
- Índices necessários nas tabelas alvo: `(tenant_id, cpf)` ou equivalentes; *EXPLAIN ANALYZE* nas *queries de varredura*.  
- Evitar *locks* longos; usar **janelas** por intervalo de datas/IDs.  
- Executar em *worker* dedicado; controlar concorrência por `(tenantId, cpf_hash)`.

---

## 🧪 Testes (obrigatório)

**Integration — `src/Tests/Integration/LGPD/lgpd_on_demand_test.go`**
- `lookup` retorna volumes coerentes e sinaliza tombstone/blocklist quando presentes.  
- `preview` grava *run* DRY_RUN + `lgpd_erasure_counts`.  
- `export` cria ZIP com manifesto e arquivos por tabela; caminho é persistido.  
- `request` cria run com `confirm_token`; `confirm` altera o estado.  
- `apply` exclui linhas (ledger, manual, synthetic, inconsistências, CA), respeitando `include_raw`; insere `tombstone` + `blocklist`; cria **receipt**; idempotência ok.  
- `retention/dry-run` gera candidatos mas **não** aplica nada.  
- RBAC/headers: `X-Tenant-Id` obrigatório; `SUPPORT` sem permissão de `apply` (403).

**Unit**
- Geração/validação de `confirm_token`; hash salgado de CPF; mascaração de CPF para logs; composição do manifesto de export; *cursor batching* por tabela.

---

## 📄 Documentação
- **Swagger** (tag `LGPD — Admin`) com exemplos sanitizados.  
- **Postman/Bruno** em `/docs/lgpd/` (lookup/preview/export/request/confirm/apply/status/runs/retention).  
- `docs/lgpd/runbook.md`: operação segura, limitações (backups), como reverter tombstone (mediante novo consentimento).

---

## 🔧 Configurações (.env)
- `LGPD_TENANT_SALT=<random-hex>`  
- `LGPD_EXPORT_DIR=/var/exports/lgpd`  
- `LGPD_ERASURE_BATCH_SIZE=5000`  
- `LGPD_BLOCKLIST_DEFAULT_SOURCE=BOTH`  
- `OBS_MASK_CPF=true`

---

## ✅ Entregáveis
- Service + Repository + Controller + migrations.  
- Testes unitários e de integração.  
- Logs, métricas, Swagger e collections.  
- **Aceite**: executar `preview → export (opcional) → request → confirm → apply` para um CPF, ver **receipt** e **tombstone/blocklist** criados, contagens coerentes e **nenhuma purga automática de RAW** fora de pedido explícito.
