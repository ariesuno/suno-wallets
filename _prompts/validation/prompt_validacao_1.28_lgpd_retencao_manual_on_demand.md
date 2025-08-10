# Prompt de Validação 1.28v — LGPD & Retenção Manual (on‑demand) — **suno-wallets**

## 🎯 Objetivo
Validar que a solução de **LGPD (exportação e eliminação manual por CPF)** está implementada com **dry‑run**, **dupla confirmação**, **recibo de eliminação**, **tombstone/blocklist**, **multi‑tenant estrito**, **sem PII em logs/métricas**, **observabilidade completa** e **testes automatizados**.  
**Nunca** executar purga automática do RAW; somente quando **explicitamente** solicitado e se `include_raw=true`.

> Escopo: controllers/serviços/repositórios LGPD, migrations (`lgpd_erasure_runs`, `lgpd_erasure_counts`, `lgpd_erasure_receipts`, `lgpd_subject_tombstones`, `lgpd_ingestion_blocklist`), empacotador de export, batch delete idempotente, Swagger/Collections, métricas Prometheus, dashboards/alertas e testes (unit/integration).


---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos com nomes em **inglês** e **comentários pt‑BR**:
  - `src/Domain/LGPD/lgpd_models.go` (ErasureRequest/Plan/Receipt/SubjectExport/RetentionRule)
  - `src/Application/LGPD/lgpd_service.go` (preview/request/confirm/apply/export/listRuns/lookup/dryRunRetention)
  - `src/Infrastructure/LGPD/lgpd_repository.go` (queries, batch delete, receipts, tombstones, blocklist)
  - `src/Api/Controllers/LGPD/lgpd_admin_controller.go` (rotas Admin)
  - Testes: `src/Tests/Unit/LGPD/...`, `src/Tests/Integration/LGPD/...`
- [ ] Separação de responsabilidades (Controller → Service → Repository) e **idempotência** no Service para `apply`.

### 2) Banco & Migrations
- [ ] Tabelas criadas conforme o prompt 1.28:
  - `lgpd_erasure_runs` com `mode` (DRY_RUN|REQUESTED|CONFIRMED|RUNNING|SUCCESS|ERROR|CANCELED), `confirm_token`, `export_path`, `include_raw`, `cpf_hash` (sem CPF puro).
  - `lgpd_erasure_counts` (contagens por tabela, `matched` e `deleted`).
  - `lgpd_erasure_receipts` (resumo JSONB, `cpf_hash`, `run_id`).
  - `lgpd_subject_tombstones` (PK `(tenant_id, cpf_hash)`).
  - `lgpd_ingestion_blocklist` (PK `(tenant_id, cpf_hash, source)`).
- [ ] Índices presentes conforme especificação; **nenhum CPF puro** gravado.
- [ ] `cpf_hash = SHA256(tenant_salt || cpf)` — **tenant_salt não está no código** (somente em env).

### 3) Fluxo & Regras
- [ ] `lookup` retorna **volumetria por domínio** (RAW/ledger/manual/system), status ativo/inativo, presença em tombstone/blocklist.
- [ ] `preview` (dry‑run) **não apaga** nada e persiste `DRY_RUN` + `lgpd_erasure_counts` por tabela.
- [ ] `export` gera um **ZIP** com `manifest.json` (versões, horários, tenant), arquivos por tabela e **não** contém tokens/segredos. Caminho salvo em `export_path`.
- [ ] `request` cria *run* `REQUESTED` com `confirm_token`.
- [ ] `confirm` exige `confirm_token` válido e transiciona para `CONFIRMED`.
- [ ] `apply`:
  - Executa **batch delete** por tabela, ordem segura, com **checkpoint/cursor** e **idempotência** por `(run_id, table)`.
  - Respeita `include_raw` (não apagar RAW quando `false`).
  - Insere em **tombstone** e **blocklist=BOTH** ao finalizar com sucesso.
  - Gera **receipt** com contagens resumidas.
- [ ] **Política Manual de Inatividade**: apenas `retention/dry-run` para listar candidatos; **aplicação sempre manual** one‑by‑one via `request/confirm/apply`.
- [ ] **Nunca** há purga automática do RAW.

### 4) Endpoints (Swagger tag: `LGPD — Admin`)
- [ ] `GET /admin/lgpd/subject/lookup?cpf=`
- [ ] `POST /admin/lgpd/erasure/preview`
- [ ] `POST /admin/lgpd/export`
- [ ] `POST /admin/lgpd/erasure/request`
- [ ] `POST /admin/lgpd/erasure/confirm`
- [ ] `POST /admin/lgpd/erasure/apply`
- [ ] `GET /admin/lgpd/erasure/status/{id}`
- [ ] `GET /admin/lgpd/erasure/runs`
- [ ] `POST /admin/lgpd/retention/dry-run`
- [ ] Códigos: 200/201/202/204 e 400/401/403/404/409/422/5xx com mensagens claras.
- [ ] **Swagger/Postman/Bruno** versionados em `/docs/lgpd/` com exemplos sanitizados.

### 5) Segurança, LGPD & Multi‑tenant
- [ ] **`X-Tenant-Id` obrigatório** em todas as rotas; **RBAC**: `ADMIN` (request/confirm/apply/export), `SUPPORT` (lookup/preview/status/runs).
- [ ] **Nunca** logar CPF puro ou tokens/segredos; logs usam `cpfMasked="***1234"` e `cpf_hash` quando necessário.
- [ ] Respostas e exports **não** expõem CPF completo — quando o CPF é necessário no pacote, apresentar **mascarado** e manter `cpf_hash` para referência.
- [ ] `tenant_salt` vem de **.env**, **rotacionável**, e não aparece em logs/métricas.

### 6) Observabilidade
**Métricas Prometheus**
- [ ] `lgpd_erasure_runs_total{status}`
- [ ] `lgpd_erasure_duration_seconds` (histogram)
- [ ] `lgpd_erasure_deleted_rows_total{table}`
- [ ] `lgpd_exports_total{format="zip"}`
- [ ] `lgpd_blocklist_gauge{source}`
**Logs (JSON)**
- [ ] Campos: `tenantId`, `runId`, `actor`, `mode`, `reason`, `includeRaw`, `counts`, `durationMs`, `error?`, `cpfMasked`.
- [ ] **Sem PII** em métricas/labels; CPF somente mascarado em logs.
**Alertas sugeridos**
- [ ] `lgpd_erasure_error_rate` alto em 30m dispara alerta.
- [ ] `lgpd_erasure_duration_seconds` p95 acima do limite por 15m dispara alerta.

### 7) Performance & Confiabilidade
- [ ] *Batch delete* em lotes (ex.: 5k) com transações curtas; evitar *locks* longos.
- [ ] Índices adequados nas tabelas alvo; *EXPLAIN ANALYZE* sem *seq scans* pesados.
- [ ] Worker dedicado com **concorrência controlada por `(tenantId, cpf_hash)`**.
- [ ] `apply` é **idempotente** (reexecução não duplica nem falha sem necessidade).

### 8) Integrações
- [ ] Backoffice 1.22 possui entradas/links para **lookup**, **preview**, **status/runs** e **download de export**, com **dupla confirmação** textual antes de acionar `/confirm`/`/apply`.
- [ ] Ingestão B3/manual respeita **blocklist/tombstone** (sem reprocessar o CPF).

### 9) Testes Automatizados
**Integration — `src/Tests/Integration/LGPD/lgpd_on_demand_test.go`**
- [ ] `lookup` retorna volumes coerentes + flags de tombstone/blocklist.
- [ ] `preview` persiste `DRY_RUN` + `lgpd_erasure_counts`.
- [ ] `export` gera ZIP válido, com `manifest.json`, caminho persistido e **sem segredos**.
- [ ] `request` → gera `confirm_token`; `confirm` muda estado para `CONFIRMED`.
- [ ] `apply` apaga linhas corretas (ledger/manual/system/CA/inconsistências), respeitando `include_raw`; insere tombstone/blocklist; cria **receipt**; **idempotência** ok.
- [ ] `retention/dry-run` lista candidatos; **nenhuma** exclusão aplicada.
- [ ] RBAC: `SUPPORT` não consegue aplicar (403); `X-Tenant-Id` ausente → 400/401/403.
**Unit**
- [ ] Hash salgado do CPF; geração/validação de `confirm_token`; mascaramento em logs; manifesto do export; *cursor batching*.


---

## 🧪 Passos de Validação Manual (rápidos)

1) **Lookup**
```bash
curl "$BASE_URL/admin/lgpd/subject/lookup?cpf=00000000000" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

2) **Preview (dry‑run)**
```bash
curl -X POST "$BASE_URL/admin/lgpd/erasure/preview" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","includeRaw":false}'
```

3) **Export**
```bash
curl -X POST "$BASE_URL/admin/lgpd/export" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000"}'
```

4) **Request → Confirm → Apply**
```bash
RUN=$(curl -s -X POST "$BASE_URL/admin/lgpd/erasure/request" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","reason":"USER_REQUEST","includeRaw":true}' | jq -r '.id,.confirmToken')

# Confirm
curl -X POST "$BASE_URL/admin/lgpd/erasure/confirm" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"runId":"<RUN_ID>","confirmToken":"<CONFIRM_TOKEN>"}'

# Apply
curl -X POST "$BASE_URL/admin/lgpd/erasure/apply" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"runId":"<RUN_ID>"}'
```

5) **Status / Runs**
```bash
curl "$BASE_URL/admin/lgpd/erasure/status/<RUN_ID>" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
curl "$BASE_URL/admin/lgpd/erasure/runs?cursor=&limit=20" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

6) **Retention Dry‑run**
```bash
curl -X POST "$BASE_URL/admin/lgpd/retention/dry-run" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"inactiveMonths":24,"includeRaw":false}'
```


---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.28 (LGPD & Retenção Manual — on‑demand) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
