# Prompt de Validação 1.19v — Manual Ops + Dedup & Alertas — **suno-wallets**

## 🎯 Objetivo
Validar que o **CRUD de operações manuais** e o **mecanismo de deduplicação B3×Manual** foram implementados corretamente, com **segurança multi‑tenant**, **idempotência**, **explicabilidade**, **observabilidade**, e contratos **Swagger/Postman/Bruno** alinhados. Sem alterar RAW/Normalized; toda escrita ocorre no **ledger unificado**.

> Escopo: `USER_MANUAL` no `b3_operations_ledger`, `dedup_candidates`, política por cliente (1.21 hook), endpoints **Manual Ops** e **Ops Dedup**, heurística/score, decisões `MERGE|OVERRIDE|IGNORE` com auditoria.

---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos com **nomes em inglês** e **comentários pt‑BR**:
  - `src/Application/Ops/manual_operations_service.go`
  - `src/Application/Ops/dedup_service.go`
  - `src/Infrastructure/Ops/operations_repository.go`
  - `src/Infrastructure/Ops/dedup_repository.go`
  - `src/Api/Controllers/Ops/manual_operations_controller.go`
  - `src/Api/Controllers/Ops/dedup_controller.go`
  - Testes: `src/Tests/Unit/Ops/...` e `src/Tests/Integration/Ops/...`
- [ ] Controllers só orquestram; regras no Service; acesso a BD via Repository.
- [ ] Ledger `b3_operations_ledger` **reutilizado** (1.18) e **nunca** escrever/alterar `B3_RAW` diretamente.

### 2) Migrations & Esquema
- [ ] `b3_operations_ledger` possui colunas/índices exigidos (1.18) e aceita `source='USER_MANUAL'`.
- [ ] Tabela `dedup_candidates` criada com:
  - `status ENUM('OPEN','AUTO_MERGED','OVERRIDDEN','IGNORED','CONFIRMED_MERGE')`
  - `rationale JSONB`, `pair_key`, `dedupe_key`
  - Índices: `uq_dedup_pair` (unique), `ix_dedup_lookup`
- [ ] (Opcional) `dedup_policies` com thresholds/tolerâncias por CPF/tenant.

### 3) Política por Cliente (hook 1.21)
- [ ] `B3_ONLY` → `POST /ops/manual` retorna **409** com mensagem clara.
- [ ] `MANUAL_ONLY` → CRUD permitido; dedup desabilita geração de candidatos com `B3_RAW` (não há base para comparar).
- [ ] `HYBRID` → ambos permitidos + dedup ativo.

### 4) Dedup — Heurística, Idempotência e Decisão
- [ ] Pré‑filtros consideram **asset_type** e **canonical_ticker**.
- [ ] Janela de data `± date_tolerance_days` aplicada.
- [ ] Score combina: **quantity**, **gross**, **side**, **extras** (broker/order).
- [ ] `pair_key`/`dedupe_key` estáveis (idempotência); reexecutar **scan** não duplica candidatos.
- [ ] Decisões:
  - `score ≥ auto_merge_threshold` → **AUTO_MERGED** (aplica **MERGE** automaticamente).
  - `alert_threshold ≤ score < auto_merge_threshold` → **OPEN** (alerta aguardando ação).
  - `< alert_threshold` → **não** gera candidato.
- [ ] **Explainability**: `rationale` armazena features, pesos e tolerâncias usadas.

### 5) Endpoints & Contratos (Swagger tags: `Manual Ops`, `Ops Dedup`)
**Manual Ops**
- [ ] `POST /ops/manual` cria USER_MANUAL (valida política; required: `cpf, ticker, assetType, operationDate, operationType, quantity`).  
- [ ] `PUT /ops/manual/{id}` atualiza **apenas** USER_MANUAL ativa (não troca `source`).  
- [ ] `DELETE /ops/manual/{id}` realiza **soft delete** (`is_active=false`).  
- [ ] `GET /ops/manual` lista paginada com filtros.

**Ops Dedup**
- [ ] `POST /ops/dedup/scan` *(admin‑only)* gera candidatos (suporta `dryRun`).  
- [ ] `GET /ops/dedup/candidates` filtra por `status`, `scoreRange`, `ticker`, `cpf`.  
- [ ] `POST /ops/dedup/resolve` aplica `MERGE|OVERRIDE|IGNORE` em lote; registra auditoria.

**Códigos esperados**: 200/201/204 e 400/401/403/404/409/5xx com mensagens coerentes.  
**Swagger/Postman/Bruno** atualizados em `/docs/ops/`.

### 6) Semântica de Resolução
- **MERGE**: mantém **uma** operação ativa (preferência por policy ou `prefer` no body); a outra vira `is_active=false` com `superseded_by_operation_id`.  
- **OVERRIDE**: força prevalência de **USER_MANUAL** (marca correspondente B3_RAW como inativa **no ledger**; não toca no RAW).  
- **IGNORE**: marca candidato como `IGNORED`; não reapresenta em novas varreduras.

### 7) Segurança & Multi‑tenant
- [ ] **`X-Tenant-Id` obrigatório**; RBAC: CRUD do usuário/operador; `scan/resolve` **admin**.  
- [ ] **CPF mascarado** em logs; nenhum token/segredo em logs.  
- [ ] Todas as queries escrevem **apenas** dentro do tenant do header.

### 8) Observabilidade
- [ ] Logs (JSON): `tenantId`, `cpfMasked`, `opId`, `action`, `score`, `decision`, `durationMs`.  
- [ ] Métricas Prometheus:
  - `ops_manual_writes_total{result}`
  - `ops_dedup_scan_runs_total{result}`
  - `ops_dedup_candidates_total{status}`
  - `ops_dedup_resolutions_total{action}`
  - `ops_dedup_duration_seconds` (histogram)
- [ ] Baixa cardinalidade — CPF só nos logs.

### 9) Performance
- [ ] Índices no ledger: `(tenant_id, cpf, ticker, operation_date)` e `(tenant_id, cpf, source, is_active)` presentes e utilizados.  
- [ ] `EXPLAIN ANALYZE` nos principais SELECTs/UPDATEs do dedup e listagens.  
- [ ] Varredura por **lotes** (janelas mensais) em `scan` e paginação consistente.

### 10) Testes Automatizados
**Integration — `src/Tests/Integration/Ops/manual_and_dedup_test.go`**
- [ ] CRUD de USER_MANUAL completo (create/update/soft delete/list).  
- [ ] `HYBRID`: criar USER_MANUAL semelhante a B3_RAW → `scan` gera candidato com `score` alto; `MERGE` inativa uma; `OVERRIDE` prevalece manual; `IGNORE` não reaparece.  
- [ ] Idempotência: reexecutar `scan` não duplica; `resolve` repetido mantém consistência.  
- [ ] `B3_ONLY` rejeita `POST /ops/manual` com 409.

**Unit**
- [ ] Cálculo de `score` (features/pesos/tolerâncias) e estabilidade de `pair_key/dedupe_key`.  
- [ ] Regras de `MERGE/OVERRIDE/IGNORE`, auditoria e efeitos no ledger.

### 11) Documentação
- [ ] Swagger (duas tags) com exemplos realistas (sanitizados).  
- [ ] Coleções **Postman/Bruno** versionadas.  
- [ ] `docs/ops/dedup.md` explica a heurística, thresholds e `rationale` com exemplos.

---

## 🧪 Passos de Validação Manual (rápido)

1) **Criar operação manual**  
```bash
curl -X POST "$BASE_URL/ops/manual" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","ticker":"ITSA4","assetType":"EQUITY","operationDate":"2024-06-10","operationType":"BUY","quantity":"100","unitPrice":"10.00"}'
```
**Esperado**: 201, operação `USER_MANUAL` criada.

2) **Scan de dedup (HYBRID)**  
```bash
curl -X POST "$BASE_URL/ops/dedup/scan" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","scanMode":"BATCH"}'
```
**Esperado**: 200, candidatos criados; sem duplicação em reexecução.

3) **Resolver — MERGE**  
```bash
curl -X POST "$BASE_URL/ops/dedup/resolve" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"action":"MERGE","candidateIds":["<id>"]}'
```
**Esperado**: 200, uma operação ativa e a outra `is_active=false` com `superseded_by_operation_id` setado.

4) **Resolver — OVERRIDE**  
Aciona prevalência do `USER_MANUAL`; ver registros no ledger.

5) **Resolver — IGNORE**  
Marca candidato como `IGNORED`; reexecutar `scan` não reabre o mesmo par.

6) **Política `B3_ONLY`**  
`POST /ops/manual` → **409** com explicação.

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.19 (Manual Ops + Dedup & Alertas) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
