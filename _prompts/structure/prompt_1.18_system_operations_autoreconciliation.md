# Prompt 1.18 — System Operations (auto‑reconciliação) — **suno-wallets**

Gere **operações de sistema** para **resolver inconsistências** detectadas no 1.17 **sem intervenção do usuário**, mantendo tudo **auditável, idempotente e reversível**.  
As operações criadas **não alteram RAW/Normalized**; são gravadas no **ledger unificado** e vinculadas à inconsistência de origem. O usuário poderá **substituir** depois (1.19).

> Nomes de arquivos/classes/funções em **inglês**; comentários em **pt‑BR**. **Sem dados fake**: se faltar preço, aplique as regras abaixo ou **marque como PENDING** e **não** gere operação.

---

## 🎯 Objetivos
1) Transformar inconsistências (1.17) em **System Operations** no ledger, com **razões claras** e **trilhas de auditoria**.  
2) Cobrir **duas regras** agora (CA ficará no 1.24):  
   - **OPENING_BALANCE_MISSING** → criar `OPENING_BALANCE` no **primeiro dia da API**.  
   - **SELL_WITHOUT_BUY** → criar `BUY` no **mesmo dia/preço da venda** (P&L pré‑API = 0).  
3) Garantir **idempotência** (reexecuções não duplicam), **locks** por CPF, **observabilidade** completa e **segurança**.  
4) Expor endpoints admin (`auto-fix`) com **dry‑run** e modo por **CPF** ou **por inconsistency id**.

---

## 🧱 Arquitetura (DDD)
Criar/atualizar (nomes em inglês, comentários pt‑BR):

- `src/Application/B3/reconciliation/system_operations_service.go`  
  Regras de geração, orquestração, idempotência e atualização do status da inconsistência.

- `src/Infrastructure/B3/reconciliation/system_operations_repository.go`  
  Upserts no ledger, vínculo `inconsistency_id`, travas por `(tenantId, cpf)`, leitura de preços auxiliares.

- `src/Api/Controllers/B3/reconciliation_controller.go`  
  Novas rotas: `POST /reconciliation/auto-fix` e `POST /reconciliation/auto-fix/{id}` (admin‑only).

- (Reuso) `b3_inconsistencies` do 1.17.  
- (Novo) **Ledger unificado**: `b3_operations_ledger` (migrations abaixo).

Pastas de testes:
- `src/Tests/Unit/B3/reconciliation/...`
- `src/Tests/Integration/B3/reconciliation/...`

---

## 🗃️ Banco de Dados (migrations)

### a) Ledger unificado de operações
```sql
CREATE TYPE operation_source AS ENUM ('B3_RAW','SYSTEM_SYNTHETIC','USER_MANUAL');
CREATE TYPE operation_type AS ENUM ('BUY','SELL','OPENING_BALANCE','ADJUSTMENT','FRACTION_ADJUSTMENT','CASH_ADJUSTMENT');

CREATE TYPE price_confidence AS ENUM ('MARKET_CLOSE','MIRRORED_SELL','DERIVED_POSITION','UNKNOWN');

CREATE TABLE IF NOT EXISTS b3_operations_ledger (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  cpf VARCHAR(11) NOT NULL,
  ticker TEXT NOT NULL,
  asset_type TEXT NOT NULL DEFAULT 'EQUITY',
  operation_date DATE NOT NULL,
  operation_type operation_type NOT NULL,
  source operation_source NOT NULL,
  quantity NUMERIC(28,10) NOT NULL,
  unit_price NUMERIC(28,10),
  currency TEXT NOT NULL DEFAULT 'BRL',

  -- rastreabilidade e reconciliação
  reason_code TEXT,                -- ex.: OPENING_BALANCE_PRE_API, ZERO_PNL_PRE_API
  price_confidence price_confidence NOT NULL DEFAULT 'UNKNOWN',
  generated_by_inconsistency_id UUID,   -- fk lógica para b3_inconsistencies.id
  supersedes_operation_id UUID,         -- preenchido no 1.19 (override do usuário)
  superseded_by_operation_id UUID,      -- preenchido no 1.19

  is_active BOOLEAN NOT NULL DEFAULT TRUE,  -- system/manual podem ser desativadas via override
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by TEXT
);

CREATE INDEX IF NOT EXISTS ix_ops_ledger_lookup
  ON b3_operations_ledger (tenant_id, cpf, ticker, operation_date);

-- idempotência de system ops (chave natural + hash)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ops_ledger_system_dedupe
ON b3_operations_ledger (tenant_id, cpf, ticker, operation_type, source, operation_date, reason_code, generated_by_inconsistency_id);
```

> Observação: `FRACTION_ADJUSTMENT` é reservado para **1.24** (CA Projection). Não usar aqui.

### b) Status da inconsistência (1.17)
Nenhuma mudança estrutural necessária. Apenas **atualize** `status` conforme regra abaixo (`RESOLVED` | `OPEN` | `OVERRIDDEN`), além de **gravar** no `details` os IDs de operações geradas.

---

## 🧠 Regras de Auto‑Reconciliação

### 1) **OPENING_BALANCE_MISSING → OPENING_BALANCE**
- **Quando**: posição existe no início do período B3, mas não há compras suficientes (detectado no 1.17).
- **Ação**: criar **uma** operação `OPENING_BALANCE` em `operation_date = B3_API_EARLIEST_DATE` (ou na **primeira `reference_date` com posição**).
- **Quantidade**: igual ao **saldo de posição** na data escolhida **menos** o acumulado líquido de transações até a data (não deve ser negativa).
- **Preço (`unit_price`)**:  
  1. Tentar **Preço de Fechamento** do papel na `operation_date` via `PriceLookupPort` (`MARKET_CLOSE`).  
  2. Se indisponível **e** existir valor na posição normalizada: usar `position_value / quantity` (`DERIVED_POSITION`).  
  3. Se tudo indisponível e `AUTO_FIX_REQUIRE_PRICE=true` → **não gerar** e marcar `status='OPEN'` com `details.reason='PENDING_PRICE'`.  
  4. Se `AUTO_FIX_REQUIRE_PRICE=false` → gerar com `unit_price=NULL`, `price_confidence='UNKNOWN'` (permitir override depois no 1.19).
- **Reason Code**: `OPENING_BALANCE_PRE_API`.
- **Pós‑ação**: `b3_inconsistencies.status='RESOLVED'` e anexar `generated_op_ids` nos `details`.

### 2) **SELL_WITHOUT_BUY → BUY espelho (P&L zero)**
- **Quando**: a primeira transação do ticker é `SELL` **ou** cumulativo líquido fica negativo.
- **Ação**: criar **uma** operação `BUY` na **mesma data** da primeira venda “inconsistente” com:  
  - **Quantidade**: magnitude necessária para **anular** o negativo (ou equiparar a venda).  
  - **Preço (`unit_price`)**: o **mesmo** da venda (espelho) → `price_confidence='MIRRORED_SELL'`.  
  - **Reason Code**: `ZERO_PNL_PRE_API`.
- **Pós‑ação**: `status='RESOLVED'`, registrar IDs gerados no `details`.

### Regras Comuns
- **Nunca** tocar em RAW/Normalized.  
- **Idempotência**: antes de inserir, calcular a chave natural → se já existe, **não** duplicar.  
- **Locks** por `(tenantId, cpf)` durante a execução.  
- **Multiexecução**: `dryRun=true` apenas **simula** e retorna plano; `dryRun=false` persiste.  
- **Auditoria**: `created_by`, `updated_by` (ex.: `system:auto-fix`).

---

## 🔌 Integração de Preço (porta/adapter)
Crie uma porta síncrona `PriceLookupPort`:
```go
type PriceLookupPort interface {
    // Retorna preço de fechamento em BRL para ticker/date.
    // Pode retornar (0, false) quando não disponível.
    GetClosingPrice(ctx context.Context, ticker string, date time.Time) (decimal.Decimal, bool, error)
}
```
- Implementação real pode vir depois; aqui você pode começar com um **adapter que consulta a própria posição normalizada** (quando possível).  
- Se não houver preço e `AUTO_FIX_REQUIRE_PRICE=true`, **não** gerar operação (deixe a inconsistência **OPEN** com motivo `PENDING_PRICE`).

---

## 📡 Endpoints (Swagger tag: `B3 Reconciliation`)

### `POST /reconciliation/auto-fix` *(admin‑only)*
Body:
```json
{
  "cpf": "00000000000",
  "types": ["OPENING_BALANCE_MISSING","SELL_WITHOUT_BUY"],
  "tickers": null,
  "dryRun": false,
  "force": false,
  "concurrency": 2
}
```
Semântica:
- Varre inconsistências **OPEN** do CPF filtradas por `types/tickers`.
- `dryRun=true` → **não** persiste, retorna plano e pré‑visualização das operações.
- `force=true` → reavalia inconsistências **RESOLVED** e tenta regenerar (idempotente).
- Resposta: sumário `{created, skipped, pending, resolvedIds, pendingIds}` + **lista de operações simuladas/geradas**.

### `POST /reconciliation/auto-fix/{id}` *(admin‑only)*
- Executa auto‑fix **apenas** para a inconsistência informada (útil em debugging).

**Segurança**: `X-Tenant-Id` obrigatório; RBAC admin; logs com CPF **mascarado**.  
**Rate‑limit** defensivo recomendado.

---

## 📊 Observabilidade
- **Logs estruturados**: `tenantId`, `cpfMasked`, `types`, `tickers`, `created/skipped/pending`, `durationMs`, `priceConfidenceMix`.  
- **Métricas Prometheus**:
  - `b3_autofix_runs_total{result}`
  - `b3_autofix_ops_created_total{reason_code}`
  - `b3_autofix_pending_total{reason="PENDING_PRICE|OTHER"}`
  - `b3_autofix_duration_seconds` (histogram)

> **Nota**: evitar alta cardinalidade; `tenantId/cpf` apenas nos **logs**.

---

## 🚦 Performance & Concorrência
- Processar por **lotes** de inconsistências; **janela por CPF** com lock.  
- Minimizar round‑trips com **upserts** em lote.  
- Respeitar `concurrency` para cálculos de preço/consulta a posições.
- Validar **índices**: `b3_inconsistencies(tenant_id, cpf, status, type)` e `b3_operations_ledger(tenant_id, cpf, ticker, operation_date)`.

---

## 🔐 Segurança & LGPD
- **Nunca** logar tokens/segredos; **mascarar** CPF (`***1234`).  
- Payloads de resposta **sem PII** sensível além do necessário.  
- **Multi‑tenant** estrito em todas as consultas/gravações.

---

## 🧪 Testes (obrigatório)

**Integration** — `src/Tests/Integration/B3/reconciliation/system_operations_test.go`
- `OPENING_BALANCE_MISSING` → cria `OPENING_BALANCE` na data correta; preço via `MARKET_CLOSE` ou `DERIVED_POSITION`; `PENDING` quando `AUTO_FIX_REQUIRE_PRICE=true` e sem preço.  
- `SELL_WITHOUT_BUY` → cria `BUY` espelho no mesmo dia/preço; zera o negativo; `status=RESOLVED`.  
- **Idempotência**: reexecutar **não** duplica (ver `uq_ops_ledger_system_dedupe`).  
- **Dry‑run** não persiste; **force** reavalia resolvidos.  
- **Lock** evita corrida com 1.13/1.14.

**Unit** — `src/Tests/Unit/B3/reconciliation/...`
- Cálculo de quantidade/price‑confidence; chaves naturais; mapeamento de reason codes; atualização de status e details da inconsistência.

---

## 📄 Documentação
- Swagger (tag `B3 Reconciliation`) atualizado com exemplos e erros (200/400/401/403/409/5xx).  
- Coleções Postman/Bruno em `/docs/reconciliation/` com **rotas de auto‑fix**.  
- `docs/b3/auto_fix.md` explicando as regras, *price confidence*, e cenários de `PENDING_PRICE`.

---

## 🔧 Configurações (.env sugeridas)
- `B3_API_EARLIEST_DATE=2019-10-01` *(confirmar)*  
- `AUTO_FIX_REQUIRE_PRICE=true` *(bloqueia geração sem preço)*  
- `AUTOFIX_MAX_CONCURRENCY=2`  
- `OBS_MASK_CPF=true`

---

## ✅ Entregáveis
- Service + Repository + Controller + migrations (ledger).  
- Testes unitários/integrados.  
- Logs, métricas, RBAC, Swagger e collections.  
- **Aceite**: rodar `POST /reconciliation/auto-fix` (dry‑run e real) para um CPF com inconsistências abertas e obter operações geradas **sem duplicação**, com **status** e **auditoria** corretos.
