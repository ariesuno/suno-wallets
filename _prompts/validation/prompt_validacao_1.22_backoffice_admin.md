# Prompt de Validação 1.22v — Backoffice Admin (Painel Operacional) — **suno-wallets**

## 🎯 Objetivo
Validar que o **Backoffice Admin (APIs)** oferece **busca**, **perfil 360** do cliente, **ações operacionais com dry‑run e dupla confirmação**, **auditoria completa**, **exports**, **RBAC rígido**, **observabilidade** e **performance**, integrando módulos 1.11/1.14/1.17/1.18/1.19/1.20/1.21.  
Sem dados fake; multi‑tenant estrito; LGPD aplicada (mascarar CPF em logs/respostas públicas).

> Escopo: services/repos/controllers do backoffice, migrations (`admin_action_audit`, `client_operational_checkpoints` opcional), integração com jobs/serviços existentes, Swagger/Collections, testes unitários e de integração.


---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos criados com **nomes em inglês** e **comentários pt‑BR**:
  - `src/Application/Admin/backoffice_query_service.go`
  - `src/Application/Admin/backoffice_actions_service.go`
  - `src/Infrastructure/Admin/backoffice_repository.go`
  - `src/Api/Controllers/Admin/backoffice_controller.go`
  - Testes: `src/Tests/Unit/Admin/...`, `src/Tests/Integration/Admin/...`
- [ ] Controllers apenas **orquestram**; regras no **Service**; acesso a BD via **Repository**.
- [ ] Ações perigosas usam **lock por (tenantId, cpf)** e são **idempotentes**.

### 2) Banco & Migrations
- [ ] `admin_action_audit` criado com enums `admin_action_type` e `admin_action_status` e colunas exigidas (`confirm_token`, `confirm_deadline`, `request_payload`, `result_payload`, etc.).
- [ ] Índice `ix_admin_action_lookup` presente.
- [ ] (Opcional) `client_operational_checkpoints` criado quando necessário.
- [ ] Constraints e **FKs lógicas** documentadas (IDs referenciados em payloads).

### 3) Endpoints (Swagger tag: `Backoffice Admin`)
**Search & Profile**
- [ ] `GET /admin/backoffice/search?query=` retorna lista com **CPF mascarado** e metadados (policy, últimas ingestões).
- [ ] `GET /admin/backoffice/profile?cpf=` retorna visão 360 coerente (b3, reconciliation, performance, lastActions).

**Actions (padrão 2 passos)**  
Fluxo: `POST /admin/backoffice/actions/<action>` → cria **REQUESTED** (com `confirmToken`) → `POST /.../<action>/confirm` executa (**CONFIRMED** → **RUNNING** → **SUCCESS|ERROR|SKIPPED**).
- [ ] `b3-full-fetch` aciona 1.11 (aceita `from/to`, `dryRun`).
- [ ] `b3-incremental` aciona 1.14 (aceita `date?`, `dryRun`).
- [ ] `recon-scan` aciona 1.17.
- [ ] `auto-fix` aciona 1.18.
- [ ] `dedup-scan` e `dedup-resolve` acionam 1.19.
- [ ] `policy-set` atalho para 1.21 com auditoria integrada.
- [ ] `client-reset` e `zero-and-refetch` exigem **dupla confirmação** sempre (sem confirm não executa).

**Audit & Export**
- [ ] `GET /admin/backoffice/actions` lista paginada, filtros por `action,status,cpf`.
- [ ] `GET /admin/backoffice/actions/{id}` detalha auditoria (payloads, tempos, erro).
- [ ] `GET /admin/backoffice/export/ledger?cpf=&format=csv|json&limit=` exporta respeitando policy e limites.

**Códigos**: 200/201/202/204, 400/401/403/404/409/429/5xx com mensagens claras.  
**Swagger/Postman/Bruno** atualizados em `/docs/admin/` (folders por ação: dry‑run e confirm).

### 4) Segurança, RBAC & LGPD
- [ ] Header **`X-Tenant-Id` obrigatório** em todas as rotas.
- [ ] Roles: `ADMIN` (executa tudo), `SUPPORT` (consulta e ações não destrutivas/dry‑run), `AUDITOR` (somente leitura).
- [ ] **CPF mascarado** nas respostas de `search`; CPF completo apenas quando estritamente necessário (admin).
- [ ] **Rate‑limit** por ação e cotas por ator configuradas.
- [ ] Nenhum token/segredo exposto em logs/respostas.

### 5) Observabilidade
- [ ] **Logs** estruturados (JSON): `tenantId`, `actor`, `cpfMasked`, `action`, `status`, `durationMs`, `dryRun`, `resultCounts`/IDs.
- [ ] **Métricas Prometheus**:
  - `admin_actions_total{action,status}`
  - `admin_actions_duration_seconds{action}` (histogram)
  - `admin_exports_total{format}`
  - `admin_profile_requests_total{result}`
- [ ] Alertas definidos para picos de `ERROR` e uso anômalo de `zero-and-refetch`.

### 6) Performance & Concorrência
- [ ] Locks de execução por `(tenantId, cpf)` funcionando (sem corrida com jobs 1.11/1.14).
- [ ] Exports em **streaming**, respeitando **cap** de linhas/tempo.
- [ ] Índices utilizados nas consultas (ledger, inconsistências, audit); `EXPLAIN ANALYZE` sem *seq scans* desnecessários.

### 7) Integrações
- [ ] Ações realmente **orquestram** os módulos 1.11/1.14/1.17/1.18/1.19/1.21 e retornam **sumários** consistentes.
- [ ] `policy-set` invalida cache (1.21) e registra **audit**.
- [ ] `zero-and-refetch` zera ledger normalizado do CPF e dispara 1.11 (full) com auditoria.

### 8) Testes Automatizados
**Integration — `src/Tests/Integration/Admin/backoffice_admin_test.go`**
- [ ] `search` retorna dados mascarados e metadados corretos.  
- [ ] `profile` agrega dados coerentes com ledger/inconsistências/policy/checkpoints.  
- [ ] Fluxo 2 passos: **REQUESTED → CONFIRMED → RUNNING → SUCCESS** (com `confirmToken` válido).  
- [ ] **Dry‑run** não altera estado; **confirm** executa e preenche audit.  
- [ ] **Dupla confirmação** obrigatória em `client-reset`/`zero-and-refetch`; sem confirm → não executa.  
- [ ] **Idempotência**: confirmar duas vezes não duplica efeitos.  
- [ ] RBAC: `SUPPORT` bloqueado para ações perigosas com 403.

**Unit**
- [ ] Geração/validação de `confirmToken` + TTL; máscaras de CPF; serialização de payloads; agregação do `profile`.

### 9) Documentação
- [ ] Swagger completo (exemplos sanitizados) e coleções Postman/Bruno versionadas.  
- [ ] `docs/admin/operations_guide.md` explica o que observar no Grafana/Prometheus por ação.

---

## 🧪 Passos de Validação Manual (rápidos)

1) **Search**  
```bash
curl "$BASE_URL/admin/backoffice/search?query=00000000000" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

2) **Profile**  
```bash
curl "$BASE_URL/admin/backoffice/profile?cpf=00000000000" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

3) **Ação 2 passos (ex.: b3-full-fetch)**  
```bash
# Passo 1 - REQUESTED
curl -X POST "$BASE_URL/admin/backoffice/actions/b3-full-fetch" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","from":"2020-01","to":"2020-12","dryRun":true}'

# Passo 2 - CONFIRMED (usar confirmToken do passo 1)
curl -X POST "$BASE_URL/admin/backoffice/actions/b3-full-fetch/confirm" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"confirmToken":"<token>"}'
```

4) **Export ledger**  
```bash
curl -D headers.txt -o ledger.csv "$BASE_URL/admin/backoffice/export/ledger?cpf=00000000000&format=csv&limit=10000" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

5) **Audit trail**  
```bash
curl "$BASE_URL/admin/backoffice/actions?cpf=00000000000&status=SUCCESS" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.22 (Backoffice Admin) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
