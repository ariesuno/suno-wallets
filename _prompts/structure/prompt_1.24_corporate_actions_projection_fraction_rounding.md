# Prompt 1.24 — Corporate Actions Projection & Fraction Rounding (aplicação + *comicota*) — **suno-wallets**

Implemente a **aplicação de eventos corporativos** (projeção sobre posições) com **ajustes de quantidade/preço neutros economicamente**, incluindo a regra **“*comicota*”**: **nunca** manter frações — **sempre arredondar para baixo** — e registrar o residual como **ajuste de fração** (e opcionalmente **ajuste em dinheiro**).  
Este módulo **gera operações sintéticas** no **ledger unificado** para refletir **SPLIT, REVERSE_SPLIT, BONUS, MERGER, SPIN_OFF**, sem alterar RAW/Normalized e com **rastreabilidade completa**.

> Nomes de arquivos/classes/funções em **inglês**; comentários em **pt‑BR**. **Sem dados fake**. Multi‑tenant estrito. Observabilidade e testes obrigatórios.  
> **Identidade/canonicalização** veio do **1.23**. Aqui aplicamos **projeções** (quantidade/preço) com neutralidade econômica e *comicota*.


---

## 🎯 Objetivos
1) Aplicar **CA Projections** em nível de **posição por CPF/Ticker** gerando **SYSTEM_SYNTHETIC** no ledger:  
   - `ADJUSTMENT` (ajuste de quantidade neutro)  
   - `FRACTION_ADJUSTMENT` (remoção da fração → arredonda para baixo)  
   - `CASH_ADJUSTMENT` (opcional: *cash-in-lieu* da fração, se houver preço)  
2) **Neutralidade econômica**: ajustes **não** devem criar P&L; servem para reconciliar quantidade e base de custo.  
3) **Idempotência, versionamento e reversão**: permitir **preview**, **apply** e **rollback** por **versão**.  
4) **Observabilidade**, **RBAC admin**, **segurança/LGPD** e **testes** completos.


---

## 🧱 Arquitetura (DDD)
Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Domain/CorporateActions/ca_types.go`  
  Modelos: `CorporateActionKind {SPLIT, REVERSE_SPLIT, BONUS, MERGER, SPIN_OFF}`, DTO dos parâmetros por evento.

- `src/Application/CorporateActions/ca_projection_service.go`  
  Orquestra **preview/apply/rollback**, calcula deltas de quantidade com *comicota* e neutralidade econômica; usa `PriceLookupPort` (1.18).

- `src/Infrastructure/CorporateActions/ca_projection_repository.go`  
  Leitura agregada de **posições** (ledger) no *effective_date*, upserts de operações sintéticas e controle de versões/aplicações.

- `src/Api/Controllers/CorporateActions/ca_projection_controller.go`  
  Endpoints (preview/apply/rollback/status).

- **Reuso**: catálogo/canonicalização (1.23), ledger (1.18), timeline (1.20), backoffice (1.22).

Pastas de testes:
- `src/Tests/Unit/CorporateActions/...`
- `src/Tests/Integration/CorporateActions/...`


---

## 🗃️ Banco de Dados (migrations)

### a) Controle de projeções (versões)
```sql
CREATE TABLE IF NOT EXISTS ca_projection_runs (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  cpf VARCHAR(11),                     -- NULL = execução em lote do tenant
  version INTEGER NOT NULL,            -- versão incremental por tenant
  status TEXT NOT NULL,                -- DRAFT|APPLIED|ROLLED_BACK|ERROR
  effective_from DATE NOT NULL,
  effective_to DATE NOT NULL,
  params JSONB NOT NULL,               -- lista de eventos aplicados e regras
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT,
  completed_at TIMESTAMPTZ,
  notes TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_ca_runs_tenant_version
  ON ca_projection_runs (tenant_id, version);
```

### b) Ligação das operações com CA
```sql
ALTER TABLE b3_operations_ledger
  ADD COLUMN IF NOT EXISTS generated_by_ca_event_id UUID,
  ADD COLUMN IF NOT EXISTS ca_version_applied INTEGER;

-- Evitar duplicações da mesma projeção
CREATE UNIQUE INDEX IF NOT EXISTS uq_ops_ca_dedupe
ON b3_operations_ledger (
  tenant_id, cpf, ticker, operation_date, operation_type, source, generated_by_ca_event_id, ca_version_applied
);
```

> Observação: `operation_type` já inclui `ADJUSTMENT`, `FRACTION_ADJUSTMENT` e `CASH_ADJUSTMENT` (reservados no 1.18).


---

## 🧠 Regras de Projeção (por evento)

> **Ponto de partida**: posição agregada por **CPF/Ticker** no fechamento do dia **anterior** ao `effective_date` do evento.  
> **Alvo**: posição no **`effective_date`** após a aplicação das regras.

### Tipos suportados
- **SPLIT (Desdobramento)** `ratio = a:b` (ex.: 1:5 → multiplica quantidade por 5)  
  `newQty = floor(oldQty * (b/a))` (**aplicar *comicota***).

- **REVERSE_SPLIT (Grupamento)** `ratio = a:b` (ex.: 5:1 → divide quantidade por 5)  
  `newQty = floor(oldQty * (b/a))` (**aplicar *comicota***).

- **BONUS (Bonificação em ações)** `percent` (ex.: +10%)  
  `bonusQty = floor(oldQty * percent)` (**aplicar *comicota***).

- **MERGER (Incorporação/Conversão)** `ratio = a:b`, `toTicker`  
  `newQty(toTicker) = floor(oldQty(fromTicker) * (b/a))` (**aplicar *comicota***).  
  O `fromTicker` passa a 0; gerar `ADJUSTMENT` negativo nele e positivo no `toTicker`.

- **SPIN_OFF (Cisão/Distribuição)** `ratio = a:b`, `childTicker`  
  `childQty = floor(oldQty(parentTicker) * (b/a))` (**aplicar *comicota***).  
  **Não** reduzir `parentQty` a menos que a regra do evento exija.

### Neutralidade econômica
- **Objetivo**: **não** criar P&L. `ADJUSTMENT`/`FRACTION_ADJUSTMENT` devem ser **neutros** para custo e rentabilidade.  
- **Implementação sugerida**:
  - Marcar `ADJUSTMENT`/`FRACTION_ADJUSTMENT` com `reason_code='CA_PROJECTION'`.  
  - `unit_price = NULL` e tratar estes tipos como **neutros** nos cálculos da Fase 2 (não alteram custo ou P&L).  
  - `CASH_ADJUSTMENT` (opcional): se houver preço de referência no `effective_date` (`PriceLookupPort`), calcular `cash = fraction * price` e registrar com `currency='BRL'`. Se **não** houver preço e `AUTO_FIX_REQUIRE_PRICE=true`, **não** criar `CASH_ADJUSTMENT` (registrar *pending* no run).

### *Comicota* (frações)
- Sempre **arredonde para baixo**: `floor(x)`
- Registre a fração **removida** em `FRACTION_ADJUSTMENT` (`quantity = -fractionRemoved`).  
- Opcionalmente gere `CASH_ADJUSTMENT` pelo valor da fração (ver acima).
- Salve **detalhes** do cálculo (quantidades teóricas x aplicadas) nos `details` do run (payload).


---

## 🔌 Algoritmo (esboço)

1) **Coletar eventos** do catálogo (1.23) por intervalo `effective_from..effective_to` e `tickers` alvo.  
2) Para cada **CPF** (ou o CPF informado), por **ticker** e por **evento**, calcular:  
   - `oldQty` na *véspera* (`effective_date - 1`).  
   - `newQtyTheoretical` conforme regra do evento.  
   - `newQty = floor(newQtyTheoretical)`; `fraction = newQtyTheoretical - newQty`.  
3) **Gerar plano** (`preview`):  
   - `ADJUSTMENT` com `quantity = newQty - oldQtyConverted` (para MERGER/SPIN_OFF considere `toTicker/childTicker`).  
   - `FRACTION_ADJUSTMENT` com `quantity = -fraction` (se `fraction > 0`).  
   - `CASH_ADJUSTMENT` se preço disponível e configurado.  
4) **Apply**: inserir no ledger com `source='SYSTEM_SYNTHETIC'`, `operation_date = effective_date`, `generated_by_ca_event_id`, `ca_version_applied = version`.  
5) **Idempotência**: checar `uq_ops_ca_dedupe`; reexecuções **não** duplicam.  
6) **Rollback**: localizar operações da `version` e **desativar** (`is_active=false`).


---

## 📡 Endpoints (Swagger tag: `Corporate Actions — Projection`)

### `POST /catalog/ca/projection/preview`
Body:
```json
{
  "cpf": "00000000000|null",        // null = todos do tenant (batch)
  "from": "YYYY-MM-DD",
  "to": "YYYY-MM-DD",
  "tickers": null,
  "includeKinds": ["SPLIT","REVERSE_SPLIT","BONUS","MERGER","SPIN_OFF"],
  "simulateCashInLieu": true
}
```
Resposta (200): plano com contagens, frações removidas, cash estimado, e lista de operações simuladas (IDs **não** gerados).

### `POST /catalog/ca/projection/apply`
Body: igual ao preview + `{ "version": <int>, "dryRun": false }`  
- Se `version` **não** enviado: alocar **próxima versão** (`MAX(version)+1` no tenant).  
- Resposta: sumário com `version`, `created/skipped/pendingCash`, `byKind`, `byTicker` (top‑N), `runId`.

### `POST /catalog/ca/projection/rollback`
Body:
```json
{ "version": 7, "cpf": "00000000000|null" }
```
- Desativa (`is_active=false`) todas as operações da `version` no escopo.  
- Resposta: contagens por `operation_type`.


### `GET /catalog/ca/projection/status?cpf=&version=`
- Lista **runs** e estatísticas por versão (APPLIED/ROLLED_BACK), *pending cash*, frações removidas, etc.

**Segurança**: `X-Tenant-Id` obrigatório; todas as rotas **admin**.  
**LGPD**: mascarar CPF nos logs; payloads de relatórios **sem** PII desnecessária.


---

## 📊 Observabilidade
- **Logs** (JSON): `tenantId`, `scope` (cpf|tenant), `version`, `kinds`, `tickers`, `created/skipped/pending`, `fractionsTotal`, `durationMs`.  
- **Métricas Prometheus**:
  - `ca_projection_runs_total{status}`
  - `ca_projection_ops_created_total{operation_type}`
  - `ca_projection_fractions_total`
  - `ca_projection_duration_seconds` (histogram)

Evitar alta cardinalidade; **CPF apenas nos logs** (mascarado).


---

## 🚦 Performance
- Processamento por **lotes** de CPF/ticker; **keyset** para varrer ledger.  
- Índices: `b3_operations_ledger(tenant_id, cpf, ticker, operation_date)` + canônico (1.23).  
- Uso criterioso de `PriceLookupPort` (cacheável) no *cash-in-lieu*.  
- `EXPLAIN ANALYZE` nas queries de leitura e escrita em lote.


---

## 🔐 Segurança
- RBAC **admin** apenas; `X-Tenant-Id` obrigatório.  
- **Locks** por `(tenantId, cpf)` durante apply/rollback; idempotência garantida.  
- **Sem** logar tokens ou dados sensíveis.


---

## 🧪 Testes (obrigatório)

**Integration — `src/Tests/Integration/CorporateActions/ca_projection_test.go`**
- **SPLIT** e **REVERSE_SPLIT**: verificar `ADJUSTMENT` correto, fração → `FRACTION_ADJUSTMENT`; idempotência.  
- **BONUS**: bonificação aplica *floor*; `ADJUSTMENT` positivo + `FRACTION_ADJUSTMENT` se necessário.  
- **MERGER**: `fromTicker` zera (ajuste negativo) e `toTicker` recebe ajuste positivo no mesmo dia; *comicota* aplicada; idempotência.  
- **SPIN_OFF**: cria ajustes no `childTicker` (e no `parent` se regra exigir); *comicota* aplicada.  
- **CASH_IN_LIEU**: quando preço disponível e habilitado → `CASH_ADJUSTMENT` criado; se preço ausente e `AUTO_FIX_REQUIRE_PRICE=true` → `pendingCash>0` e **não** cria cash.  
- **Rollback** desativa operações da versão.  
- **Neutralidade econômica**: flags/semântica para a Fase 2 (verificação superficial nesta fase).

**Unit**
- Cálculo de `newQty`, fração e *floor*; composição das operações; versionamento/idempotência (chaves dedupe).


---

## 📄 Documentação
- **Swagger** (tag `Corporate Actions — Projection`) com exemplos.  
- **Postman/Bruno** em `/docs/ca/` (preview/apply/rollback/status).  
- `docs/ca/projection.md` explicando *comicota*, neutralidade econômica, versionamento e rollback.


---

## 🔧 Configurações (.env)
- `CA_PROJECTION_ENABLE_CASH_IN_LIEU=true`  
- `AUTO_FIX_REQUIRE_PRICE=true` *(reaproveitado do 1.18 para cash)*  
- `CA_PROJECTION_DEFAULT_BATCH_SIZE=5000`  
- `OBS_MASK_CPF=true`


---

## ✅ Entregáveis
- Service + Repository + Controller + migrations.  
- Testes unitários e de integração.  
- Logs, métricas, RBAC, Swagger e collections.  
- **Aceite**: executar `preview` e depois `apply` para um conjunto de eventos; ver `ADJUSTMENT`/`FRACTION_ADJUSTMENT` (e `CASH_ADJUSTMENT` se habilitado) gerados **sem duplicação**, com **versionamento** e **rollback** funcionais, e métricas/logs refletindo a execução.
