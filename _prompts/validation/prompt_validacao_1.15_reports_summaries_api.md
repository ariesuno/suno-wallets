# Prompt de Validação 1.15v — Reports & Summaries API (E2E Test Helpers) — **suno-wallets**

## 🎯 Objetivo
Validar que a **API de relatórios/sumários** implementada no **Prompt 1.15** está correta, performática, segura (multi‑tenant), **somente leitura**, e alinhada aos contratos e à observabilidade definidos.

> Escopo cobre: `GET /b3/client/raw-date-range`, `GET /b3/client/summary`, `GET /b3/client/tickers`.

---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos estão nas pastas corretas, com **nomes em inglês** e **comentários pt‑BR**:
  - `src/Api/Controllers/B3/reports_controller.go`
  - `src/Application/B3/reports/reports_service.go`
  - `src/Infrastructure/B3/reports/reports_repository.go`
  - **Tests**: `src/Tests/Unit/B3/reports/...` e `src/Tests/Integration/B3/reports/...`
- [ ] Camadas respeitadas: Controller (valida e chama Service) → Service (orquestra e aplica regras de leitura) → Repository (SQL/DB).

### 2) Contratos / Swagger / Collections
- **`GET /b3/client/raw-date-range`**
  - [ ] Requer **`X-Tenant-Id`** e autenticação.
  - [ ] Retorna `{ cpf, from, to, dataTypes, assetTypes }` conforme especificação.
- **`GET /b3/client/summary`**
  - [ ] Query: `cpf`, `from`, `to` (`YYYY-MM-DD`), `X-Tenant-Id` em header.
  - [ ] Retorna campos: `monthsWithTransactions`, `totalTransactions`, `tickersCount`, `positionsCount`, `grossValueBRLSum` (se disponível) e `notes`.
- **`GET /b3/client/tickers`**
  - [ ] Query: `cpf`, `from`, `to` + paginação opcional (`limit`, `offset`).
  - [ ] Retorna `{ tickers: [{ticker, quantityTotal}] }`.
- [ ] **Swagger** (tag `B3 Reports`) atualizado com exemplos de 200/400/401/403/5xx.
- [ ] **Postman/Bruno** em `/docs/reports/` com requisições sanitizadas.

### 3) Validações de Entrada
- [ ] `cpf` com **11 dígitos**, `from`/`to` no formato **`YYYY-MM-DD`**; `from <= to`.
- [ ] `X-Tenant-Id` **obrigatório**; **mascarar CPF** em logs.
- [ ] **Sem** defaults perigosos: exigir `from/to` em `summary`/`tickers` (ou documentar defaults explícitos).
- [ ] Paginação em `tickers` (quando aplicável): `limit` máx. sensato (ex.: 1000).

### 4) SQL / Repositório (PostgreSQL)
- [ ] Todas as queries **filtram por `tenant_id` e `cpf`** (estrito).
- [ ] `raw-date-range`: usa `MIN(period_start)` / `MAX(period_end)` em `b3_raw_data_client` com filtros.
- [ ] `monthsWithTransactions`: `DATE_TRUNC('month', trade_date)` + `COUNT(*)` distinto.
- [ ] `positionsCount`: conta em `b3_normalized_positions` por intervalo.
- [ ] `grossValueBRLSum`: soma **apenas** se campo existir e `currency='BRL'`; **não** calcular valores ausentes.
- [ ] `tickers`: soma `quantity` por `ticker` **preferindo posições v3** (foto por data) — documentar escolha.
- [ ] Usar **prepared statements**/placeholders (evitar concatenação).
- [ ] Conferir **precisão** numérica: `numeric(28,10)` no retorno para quantidades/valores.
- [ ] Confirmação via **EXPLAIN (ANALYZE)** de que as queries usam **índices** (sem seq scans desnecessários).

### 5) Índices & Performance
- [ ] Índices presentes (ou migrations criadas) para:
  - `b3_normalized_transactions(tenant_id, cpf, trade_date)`
  - `b3_normalized_transactions(tenant_id, cpf, ticker)`
  - `b3_normalized_positions(tenant_id, cpf, reference_date)`
  - `b3_normalized_positions(tenant_id, cpf, ticker)`
  - `b3_raw_data_client(tenant_id, cpf, period_start, period_end)`
- [ ] Respostas de `tickers` **paginadas** quando volumosas; avaliar compressão HTTP.
- [ ] Tempo de resposta aceitável nos cenários com volumetria típica (documentar metas, ex.: p95 < 300ms).

### 6) Segurança & Somente Leitura
- [ ] Endpoints **não** alteram estado (somente SELECTs).
- [ ] **RBAC** aplicado (escopo mínimo necessário para leitura).
- [ ] **Sem** logging de tokens/segredos; CPF mascarado nos logs.
- [ ] Rate‑limit defensivo opcional (evitar abuso de leitura).

### 7) Observabilidade
- [ ] **Logs estruturados** (JSON) com: `tenantId`, `cpfMasked`, `endpoint`, `from`, `to`, `durationMs`, `rows`.
- [ ] **Métricas Prometheus**:
  - `b3_reports_requests_total{endpoint}`
  - `b3_reports_duration_seconds{endpoint}` (histogram)
  - `b3_reports_errors_total{endpoint}`
- [ ] Sem **alta cardinalidade**: **não** usar `tenantId` como label; deixá-lo apenas nos **logs**.

### 8) Testes
- **Integration —** `src/Tests/Integration/B3/reports_endpoints_test.go`
  - [ ] `raw-date-range` retorna `from/to` coerentes após 1.13/1.14.
  - [ ] `summary` retorna contagens consistentes e `grossValueBRLSum` apenas quando houver valor nos dados.
  - [ ] `tickers` lista quantidades esperadas (paginação quando necessário).
  - [ ] Headers obrigatórios validados; erros 400/401/403/5xx cobrindo cenários.
- **Unit —** `src/Tests/Unit/B3/reports/...`
  - [ ] Service: validação de parâmetros, montagem de respostas.
  - [ ] Repository: mocks/fixtures para cenários sem dados, poucos dados e dados extensos.

### 9) Documentação & Operação
- [ ] `docs/b3/reports.md` descreve campos, limitações (sem PM/P&L), exemplos cURL, e metas de desempenho.
- [ ] Coleções Postman/Bruno atualizadas e versionadas junto do código.
- [ ] Se exposto em Grafana, dashboards incluem latência, taxa de erro e throughput destes endpoints.

---

## 🧪 Passos de Validação Manual (rápido)
1. **Intervalo RAW**
   ```bash
   curl "$BASE_URL/b3/client/raw-date-range?cpf=00000000000" \
     -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
   ```
   Esperado: datas coerentes com o que foi ingerido (1.9, 1.13, 1.14).

2. **Sumário**
   ```bash
   curl "$BASE_URL/b3/client/summary?cpf=00000000000&from=2020-01-01&to=2025-08-08" \
     -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
   ```
   Esperado: contagens > 0 em ambiente com dados; `grossValueBRLSum` só quando existir no payload.

3. **Tickers**
   ```bash
   curl "$BASE_URL/b3/client/tickers?cpf=00000000000&from=2020-01-01&to=2025-08-08" \
     -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
   ```
   Esperado: lista paginada, ordenada por `ticker`, com `quantityTotal` em `numeric`.

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.15 (Reports & Summaries API) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
