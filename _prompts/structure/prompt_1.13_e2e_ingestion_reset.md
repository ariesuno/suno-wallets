# Prompt 1.13 — E2E Ingestion Runner + Reset & Full Re‑Fetch (admin) — **suno-wallets**

Implemente uma **orquestração fim‑a‑fim** que permita **resetar com segurança** todos os dados B3 de um **CPF** (RAW + Normalized + marcadores de sync) e executar uma **reingestão histórica completa**, a partir do **primeiro dia disponível da API da B3** até **hoje**, concluindo com a **normalização**.  
> **Mandatório**: fluxo **manual/admin**; **sem limpeza automática**. Reset somente mediante **confirmação explícita** e registro de auditoria.

---

## 🎯 Objetivos
1. Endpoint **admin‑only**: `POST /b3/admin/reset-and-refetch`
   - Confirmação obrigatória (`"confirm": "RESET_AND_REFETCH"`).
   - Suporta **`dryRun`** (plano sem execução) e **`force`** (reconsulta mesmo já havendo RAW).
   - Modo de limpeza: `mode = "archive" | "hard-delete"` (default: `archive`, seguro).
2. **Reingestão histórica** via serviços do **1.9 (RAW Persistence)**, **mês a mês**, para **transações (v2)** e **posições (v3)**.
3. **Normalização completa** via serviço do **1.10 (Normalization)** para todo RAW recém‑gravado.
4. **Sumário final** com tempos e totais por etapa.
5. **Observabilidade completa** (logs, métricas, auditoria) e **locks** por `(tenantId, cpf)`.

---

## 🧱 Arquitetura (DDD)
**Criar/atualizar** (nomes em inglês, comentários pt‑BR):

- `src/Application/B3/e2e/e2e_orchestrator.go`  
  Orquestra o fluxo **reset → ingest (1.9) → normalize (1.10)**; implementa **locks**, plano (`dryRun`) e consolidação de **sumário** por janelas mensais.
- `src/Infrastructure/B3/e2e/reset_repository.go`  
  Executa **reset seguro** dos dados do CPF (mover → `_archive` ou `hard-delete`), limpa `b3_fetched_periods` e **reseta** `b3_sync_state`.
- `src/Api/Controllers/B3/admin_controller.go`  
  Exponde `POST /b3/admin/reset-and-refetch` (admin‑only), valida payload e aciona o orquestrador.
- **Reuso obrigatório** dos serviços existentes (não duplicar lógica):  
  - **1.9** ingest histórico (fetch + persist RAW),  
  - **1.10** normalize (RAW → Normalized).

**Pastas de teste**:
- `src/Tests/Unit/B3/e2e/...`
- `src/Tests/Integration/B3/e2e/...`

---

## 🔐 Segurança & Multi‑tenant
- **Admin‑only** (RBAC): exigir perfil admin no tenant.  
- **`X-Tenant-Id` obrigatório**; todo acesso filtrado por `tenantId`.  
- **LGPD**: reset é **manual**. `mode="archive"` **move** dados para tabelas `_archive` (reversível). `mode="hard-delete"` requer confirmação extra `"confirmHard": "YES_DELETE"`.  
- **Mascarar CPF** nos logs; **nunca** logar tokens/segredos.

---

## 🗃️ Banco de Dados (migrations)
Se necessário, criar tabelas de **arquivo** para suportar `archive`:

- `b3_raw_data_client_archive` — mesmo schema de `b3_raw_data_client` + `archived_at timestamptz not null`, `archived_by text`.
- `b3_normalized_transactions_archive` — idem `b3_normalized_transactions` + `archived_at/by`.
- `b3_normalized_positions_archive` — idem `b3_normalized_positions` + `archived_at/by`.

**Auditoria (opcional):** `b3_admin_resets`  
`(id uuid pk, tenant_id, cpf, mode, started_at, finished_at, raw_moved, raw_deleted, normalized_moved, normalized_deleted, requested_by, notes jsonb)`

> **Nunca** automatizar purga de RAW. Operações de limpeza são **explícitas** e auditadas.

---

## ⚙️ Contrato do endpoint
### `POST /b3/admin/reset-and-refetch`

**Body (JSON):**
```json
{
  "cpf": "00000000000",
  "assetTypes": ["equity"],
  "dataTypes": ["transactions", "positions"],
  "startOverride": null,
  "endOverride": null,
  "force": true,
  "dryRun": false,
  "mode": "archive",
  "confirm": "RESET_AND_REFETCH",
  "confirmHard": null
}
```

**Regras & defaults:**
- `cpf`: **11 dígitos** (validar).
- `assetTypes`/`dataTypes`: listas permitidas (validar).
- Janela:
  - `start = startOverride ?? B3_API_EARLIEST_DATE` (ex.: `"2019-10-01"` — **configurável** e **a confirmar**),
  - `end = endOverride ?? today()` (UTC).
- `force=true` → reconsulta **tudo**; `dryRun=true` → retorna **plano** sem executar.
- `mode`:
  - `"archive"` (default): mover dados do CPF para tabelas `_archive`.
  - `"hard-delete"`: requer `"confirmHard": "YES_DELETE"`.

**Resposta (sumário):**
```json
{
  "raw":        {"saved": 0, "skipped": 0, "errors": 0, "monthsProcessed": 0, "pagesProcessed": 0},
  "normalized": {"inserted": 0, "updated": 0, "skipped": 0, "errors": 0},
  "startedAt": "ISO-8601",
  "finishedAt": "ISO-8601",
  "durationMs": 0,
  "mode": "archive",
  "force": true,
  "dryRun": false
}
```

---

## 🔁 Fluxo detalhado
1. **Validar** admin/tenant/CPF e confirmações (`confirm`, `confirmHard` se necessário).  
2. **Lock** `(tenantId, cpf)` (Redis lock ou DB advisory lock).  
3. **Planejar** (sempre): construir **plano mensal** `start→end` por `dataType × assetType`; estimar páginas.  
4. Se **`dryRun=true`** → **não** executar; retornar **plano**.  
5. **Reset seguro**:
   - `archive`: mover linhas de `b3_raw_data_client` e `b3_normalized_*` p/ `_archive` com `archived_at/by`.
   - `hard-delete`: apagar definitivamente (somente com `"confirmHard": "YES_DELETE"`).
   - Limpar `b3_fetched_periods` do CPF e **resetar** `b3_sync_state` (`last_*_sync_at=null`, `needs_reprocess=false`, `failure_count=0`).
6. **Ingest histórico (1.9)**:
   - Para cada mês/endpoint, chamar o **serviço 1.9** (com `force=true`), respeitando **retry/backoff + jitter** (1.6) e timeouts.
7. **Normalização (1.10)**:
   - Invocar o **serviço 1.10** para o mesmo escopo (`dataTypes/assetTypes`, `start→end`).
8. **Finalizar**:
   - Consolidar **sumário**, gravar auditoria (`b3_admin_resets`) e **liberar lock**.

**Falhas & rollback**:  
- Usar **transações por bloco** (ex.: por tabela/lote).  
- Em erro, registrar no sumário (`stage`, `reason`), avaliar retorno **207** (multi‑status) ou **500** (padrão interno).  
- Manter **idempotência** (1.9/1.10) para repetição segura.

---

## 📊 Observabilidade
- **Logs (JSON)**: `tenantId`, `cpfMasked`, `mode`, `force`, `dryRun`, `stage`, `durationMs`, totais por etapa.  
- **Métricas Prometheus**:
  - `b3_e2e_runs_total{result="success|error", mode, dataType, assetType}`
  - `b3_e2e_duration_seconds` (histogram)
  - `b3_e2e_errors_total{stage="reset|ingest|normalize"}`

---

## 🚦 Performance & Limites
- Processar por **meses** e **páginas** (reaproveitar paginação 1.7/1.8).  
- Controlar **concorrência** (`E2E_MAX_CONCURRENCY`).  
- Respeitar **rate limit** da B3 (regras do 1.6).  
- Propagar **context** e **timeouts**.

---

## 🧪 Testes (obrigatório)
**Integration** — `src/Tests/Integration/B3/e2e_reset_refetch_test.go`  
- `dryRun=true` → retorna **plano** sem alterar BD.  
- Execução real (`mode=archive`) → dados do CPF movidos p/ `_archive`; **RAW (1.9)** salvo; **Normalized (1.10)** gerado; sumário coerente.  
- **Idempotência**: rodar 2x com `force=false` **não duplica**.  
- Simular `429/5xx` → observar retries/métricas no **stage** correto.

**Unit** — `src/Tests/Unit/B3/e2e/e2e_orchestrator_test.go`  
- Planejamento mensal, lock, composição de sumário, rollback por etapa.

---

## 📄 Documentação
- **Swagger** (tag `B3 Admin`) com request/response e erros 200/400/401/403/409/429/5xx.  
- **Postman/Bruno** em `/docs/admin/` (exemplos `archive` e `hard-delete`).  
- `docs/b3/e2e_reset_refetch.md` com cURLs e troubleshooting.

### Exemplos cURL
```bash
# Dry run (plano)
curl -X POST "$BASE_URL/b3/admin/reset-and-refetch" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
        "cpf":"00000000000",
        "assetTypes":["equity"],
        "dataTypes":["transactions","positions"],
        "force": true,
        "dryRun": true,
        "mode": "archive",
        "confirm": "RESET_AND_REFETCH"
      }'

# Execução real (archive)
curl -X POST "$BASE_URL/b3/admin/reset-and-refetch" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
        "cpf":"00000000000",
        "assetTypes":["equity"],
        "dataTypes":["transactions","positions"],
        "force": true,
        "dryRun": false,
        "mode": "archive",
        "confirm": "RESET_AND_REFETCH"
      }'

# Execução com hard-delete (exige confirmação extra)
curl -X POST "$BASE_URL/b3/admin/reset-and-refetch" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
        "cpf":"00000000000",
        "assetTypes":["equity"],
        "dataTypes":["transactions","positions"],
        "force": true,
        "dryRun": false,
        "mode": "hard-delete",
        "confirm": "RESET_AND_REFETCH",
        "confirmHard": "YES_DELETE"
      }'
```

---

## 🔧 Configurações (.env sugeridas)
- `B3_API_EARLIEST_DATE=2019-10-01`  ← **Confirmar data oficial** da B3 v2/v3.  
- `E2E_MAX_CONCURRENCY=2`  
- `E2E_ARCHIVE_ENABLED=true`  
- `OBS_MASK_CPF=true`

---

## ✅ Entregáveis
- Controller + orquestrador + repos de reset + migrations `_archive` (se faltarem).  
- Testes unitários e de integração.  
- Métricas, logs e auditoria.  
- Swagger + Postman/Bruno + docs atualizados.

**Critério de aceite**: executar **dryRun** e **execução real** com CPF de teste, obter **sumários coerentes**, sem duplicidade, com logs/métricas publicadas e **sem** vazamento de PII/segredos.
