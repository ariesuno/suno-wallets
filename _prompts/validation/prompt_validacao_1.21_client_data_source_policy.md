# Prompt de Validação 1.21v — Client Data Source Policy (B3_ONLY | MANUAL_ONLY | HYBRID) — **suno-wallets**

## 🎯 Objetivo
Confirmar que a **política de fonte de dados por cliente** está implementada, cacheada e aplicada de forma consistente em jobs, CRUD de operações manuais, dedup e leitura (timeline/summary), com **segurança**, **observabilidade**, **idempotência** e **documentação** adequadas.

> Escopo: tabelas `client_data_source_policy` e `client_data_source_policy_audit`, enforcers/hookpoints (1.11, 1.14, 1.19, 1.20), controller/admin endpoints, cache com TTL, logs/métricas, Swagger/Collections e testes automatizados.


---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos presentes (nomes em inglês, comentários pt‑BR):
  - `src/Domain/ClientPolicy/client_policy.go`
  - `src/Application/ClientPolicy/client_policy_service.go`
  - `src/Infrastructure/ClientPolicy/client_policy_repository.go`
  - `src/Infrastructure/ClientPolicy/client_policy_cache.go`
  - `src/Api/Controllers/ClientPolicy/client_policy_controller.go`
  - Enforcers:
    - `src/Application/B3/ingest/ingest_enforcer.go` (1.11/1.14)
    - `src/Application/Ops/manual_ops_enforcer.go` (1.19)
    - `src/Application/Ops/read_enforcer.go` (1.20)
  - Testes: `src/Tests/Unit/ClientPolicy/...`, `src/Tests/Integration/ClientPolicy/...`
- [ ] Controllers apenas orquestram; regras no Service; acesso a BD via Repository.
- [ ] Cache in‑memory + interface para backend externo (ex.: Redis) definida.

### 2) Banco & Migrations
- [ ] Tabelas criadas conforme especificação:
  - `client_data_source_policy (mode ENUM, reason, effective_from/to, audit fields)`
  - `client_data_source_policy_audit (old_mode, new_mode, changed_by, reason)`
- [ ] Índices presentes: `ix_client_policy_lookup`, `ix_client_policy_audit`.
- [ ] Restrições: `UNIQUE (tenant_id, cpf)` na tabela principal.

### 3) Endpoints (Swagger tag: `Client Policy`)
- [ ] `GET /admin/client/policy?cpf=` retorna policy atual + metadados.
- [ ] `POST /admin/client/policy` upsert:
  ```json
  { "cpf":"00000000000", "mode":"B3_ONLY|MANUAL_ONLY|HYBRID", "reason":"..." }
  ```
  - [ ] Grava em `client_data_source_policy` **e** em `..._audit`.
  - [ ] **Invalida** cache.
- [ ] `POST /admin/client/policy/dry-run` retorna efeitos esperados: `affectedJobs`, `blockedEndpoints`, `readSideChanges`.
- [ ] `GET /admin/client/policy/audit?cpf=` lista histórico.
- [ ] Códigos: 200/201/400/401/403/409/5xx com mensagens claras.
- [ ] Swagger/Postman/Bruno atualizados em `/docs/client-policy/`.

### 4) Enforce por modo
- **B3_ONLY**
  - [ ] Jobs 1.11/1.14 executam normalmente.
  - [ ] `POST/PUT/DELETE /ops/manual` retorna **409** (`POLICY_VIOLATION_MANUAL_DISABLED`).
  - [ ] Dedup 1.19 **não** gera candidatos entre B3_RAW×Manual (sem base manual ativa).
- **MANUAL_ONLY**
  - [ ] Jobs 1.11/1.14 são **skipados** com métrica `policy_skipped_total{job="B3_*"}`.
  - [ ] Leitura/timeline/summary **ignora B3_RAW** quando `respectPolicy=true` (default).
  - [ ] CRUD USER_MANUAL permitido.
- **HYBRID**
  - [ ] Tudo permitido; Dedup ativo.
- [ ] Enforcers aplicam **multi‑tenant** e retornam erros consistentes.

### 5) Cache & Invalidação
- [ ] TTL configurável (`CLIENT_POLICY_TTL_SECONDS`); hit/miss métricas.
- [ ] On‑write: `Invalidate(tenant, cpf)` é chamado; próxima leitura reflete novo modo.
- [ ] Fallback seguro (cache indisponível → leitura do repositório).

### 6) Segurança & LGPD
- [ ] **`X-Tenant-Id` obrigatório**; RBAC: writes/dry‑run somente **admin**, leitura admin/suporte.
- [ ] CPF **mascarado** nos logs (`***1234`); sem tokens/segredos em logs.
- [ ] Validações: `cpf` 11 dígitos; `mode` dentro do ENUM; `reason` ≤ 500 chars.

### 7) Observabilidade
- [ ] Logs estruturados (JSON): `tenantId`, `cpfMasked`, `mode`, `actor`, `action`, `durationMs`, `result`.
- [ ] Métricas Prometheus:
  - `client_policy_reads_total{source="cache|db",result}`
  - `client_policy_writes_total{result}`
  - `client_policy_cache_invalidations_total`
  - `policy_skipped_total{job}`
- [ ] Baixa cardinalidade (evitar labels com CPF).

### 8) Performance
- [ ] Índices suportam consultas por `(tenant_id, cpf)`.
- [ ] Cache reduz latência em endpoints/jobs quentes (comparar p50/p95 opcional).

### 9) Testes Automatizados
**Integration — `src/Tests/Integration/ClientPolicy/client_policy_test.go`**
- [ ] Upsert de policy cria/atualiza + grava audit + invalida cache.
- [ ] Trocar para `B3_ONLY` → `POST /ops/manual` retorna **409**.
- [ ] Trocar para `MANUAL_ONLY` → job 1.14 retorna **skip** e incrementa `policy_skipped_total`.
- [ ] Timeline/summary com `respectPolicy=true` não retorna B3_RAW em `MANUAL_ONLY`.
- [ ] Dry‑run lista efeitos corretos.

**Unit**
- [ ] Cache (hit/miss/ttl/invalidate); validações de entrada; enforcers isolados.

### 10) Documentação
- [ ] Swagger e coleções **Postman/Bruno** atualizados.
- [ ] `docs/client_policy/overview.md` descreve os modos, enforce, cache e exemplos de respostas.

---

## 🧪 Passos de Validação Manual (rápidos)

1) **Criar/alterar policy (admin)**  
```bash
curl -X POST "$BASE_URL/admin/client/policy" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","mode":"B3_ONLY","reason":"experimento A"}'
```
**Esperado**: 201; audit gravado; cache invalidado.

2) **Dry‑run**  
```bash
curl -X POST "$BASE_URL/admin/client/policy/dry-run" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","newMode":"MANUAL_ONLY"}'
```
**Esperado**: 200 com `affectedJobs`, `blockedEndpoints`, `readSideChanges` coerentes.

3) **Efeito em Manual Ops**  
Com `mode=B3_ONLY`:
```bash
curl -X POST "$BASE_URL/ops/manual" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","ticker":"ITSA4","assetType":"EQUITY","operationDate":"2024-06-10","operationType":"BUY","quantity":"10"}'
```
**Esperado**: **409** `POLICY_VIOLATION_MANUAL_DISABLED`.

4) **Efeito em B3 Jobs**  
Com `mode=MANUAL_ONLY`: disparar job 1.14.  
**Esperado**: **skip** + métrica `policy_skipped_total` incrementada.

5) **Leitura com respeito à policy**  
Com `mode=MANUAL_ONLY`:
```bash
curl "$BASE_URL/ops/timeline?cpf=00000000000" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```
**Esperado**: itens **B3_RAW** **não** retornam (quando `respectPolicy=true`).

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.21 (Client Data Source Policy) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
