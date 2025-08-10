# Prompt de Validação 1.17v — Inconsistency Detector (read-only) — **suno-wallets**

## 🎯 Objetivo
Validar que o **detector de inconsistências** (1.17) identifica corretamente casos pré‑API e divergências entre **posições v3** e **transações normalizadas**, **sem** modificar ledger/operções, persistindo achados de forma **idempotente** em `b3_inconsistencies`, com **observabilidade**, **segurança** e **documentação** completas.

> Escopo: regras `OPENING_BALANCE_MISSING`, `SELL_WITHOUT_BUY`, `POSITION_TX_DIVERGENCE`; endpoints de scan/leitura; migrations; testes unitários/integrados.

---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Pastas/arquivos existem com **nomes em inglês** e **comentários pt‑BR**:
  - `src/Application/B3/reconciliation/inconsistency_detector_service.go`
  - `src/Infrastructure/B3/reconciliation/inconsistency_repository.go`
  - `src/Api/Controllers/B3/reconciliation_controller.go`
  - Testes: `src/Tests/Unit/B3/reconciliation/...` e `src/Tests/Integration/B3/reconciliation/...`
- [ ] **Somente leitura** sobre RAW/Normalized; **nenhuma** escrita em ledger/operations nesta etapa.
- [ ] Locks por `(tenantId, cpf)` para evitar corrida com 1.13/1.14.

### 2) Migrations & Esquema
- [ ] Tipos `inconsistency_type` e `inconsistency_status` criados conforme especificação.
- [ ] Tabela `b3_inconsistencies` criada com colunas exigidas (`severity`, `sample_dates`, `details`, `dedupe_hash`, auditoria).
- [ ] Índices:
  - [ ] `uq_b3_inconsistencies_dedupe` (unique por `tenant_id, cpf, ticker, type, dedupe_hash`).
  - [ ] `ix_b3_inconsistencies_lookup` (consulta por `tenant_id, cpf, status, type`).

### 3) Regras de Detecção
- [ ] `OPENING_BALANCE_MISSING`: compara **primeira foto** de posição vs **acúmulo de buys-sells** até a data.
- [ ] `SELL_WITHOUT_BUY`: primeira transação = `SELL` **ou** cumulativo líquido negativo em algum ponto.
- [ ] `POSITION_TX_DIVERGENCE`: para `reference_date`s disponíveis, posição ≠ acumulado líquido; amostragem (top‑N) registrada.
- [ ] Usa `B3_API_EARLIEST_DATE` (env) e filtra sempre por `tenant_id` + `cpf`.
- [ ] `dedupe_hash` estável (sha256 com campos de assinatura) para **idempotência**.
- [ ] Parâmetros configuráveis: `maxSamplesPerType`, janela (`from/to`), `concurrency`.

### 4) Endpoints & Contratos (Swagger tag: `B3 Reconciliation`)
- [ ] `POST /reconciliation/scan` (admin‑only) aceita body:
  ```json
  {
    "cpf": "00000000000",
    "tickers": null,
    "from": null,
    "to": null,
    "dryRun": false,
    "maxSamplesPerType": 5,
    "concurrency": 2
  }
  ```
  - [ ] `dryRun=true` **não** persiste; `dryRun=false` faz **upsert** com `dedupe_hash`.
  - [ ] Resposta traz **sumário por tipo** com contagens e amostras.
- [ ] `GET /reconciliation/inconsistencies` com filtros (`cpf`, `status?`, `type?`, `ticker?`, `from?`, `to?`, `page?`, `pageSize?`).
- [ ] `GET /reconciliation/inconsistencies/{id}` retorna o registro completo (incluindo `details` e `sample_dates`).
- [ ] **Códigos**: 200/400/401/403/409/5xx com mensagens claras.

### 5) Segurança & Multi‑tenant
- [ ] **`X-Tenant-Id` obrigatório** em todas as rotas; `POST` restrito a **admin** (RBAC).
- [ ] **CPF mascarado** em logs; **nunca** logar tokens/segredos.
- [ ] Rate‑limit defensivo (opcional) no `POST /scan` para evitar abusos.

### 6) Observabilidade
- [ ] **Logs estruturados** (JSON) incluem: `tenantId`, `cpfMasked`, `tickers`, `from`, `to`, `found_by_type`, `durationMs`.
- [ ] **Métricas Prometheus** expostas/incrementadas:
  - `b3_recon_scan_runs_total{result="success|error"}`
  - `b3_recon_inconsistencies_found_total{type}`
  - `b3_recon_scan_duration_seconds` (histogram)
- [ ] Evitar **alta cardinalidade** (CPF apenas nos logs).

### 7) Performance & Índices
- [ ] Consultas set‑based (CTEs/agregações) minimizam I/O; processamento por **janela mensal** em grandes intervalos.
- [ ] Índices presentes em tabelas base:  
  `b3_normalized_positions(tenant_id, cpf, ticker, reference_date)` e  
  `b3_normalized_transactions(tenant_id, cpf, ticker, trade_date, side)`.
- [ ] Respostas leves: limitar `details` por `maxSamplesPerType`; paginação no `GET`.

### 8) Testes Automatizados
- **Integration** — `src/Tests/Integration/B3/reconciliation/inconsistency_detector_test.go`
  - [ ] Posição sem compras → detecta `OPENING_BALANCE_MISSING` com evidências coerentes.
  - [ ] Primeira transação `SELL` → detecta `SELL_WITHOUT_BUY` (ou cumulativo negativo).
  - [ ] Posição × transações divergente → detecta `POSITION_TX_DIVERGENCE` com amostras.
  - [ ] `dryRun=true` não persiste; `dryRun=false` persiste com **idempotência** (repetir scan não duplica).
  - [ ] Convivência com 1.13/1.14: lock impede corrida.
- **Unit** — `src/Tests/Unit/B3/reconciliation/...`
  - [ ] Cálculo de `dedupe_hash` (campos da assinatura) e estabilidade.
  - [ ] Paginação/filtros no `GET` e montagem de `details`.

### 9) Documentação & Collections
- [ ] **Swagger** atualizado com exemplos reais (sanitizados) e erros.
- [ ] **Postman/Bruno** em `/docs/reconciliation/` com requests prontos.
- [ ] `docs/b3/inconsistencies.md` explica tipos, evidências e troubleshooting.

---

## 🧪 Passos de Validação Manual (rápido)

1) **Scan — dry run** (não persiste)  
```bash
curl -X POST "$BASE_URL/reconciliation/scan" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","dryRun":true,"maxSamplesPerType":3,"concurrency":2}'
```
**Esperado**: 200 com **sumário** por tipo; **BD inalterado**.

2) **Scan — persistente** (upsert)  
```bash
curl -X POST "$BASE_URL/reconciliation/scan" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"cpf":"00000000000","dryRun":false,"maxSamplesPerType":3,"concurrency":2}'
```
**Esperado**: 200 com contagens; registros criados/atualizados em `b3_inconsistencies` (sem duplicação).

3) **Listar**  
```bash
curl "$BASE_URL/reconciliation/inconsistencies?cpf=00000000000&status=OPEN&type=OPENING_BALANCE_MISSING&page=1&pageSize=20" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

4) **Detalhar**  
```bash
curl "$BASE_URL/reconciliation/inconsistencies/<id>" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

5) **Idempotência** (rodar novamente o passo 2)  
**Esperado**: **nenhuma** duplicação; apenas atualização de `last_detected_at`/`updated_at`/contadores.

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.17 (Inconsistency Detector) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
