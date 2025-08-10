# Prompt 1.15 — Reports & Summaries API (E2E Test Helpers) — **suno-wallets**

Implemente uma **API de relatórios e sumários** para **inspeção funcional** do pipeline B3 (RAW → Normalized) **sem** aplicar regras de negócio (sem PM/P&L/impostos). Esses endpoints serão usados para **validação E2E** dos fluxos das etapas **1.9, 1.10, 1.11, 1.13 e 1.14**.

> Objetivo: dar **visibilidade rápida** sobre cobertura temporal, volumes por tipo e visão agregada por ticker, usando dados **persistidos** (RAW e Normalized). **Nada de gravação**, apenas leitura/consulta.

---

## 🎯 Objetivos
1. Expor endpoints **somente leitura** para:
   - **Cobertura temporal do RAW** (primeira/última data por CPF).
   - **Sumário por CPF**: meses com transações, total de transações, total de tickers, contagem de posições, somatório simples de valores **se vier do payload**.
   - **Listagem consolidada por ticker** (quantidades).
2. Implementar consultas **eficientes** (SQL, índices) com **multi‑tenant** estrito.
3. Integrar **observabilidade** (logs/métricas), **segurança** e testes (**unit** + **integration**).

---

## 🧱 Arquitetura (DDD)
Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Api/Controllers/B3/reports_controller.go`
  - `GET /b3/client/raw-date-range` — intervalo temporal no RAW por CPF.
  - `GET /b3/client/summary`        — sumário agregado.
  - `GET /b3/client/tickers`        — visão consolidada por ticker.
- `src/Application/B3/reports/reports_service.go`
  - Orquestra consultas, valida entradas, aplica filtros por tenant/CPF e monta respostas.
- `src/Infrastructure/B3/reports/reports_repository.go`
  - Contém **SQLs otimizadas** (PostgreSQL) para cada endpoint.
- **Tests**:
  - `src/Tests/Unit/B3/reports/...`
  - `src/Tests/Integration/B3/reports/...`

**Observação**: manter **nomes em inglês** e **comentários pt‑BR** em todos os arquivos.

---

## 🔗 Contratos (Swagger tag: `B3 Reports`)

### 1) `GET /b3/client/raw-date-range?cpf=00000000000`
**Headers**: `X-Tenant-Id`, `Authorization`  
**Resposta 200:**
```json
{
  "cpf": "00000000000",
  "from": "YYYY-MM-DD",
  "to": "YYYY-MM-DD",
  "dataTypes": ["transactions","positions"],
  "assetTypes": ["equity"]
}
```
**Semântica**: usa `b3_raw_data_client.period_start`/`period_end` (mínimo e máximo por CPF/tenant).

---

### 2) `GET /b3/client/summary?cpf=00000000000&from=YYYY-MM-DD&to=YYYY-MM-DD`
**Headers**: `X-Tenant-Id`, `Authorization`  
**Resposta 200 (exemplo):**
```json
{
  "cpf": "00000000000",
  "from": "YYYY-MM-DD",
  "to": "YYYY-MM-DD",
  "monthsWithTransactions": 7,
  "totalTransactions": 124,
  "tickersCount": 23,
  "positionsCount": 15,
  "grossValueBRLSum": "123456.78",
  "notes": "Somatórios apenas se presentes no payload; sem PM/P&L."
}
```
**Semântica**:
- `monthsWithTransactions`: nº de meses no intervalo com ao menos 1 transação normalizada.
- `totalTransactions`: contagem em `b3_normalized_transactions` no intervalo.
- `tickersCount`: nº de tickers **distintos** (transações ou posições) no intervalo.
- `positionsCount`: contagem em `b3_normalized_positions` no intervalo.
- `grossValueBRLSum`: soma **apenas** se existir nos registros (RAW/Normalized) — **não calcular** valores ausentes.

---

### 3) `GET /b3/client/tickers?cpf=00000000000&from=YYYY-MM-DD&to=YYYY-MM-DD`
**Headers**: `X-Tenant-Id`, `Authorization`  
**Resposta 200 (exemplo):**
```json
{
  "cpf": "00000000000",
  "from": "YYYY-MM-DD",
  "to": "YYYY-MM-DD",
  "tickers": [
    {"ticker": "ABCD3", "quantityTotal": "150.0000000000"},
    {"ticker": "EFGH4", "quantityTotal": "12.0000000000"}
  ]
}
```
**Semântica**: soma de `quantity` por `ticker` considerando **apenas dados normalizados** no intervalo, **sem** PM ou preço médio. A fonte pode ser `b3_normalized_positions` (foto por data de referência) ou `b3_normalized_transactions` (soma de quantidades líquidas, se o payload fornecer sentido suficiente). **Escolha primária**: posições (v3), pois já são “fotos” por data.

---

## 🗃️ SQLs (PostgreSQL — sugestões)

> **Filtrar sempre por `tenant_id` e `cpf`**. **Não** fazer varredura sem índices.

### a) Intervalo no RAW
```sql
SELECT
  MIN(period_start) AS from_date,
  MAX(period_end)   AS to_date
FROM b3_raw_data_client
WHERE tenant_id = $1 AND cpf = $2;
```

### b) Meses com transações (normalized)
```sql
WITH months AS (
  SELECT DATE_TRUNC('month', trade_date)::date AS m
  FROM b3_normalized_transactions
  WHERE tenant_id = $1 AND cpf = $2 AND trade_date BETWEEN $3 AND $4
  GROUP BY 1
)
SELECT COUNT(*) AS months_with_tx FROM months;
```

### c) Total de transações e tickers distintos
```sql
SELECT
  COUNT(*) AS total_tx,
  COUNT(DISTINCT ticker) AS tickers_distinct
FROM b3_normalized_transactions
WHERE tenant_id = $1 AND cpf = $2 AND trade_date BETWEEN $3 AND $4;
```

### d) Contagem de posições por data
```sql
SELECT COUNT(*) AS positions_count
FROM b3_normalized_positions
WHERE tenant_id = $1 AND cpf = $2 AND reference_date BETWEEN $3 AND $4;
```

### e) Soma simples de valores em BRL (se houver)
```sql
SELECT COALESCE(SUM(position_value), 0) AS gross_brl_sum
FROM b3_normalized_positions
WHERE tenant_id = $1 AND cpf = $2
  AND reference_date BETWEEN $3 AND $4
  AND currency = 'BRL';
```

### f) Consolidação por ticker (quantidade)
```sql
SELECT
  ticker,
  SUM(quantity) AS quantity_total
FROM b3_normalized_positions
WHERE tenant_id = $1 AND cpf = $2
  AND reference_date BETWEEN $3 AND $4
GROUP BY ticker
ORDER BY ticker;
```

> Se optar por transações para a consolidação, a regra de sinal deveria vir do `side` (`BUY` positivo, `SELL` negativo). **Não** calcular PM; sem imputações.

---

## 🔐 Segurança & Multi‑tenant
- **`X-Tenant-Id` obrigatório**. Todas as consultas filtradas por `tenant_id` e `cpf` (11 dígitos).  
- **Somente leitura** — **não** alterar estado.  
- **Mascarar CPF** em logs; **não** logar tokens/segredos.  
- Rate‑limit defensivo nesses endpoints (opcional).

---

## 📊 Observabilidade
- **Logs estruturados** (JSON): `tenantId`, `cpfMasked`, filtros (`from/to`), tempos e tamanhos de resposta.  
- **Métricas Prometheus**:
  - `b3_reports_requests_total{endpoint}`
  - `b3_reports_duration_seconds{endpoint}` (histogram)
  - `b3_reports_errors_total{endpoint}`
- Evitar **alta cardinalidade** nas labels das métricas; **não** usar `tenantId` como label.

---

## 🚦 Performance
- Garantir **índices** em (`tenant_id`, `cpf`, `trade_date`) e (`tenant_id`, `cpf`, `reference_date`), além de (`tenant_id`, `cpf`, `ticker`).  
- Preferir **CTEs** e agregações no banco; **evitar** processamento em memória.  
- Paginar respostas longas (ex.: `tickers` com `limit/offset`).

---

## 🧪 Testes (obrigatório)
**Integration** — `src/Tests/Integration/B3/reports_endpoints_test.go`  
- Após execuções de 1.13/1.14, `raw-date-range` retorna intervalo coerente.  
- `summary` retorna contagens consistentes com os dados criados.  
- `tickers` retorna lista não vazia (quando aplicável) e quantidades esperadas.  
- Verificar uso correto de `tenant_id` e `cpf`; headers obrigatórios.

**Unit** — `src/Tests/Unit/B3/reports/...`  
- Mocks do repositório devolvendo cenários típicos (sem dados, poucos dados, dados extensos).  
- Validação de parâmetros (datas, CPF) e formatação de respostas.

---

## 📄 Documentação
- **Swagger** (tag `B3 Reports`) com exemplos de respostas e códigos de erro (200/400/401/403/5xx).  
- **Postman**/**Bruno**: coleções em `/docs/reports/` com chamadas prontas (sanitizadas).  
- `docs/b3/reports.md` com explicação dos campos, limitações (sem PM/P&L) e exemplos cURL.

### Exemplos cURL
```bash
# Intervalo do RAW
curl "$BASE_URL/b3/client/raw-date-range?cpf=00000000000" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"

# Sumário por intervalo
curl "$BASE_URL/b3/client/summary?cpf=00000000000&from=2020-01-01&to=2025-08-08" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"

# Quantidades por ticker
curl "$BASE_URL/b3/client/tickers?cpf=00000000000&from=2020-01-01&to=2025-08-08" \
  -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

---

## ✅ Entregáveis
- Controllers + service + repository de relatórios.  
- Testes unitários e de integração.  
- Swagger + Postman/Bruno + docs.  
- Logs e métricas integrados.

**Critério de aceite**: endpoints retornam **agregados corretos** e **performáticos** sobre um CPF com dados de 1.9/1.10/1.13/1.14, sem cálculos de negócio ou duplicidades, e com observabilidade pronta para produção.
