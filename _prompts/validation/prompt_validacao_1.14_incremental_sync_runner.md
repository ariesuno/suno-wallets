# Prompt de Validação 1.14v — Incremental Sync Runner (on‑demand) — **suno-wallets**

## 🎯 Objetivo
Validar que o **Incremental Sync Runner** executa **apenas lacunas** entre o último marco de sincronismo (1.11) e “hoje” (ou `since` informado), **reutilizando** 1.9 (RAW Persistence) e 1.10 (Normalization), com segurança multi‑tenant, observabilidade, idempotência e documentação completa.

> Pré‑requisitos concluídos: 1.6, 1.7/1.8, 1.9, 1.10, 1.11, 1.12.

---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos presentes e nas pastas certas:
  - `src/Application/B3/incremental/incremental_service.go`
  - `src/Api/Controllers/B3/admin_controller.go` (rota `POST /b3/admin/incremental-from-last`)
  - `src/Api/Controllers/B3/util_controller.go`  (rota `GET /b3/client/sync-window`)
- [ ] **Reuso obrigatório** de 1.9 (ingest RAW) e 1.10 (normalize) — **sem** duplicar lógicas.
- [ ] Nomes em **inglês**; comentários em **pt‑BR**; arquivos **coesos** (≤ ~300 linhas).

### 2) Contratos & Semântica
- **`POST /b3/admin/incremental-from-last` (admin‑only)**  
  - [ ] Body aceito:
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
  - [ ] `since` → janela `since → (endOverride||today)`; sem `since` → usar `last_*_sync_at` (1.11).
  - [ ] `dryRun=true` retorna **plano** (meses/páginas estimadas), **sem** gravar RAW/Normalized.
  - [ ] `force=true` reconsulta **toda a janela** calculada.
  - [ ] Resposta traz **sumário por tipo** com `{raw, normalized, from, to}` e tempos.
- **`GET /b3/client/sync-window` (somente leitura)**  
  - [ ] Query: `cpf`, `type=transactions|positions`, `since?`, `end?`.
  - [ ] Resposta: `{ "from": "...", "to": "...", "months": N, "pagesEstimate": M }`.
  - [ ] **Não** altera estado; **não** persiste.

### 3) Cálculo de Janela & Cobertura
- [ ] Janela por tipo: `from = since || last_*_sync_at`, `to = endOverride || today()`.
- [ ] Se `from >= to`, retorna **200** com sumário vazio (nada a fazer).
- [ ] Divisão em **meses**.
- [ ] Consulta **cobertura** (1.9 / `b3_fetched_periods`) para chamar a B3 **apenas** nas **lacunas** (quando `force=false`).

### 4) Execução & Estados (1.11)
- [ ] Em **sucesso**, atualizar `b3_sync_state.last_tx_sync_at` / `last_pos_sync_at` conforme tipo executado.
- [ ] Em **erro**, **não** avançar marcos; registrar `last_result`, `last_error`, `last_checked_at`, `failure_count++`.
- [ ] Quando houver **novos RAWs**, marcar `needsReprocess=true` para o CPF.

### 5) Idempotência & Reprocesso Parcial
- [ ] Reuso das garantias de idempotência do 1.9/1.10 (hash/unique/ upsert).
- [ ] Execuções subsequentes **sem `force`** **não** duplicam RAW/Normalized.
- [ ] Execução com `force=true` reconsulta a janela, mantendo idempotência na escrita.

### 6) Segurança & Multi‑tenant
- [ ] Ambos endpoints exigem **`X-Tenant-Id`**; `POST` é **admin‑only** (RBAC).
- [ ] **CPF mascarado** em logs; **nunca** logar tokens/segredos.
- [ ] Validação de CPF (11 dígitos), datas (`YYYY-MM-DD`), `dataTypes`/`assetTypes`.

### 7) Observabilidade
- [ ] **Logs estruturados** (JSON): `tenantId`, `cpfMasked`, `types`, `since`, `from`, `to`, `months`, `pages`, `saved/skipped/errors`, `durationMs`, `result`.
- [ ] **Métricas Prometheus** expostas/incrementadas:
  - `b3_incremental_runs_total{result="success|error", type}`
  - `b3_incremental_duration_seconds` (histogram)
  - `b3_incremental_errors_total{stage="ingest|normalize"}`
- [ ] Cardinalidade de labels controlada (sem `tenantId` como label nas métricas).

### 8) Resiliência & Performance
- [ ] Retries com **backoff + jitter** para `429/5xx`; timeouts e `context.Context` em todas as chamadas.
- [ ] **Lock** por `(tenantId, cpf)` durante a execução.
- [ ] Controle de **concorrência** (`concurrency`/config), dividido por meses/páginas.
- [ ] Respeito ao **rate limit** da B3 (política 1.6).

### 9) Documentação
- [ ] **Swagger** (tag `B3 Sync`) com os 2 endpoints, parâmetros, exemplos e erros 200/400/401/403/429/5xx.
- [ ] **Postman/Bruno** com requests prontos (sanitizados) em `/docs/sync/`.
- [ ] `docs/b3/incremental.md` com fluxo, janelas, cURLs e troubleshooting.

### 10) Testes
- **Integration —** `src/Tests/Integration/B3/incremental_runner_test.go`
  - [ ] `GET /b3/client/sync-window` calcula janela correta a partir de `b3_sync_state`.
  - [ ] `POST /b3/admin/incremental-from-last` com `dryRun=true` retorna **plano**; BD **inalterado**.
  - [ ] Execução real **sem `since`** usa `last_*_sync_at`; **com `since`** respeita sobrescrita.
  - [ ] Em sucesso, marcos atualizados; com novidade → `needsReprocess=true`.
  - [ ] `force=true` reconsulta janela; execução subsequente sem `force` **não** duplica.
- **Unit —** `src/Tests/Unit/B3/incremental/...`
  - [ ] Cálculo de janelas (`since` vs `last_*_sync_at`) e bordas (`from >= to`).
  - [ ] Composição de sumário, locking e fallback de datas.

---

## 🧪 Passo‑a‑passo (manual)
1. **Inspecionar janela (somente leitura):**
   ```bash
   curl "$BASE_URL/b3/client/sync-window?cpf=00000000000&type=transactions&since=2024-01-01" \
     -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
   ```
2. **Incremental a partir do último marco:**
   ```bash
   curl -X POST "$BASE_URL/b3/admin/incremental-from-last" \
     -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
     -d '{"cpf":"00000000000","dataTypes":["transactions","positions"],"assetTypes":["equity"],"dryRun":false,"force":false}'
   ```
3. **Incremental desde `since` específico (com force):**
   ```bash
   curl -X POST "$BASE_URL/b3/admin/incremental-from-last" \
     -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
     -d '{"cpf":"00000000000","dataTypes":["transactions"],"assetTypes":["equity"],"since":"2024-06-01","dryRun":false,"force":true}'
   ```

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.14 (Incremental Sync Runner) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
