# Prompt 1.14 — Incremental Sync Runner (on‑demand) — **suno-wallets**

Implemente **gatilhos on‑demand** para sincronização **incremental** de um CPF, reutilizando os módulos de **RAW Persistence (1.9)** e **Normalization (1.10)**. A janela de sincronismo deve partir do **último marco salvo** em `b3_sync_state` (1.11) ou de um **`since`** informado, avançando **apenas** sobre o que ainda **não** foi buscado. Inclua endpoint de **execução admin** e endpoint **público (autenticado)** de **inspeção de janela** — sem persistir.

> **Importante**: não recalcular histórico inteiro; **apenas lacunas**. Em caso de nova ingestão, marcar `needs_reprocess=true` para a Fase 2 (consolidação).

---

## 🎯 Objetivos
1. **Executar** ingest/normalize incremental **sob demanda** para `transactions (v2)` e/ou `positions (v3)`.
2. Calcular a **janela** automaticamente por tipo usando `last_tx_sync_at` / `last_pos_sync_at` (1.11) **ou** parâmetro `since`.
3. Reutilizar **1.9** (busca/persistência **idempotente** do RAW) e **1.10** (normalização), **sem** duplicar lógica.
4. Atualizar `b3_sync_state` **apenas** quando houver **sucesso**; em caso de novidade, marcar `needsReprocess=true`.
5. Expor endpoint de **inspeção** da janela para debug (somente leitura, sem gravação).

---

## 🧱 Arquitetura (DDD)
**Criar/atualizar** (nomes em inglês, comentários pt‑BR):

- `src/Application/B3/incremental/incremental_service.go`  
  - Calcula a janela `(from→to]` por `dataType` baseada em `last_*_sync_at` **ou** `since` recebido.  
  - Divide em **janelas mensais**, invoca **1.9** (apenas lacunas) e **1.10** em seguida.  
  - Atualiza `b3_sync_state` somente em **sucesso**; seta `needsReprocess=true` quando houver **novos RAWs**.
- `src/Api/Controllers/B3/admin_controller.go`  
  - Adiciona ação `POST /b3/admin/incremental-from-last` (admin‑only).
- `src/Api/Controllers/B3/util_controller.go`  
  - Adiciona `GET /b3/client/sync-window` para **inspeção de janela** (somente leitura).
- **Reuso** obrigatório: `B3OfficialClient` (1.6), **1.9**, **1.10**, **1.11** (`b3_sync_state`).

**Testes**:
- `src/Tests/Unit/B3/incremental/...`
- `src/Tests/Integration/B3/incremental/...`

---

## 🗃️ Integração com `b3_sync_state` (1.11)
- `b3_sync_state.last_tx_sync_at` e `last_pos_sync_at` são os **marcos** usados para calcular a janela quando `since` **não** é informado.  
- **Somente** atualizar os marcos quando a execução de cada tipo finalizar com **sucesso**.  
- Registrar resultado em `last_result` (`OK`, `RATE_LIMIT`, `AUTH_ERROR`, `B3_ERROR`, etc.), `last_checked_at` e `failure_count` quando falhar.

---

## 🔗 Contratos dos Endpoints

### `POST /b3/admin/incremental-from-last`  *(admin‑only)*
**Body (JSON):**
```json
{
  "cpf": "00000000000",
  "dataTypes": ["transactions", "positions"],
  "assetTypes": ["equity"],
  "since": null,
  "endOverride": null,
  "dryRun": false,
  "force": false,
  "concurrency": 2
}
```
**Semântica:**
- Se `since` informado → janela `since → (endOverride || today)`; caso contrário, usar `last_*_sync_at` por tipo.  
- `force=true` → reconsulta **toda** a janela calculada (mesmo se já houver RAW).  
- `dryRun=true` → retorna **plano** (meses/páginas estimadas), **não** grava RAW nem normalize.  
- Atualiza `b3_sync_state` apenas quando `dryRun=false` e execução **OK**.  
- Se novos RAWs forem gravados, marcar `needsReprocess=true` para o CPF.

**Resposta (sumário por tipo):**
```json
{
  "transactions": {"raw": {"saved": 0, "skipped": 0, "errors": 0, "monthsProcessed": 0, "pagesProcessed": 0},
                   "normalized": {"inserted": 0, "updated": 0, "skipped": 0, "errors": 0},
                   "from": "YYYY-MM-DD", "to": "YYYY-MM-DD"},
  "positions":    {"raw": {...}, "normalized": {...}, "from": "YYYY-MM-DD", "to": "YYYY-MM-DD"},
  "startedAt": "ISO-8601",
  "finishedAt": "ISO-8601",
  "durationMs": 0,
  "dryRun": false,
  "force": false
}
```

### `GET /b3/client/sync-window`
**Query:** `cpf`, `type=transactions|positions`, `since?`, `end?`  
**Resposta (somente leitura):**
```json
{ "from": "YYYY-MM-DD", "to": "YYYY-MM-DD", "months": 0, "pagesEstimate": 0 }
```

---

## 🔁 Regras de Execução
1. **Validar** `X-Tenant-Id`, CPF (11), `dataTypes`, `assetTypes`, `since/end` (`YYYY-MM-DD`).  
2. **Lock** por `(tenantId, cpf)` para evitar execuções simultâneas.  
3. **Calcular** janela por tipo:  
   - `from` = `since` || `last_*_sync_at`;  
   - `to` = `endOverride` || `today()`.  
   - Se `from >= to`, retornar **200** com sumário vazio (nada a fazer).  
4. **Dividir** em meses e chamar **1.9** **apenas** para lacunas (usar cobertura de `b3_fetched_periods`); quando `force=true`, reconsultar tudo.  
5. **Se houver novos RAWs**, chamar **1.10** para normalizar **somente** o escopo novo.  
6. **Atualizar** `b3_sync_state` em sucesso; **não** avançar marcos em falha.  
7. **Marcar** `needsReprocess=true` quando qualquer tipo tiver novidade.  
8. **Resiliência**: retry com backoff+jitter (`429/5xx`), timeouts, e respeito a rate limit (1.6).

---

## 📊 Observabilidade
- **Logs estruturados** (JSON): `tenantId`, `cpfMasked`, `types`, `since`, `from`, `to`, `months`, `pages`, `saved/skipped/errors`, `durationMs`, `result`.  
- **Métricas Prometheus**:  
  - `b3_incremental_runs_total{result="success|error", type}`  
  - `b3_incremental_duration_seconds` (histogram)  
  - `b3_incremental_errors_total{stage="ingest|normalize"}`

---

## 🔐 Segurança & Multi‑tenant
- `POST /b3/admin/incremental-from-last` é **admin‑only**; exige **`X-Tenant-Id`** e RBAC.  
- `GET /b3/client/sync-window` exige **`X-Tenant-Id`**, porém é somente leitura (não altera estado).  
- **Mascarar CPF** em logs; **nunca** logar tokens/segredos.

---

## 🚦 Performance
- Processar por **meses** e **páginas** (mesma estratégia de 1.7/1.8/1.9).  
- Parâmetro `concurrency` para limitar execuções em paralelo (meses/páginas).  
- **Atenção** ao rate limit da B3; preferir **fila**/backoff quando alto volume.

---

## 🧪 Testes (obrigatório)
**Integration** — `src/Tests/Integration/B3/incremental_runner_test.go`  
- `GET /b3/client/sync-window` calcula janela correta a partir do estado.  
- `POST /b3/admin/incremental-from-last` com `dryRun=true` retorna **plano**; BD **inalterado**.  
- Execução real sem `since` usa `last_*_sync_at`; com `since`, respeita sobrescrita.  
- Em sucesso, `b3_sync_state` atualizado; quando há novidade, `needsReprocess=true`.  
- `force=true` reconsulta a janela; execução subsequente com `force=false` **não duplica**.

**Unit** — `src/Tests/Unit/B3/incremental/...`  
- Cálculo de janelas (`since` vs `last_*_sync_at`), composição do sumário, locks, fallback de datas e bordas (`from >= to`).

---

## 📄 Documentação
- **Swagger** (tag `B3 Sync`) com os 2 endpoints, parâmetros e códigos de resposta (200/400/401/403/429/5xx).  
- **Postman/Bruno**: coleções em `/docs/sync/` com exemplos reais (sanitizados).  
- `docs/b3/incremental.md` com fluxo, exemplos cURL e troubleshooting.

### cURL — exemplos
```bash
# Inspecionar janela (somente leitura)
curl "$BASE_URL/b3/client/sync-window?cpf=00000000000&type=transactions&since=2024-01-01" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"

# Executar incremental (admin), partindo do último marco
curl -X POST "$BASE_URL/b3/admin/incremental-from-last" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","dataTypes":["transactions","positions"],"assetTypes":["equity"],"dryRun":false,"force":false}'

# Executar incremental (admin) desde uma data específica
curl -X POST "$BASE_URL/b3/admin/incremental-from-last" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","dataTypes":["transactions"],"assetTypes":["equity"],"since":"2024-06-01","dryRun":false,"force":true}'
```

---

## 🔧 Configurações (.env sugeridas)
- `INCR_DEFAULT_CONCURRENCY=2`  
- `OBS_MASK_CPF=true`

---

## ✅ Entregáveis
- Controllers + service incremental + integração com 1.9/1.10/1.11.  
- Testes unitários e de integração.  
- Logs, métricas e RBAC.  
- Swagger + Postman/Bruno + docs atualizados.
