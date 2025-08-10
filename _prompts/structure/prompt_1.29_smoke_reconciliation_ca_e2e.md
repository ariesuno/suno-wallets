# Prompt 1.29 — Smoke Reconciliação & Corporate Actions (E2E) — **suno-wallets**

Implemente uma **suite E2E Smoke** que valida o **fluxo ponta‑a‑ponta** da Fase 1:  
**(1)** ingestão B3 (full → incremental), **(2)** detecção de inconsistências, **(3)** auto‑fix (system ops), **(4)** override manual, **(5)** projeção de Corporate Actions com *comicota* (floor), **(6)** geração de relatórios/export e **(7)** observabilidade.  
A suite deve retornar **PASS/FAIL** com um **sumário consolidado** e ser executável em **CI** e **local** (sem dados fake gravados).

> **Regras gerais:** nomes de arquivos/classes/funções em **inglês**; comentários em **pt‑BR**; **multi‑tenant** estrito; **sem PII** em logs/métricas; CPF sempre **mascarado** nos logs e parametrizado via `.env`.  
> **Dependências**: 1.11–1.25 (principalmente 1.17, 1.18, 1.20, 1.22, 1.23, 1.24, 1.25).


---

## 🎯 Objetivos
1) Executar o **caminho feliz** completo com um CPF parametrizável (ex.: `E2E_TEST_CPF`) e **tenant** (`E2E_TEST_TENANT_ID`).  
2) Validar **idempotência** (repetir a suite sem duplicações e com mesmas contagens).  
3) Checar **métricas Prometheus** esperadas e **logs** correlacionados por `traceId/requestId`.  
4) Produzir **artifact** `.json`/`.md` com sumário da execução (**PASS/FAIL** + métricas/resumo).  
5) Opcional: *gate* de CI **falha** se algum requisito mínimo não for atendido (ex.: p95 > limite, erros B3 > limite).


---

## 🧱 Estrutura (código + scripts + docs)

Crie os seguintes artefatos (nomes em inglês, comentários pt‑BR):

- `src/Tests/E2E/smoke/smoke_e2e_test.go`  
  Teste E2E orquestrado (testify).

- `scripts/e2e_smoke.sh`  
  *Wrapper* shell para rodar local/CI, produzir artifact e retornar código de saída.

- `docs/e2e/smoke_playbook.md`  
  Como executar, variáveis de ambiente, leitura do sumário e *troubleshooting*.

- `docs/postman/e2e_smoke_collection.json` e `docs/bruno/e2e_smoke.bru`  
  Coleções equivalentes para *debug* manual.

- `deploy/ci/e2e-smoke.yml`  
  Job de CI (ex.: GitHub Actions) que executa a suite e publica artifacts.


---

## ⚙️ Variáveis (.env)

- `E2E_TEST_CPF` — CPF alvo (teste real; **não** registrar PII em logs).  
- `E2E_TEST_TENANT_ID` — tenant de teste.  
- `E2E_TEST_FROM` / `E2E_TEST_TO` — janela de datas (ISO).  
- `BASE_URL`, `AUTH_TOKEN` — API base e token.  
- `OBS_REQUIRE_MIN_SAMPLE_RATE=0.05` — traços mínimos coletados.  
- **Limites/Gates CI**:  
  - `GATE_MAX_P95_API=1.5s`  
  - `GATE_MAX_B3_ERROR_5M=50`  
  - `GATE_MAX_INCONSISTENCIES_AFTER_AUTOFIX=0` (permitir N>0 se desejado)


---

## 🧪 Cenários e Passos (E2E “caminho feliz”)

> Todas as chamadas devem incluir `X-Tenant-Id` e `Authorization`. **Mascarar CPF** nos logs; exibir apenas `***1234` no sumário.

### 0) **Preflight & Status**
- `GET /health/ready` → 200
- `GET /observability/status` → campos principais OK (DB, B3 client, jobs, ledger, cache).

### 1) **Reset Seguro (opcional)** — *zero‑and‑refetch*
- Endpoint administrativo (entregue em 1.13/1.14) para **zerar** estado calculado (ledger/tabelas normalizadas) do CPF e **não** apagar RAW.  
- Verifique que timeline fica vazia **antes** do refetch.

### 2) **Ingestão Full B3**
- `POST /b3/fetch/full` (ou equivalente) com `from=2019‑12‑01` (ou `E2E_TEST_FROM`) e `to=E2E_TEST_TO`.  
- Garantir:  
  - RAW payloads foram **persistidos**.  
  - Normalized/ledger recebeu as entradas correspondentes.  
  - Métricas: `b3_requests_total`, `ledger_ops_ingested_total{source="B3_RAW"}` cresceram.

### 3) **Incremental Diário**
- `POST /b3/fetch/incremental` para o **dia seguinte** ao último `to`.  
- Garantir que **não duplica** operações já coletadas.  
- Métricas de latência e taxa de erro sob limites (`GATE_MAX_B3_ERROR_5M`).

### 4) **Detecção de Inconsistências** (1.17)
- `POST /ops/inconsistencies/scan?cpf=` (ou gatilho equivalente).  
- `GET /ops/inconsistencies?cpf=` deve retornar itens típicos (ex.: *posição sem compra*, *venda sem compra*).  
- Métrica `inconsistency_open_gauge` > 0 antes do auto‑fix.

### 5) **Auto‑fix (System Operations)** (1.18)
- `POST /ops/system/autofix?cpf=` → gera operações sintéticas **neutras** para fechar as inconsistências (sem alterar P&L).  
- `GET /ops/inconsistencies?cpf=` → **0** abertas (ou ≤ `GATE_MAX_INCONSISTENCIES_AFTER_AUTOFIX`).  
- Ledger contém novas operações `source='SYSTEM_SYNTHETIC'`.  
- Métrica `ledger_ops_ingested_total{source="SYSTEM_SYNTHETIC"}` cresceu.

### 6) **Override Manual (se aplicável)** (1.19)
- `POST /ops/manual` (ex.: inserir uma **compra** que substitui a sintética).  
- Confirmar **substituição controlada**: a operação sintética é marcada como inativa/obsoleta e a manual passa a valer.  
- Métrica `dedup_decisions_total{decision="manual_merge"|...}` (se aplicável) cresce.

### 7) **Corporate Actions Projection — *comicota*** (1.24)
- `POST /catalog/ca/projection/preview` com período/tickers relevantes.  
- `POST /catalog/ca/projection/apply` → aplica versão **N**.  
- Validar **floor** nas quantidades (sem fração), `FRACTION_ADJUSTMENT` quando houver resto, e **neutralidade econômica**.  
- Reexecutar `apply` da mesma versão **não** duplica (idempotência).  
- `GET /catalog/ca/projection/status?cpf=` exibe a versão e contagens criadas.

### 8) **Timeline & Export** (1.20)
- `GET /ops/timeline?cpf=&from=&to=&respectPolicy=true` → dados consolidados (B3 + system + manual + CA).  
- `GET /ops/timeline/export?format=csv` → arquivo baixado; validar cabeçalhos/linhas > 0.

### 9) **Sumário & PASS/FAIL**
- Coletar contagens de: RAW salvos, ledger total, inconsistências antes/depois, system ops criadas, manual overrides, CA ops.  
- Confirmar **idempotência**: repetir os passos **4→8** deve produzir **mesmos totais**.  
- Checar **métricas**: p95 API < `GATE_MAX_P95_API`, erros B3 em 5m < `GATE_MAX_B3_ERROR_5M`.  
- Persistir `artifacts/e2e_smoke_summary.json` e `artifacts/e2e_smoke_summary.md`.


---

## 🧩 Código — *Skeleton* do teste E2E (Go)

> **Observação**: este é um **esqueleto**. Preencha as rotas exatas já implementadas no seu backend.

```go
// Arquivo: src/Tests/E2E/smoke/smoke_e2e_test.go
// Comentários em pt-BR explicando cada etapa do fluxo.
package smoke

import (
  "encoding/json"
  "net/http"
  "os"
  "testing"
  "time"

  "github.com/stretchr/testify/require"
)

type ApiClient struct {
  baseURL string
  token   string
  tenant  string
  client  *http.Client
}

func (c *ApiClient) do(t *testing.T, method, path string, body []byte) *http.Response {
  // ... constrói request, injeta Authorization e X-Tenant-Id, executa e retorna response
  return nil
}

func Test_E2E_Smoke_Reconciliation_CA(t *testing.T) {
  cpf := os.Getenv("E2E_TEST_CPF")
  tenant := os.Getenv("E2E_TEST_TENANT_ID")
  base := os.Getenv("BASE_URL")
  token := os.Getenv("AUTH_TOKEN")
  require.NotEmpty(t, cpf)
  require.NotEmpty(t, tenant)
  require.NotEmpty(t, base)
  require.NotEmpty(t, token)

  api := &ApiClient{baseURL: base, token: token, tenant: tenant, client: &http.Client{Timeout: 60 * time.Second}}

  // 0) Preflight
  // GET /health/ready → 200
  // GET /observability/status → valida campos básicos

  // 1) Reset opcional — zero and refetch sem apagar RAW
  // POST /admin/maintenance/zero-and-refetch?cpf=...

  // 2) Full Fetch B3 (from..to)
  // 3) Incremental Fetch (next day)

  // 4) Inconsistency Scan + leitura
  // 5) Auto-fix (system ops) + leitura
  // 6) Manual override (opcional) + leitura
  // 7) CA Projection (preview/apply) + leitura
  // 8) Timeline + Export

  // 9) Agregar métricas e gerar sumário em artifacts/e2e_smoke_summary.json
  summary := map[string]any{
    "result": "PASS",
    "cpfMasked": "***" + cpf[len(cpf)-4:],
    "totals": map[string]int{
      "rawSaved":                0,
      "ledgerTotal":             0,
      "inconsistenciesBefore":   0,
      "inconsistenciesAfter":    0,
      "systemOpsCreated":        0,
      "manualOverrides":         0,
      "caOpsCreated":            0,
    },
    "limits": map[string]any{
      "p95ApiSec": 1.5,
      "b3ErrorsLast5m": 50,
    },
  }
  b, _ := json.MarshalIndent(summary, "", "  ")
  _ = os.MkdirAll("artifacts", 0o755)
  _ = os.WriteFile("artifacts/e2e_smoke_summary.json", b, 0o644)
}
```


---

## 🖥️ Script *wrapper*

Crie `scripts/e2e_smoke.sh` (com comentários em pt‑BR) para executar local/CI:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Carrega env
: "${BASE_URL:?}"
: "${AUTH_TOKEN:?}"
: "${E2E_TEST_TENANT_ID:?}"
: "${E2E_TEST_CPF:?}"

echo "[E2E] Iniciando Smoke..."

# Roda o teste go (ou testes do seu runner preferido)
go test ./src/Tests/E2E/smoke -v -run Test_E2E_Smoke_Reconciliation_CA

# Pós-processa artifacts (ex.: converte JSON → MD resumido)
if [ -f artifacts/e2e_smoke_summary.json ]; then
  echo "## E2E Smoke Summary" > artifacts/e2e_smoke_summary.md
  jq -r '. as $root | "Result: \($root.result)\nCPF: \($root.cpfMasked)\nTotals: \($root.totals)\nLimits: \($root.limits)"' artifacts/e2e_smoke_summary.json >> artifacts/e2e_smoke_summary.md
fi

echo "[E2E] Finalizado."
```


---

## 📊 Observabilidade na Execução

- Confirmar presença de *spans* para as rotas usadas (HTTP server/client B3, DB, jobs).  
- Métricas mínimas esperadas durante a suite:
  - `b3_requests_total` ↑
  - `ledger_ops_ingested_total{source="B3_RAW|SYSTEM_SYNTHETIC|USER_MANUAL"}` ↑
  - `inconsistency_open_gauge` → pico antes do auto‑fix, volta a 0 (ou ≤ `GATE_MAX_INCONSISTENCIES_AFTER_AUTOFIX`)
  - `ca_projection_runs_total{status}` ↑ quando aplicar
  - `http_request_duration_seconds` p95 ≤ `GATE_MAX_P95_API`

- **Logs**: entradas com `traceId`, `spanId`, `requestId`, `tenantId` e `cpfMasked` (`***1234`).


---

## ✅ Critérios de Aceite (PASS)

- Suite executa com **código de saída 0**.  
- **Idempotência**: segunda execução não duplica dados; totais coerentes.  
- **Auto‑fix** elimina (ou reduz ao limite) incoerências detectadas.  
- **CA Projection** aplica *floor* corretamente e é idempotente por **versão**.  
- **Export** gera arquivo válido com cabeçalho e ≥1 linha.  
- **Observabilidade**: métricas/traços/logs visíveis; nenhum CPF exposto em métricas.  
- **Artifacts** `e2e_smoke_summary.json` e `.md` criados.


---

## 📄 Documentação
- Atualize **Swagger** com tags de rotas usadas na suite.  
- Atualize **Postman/Bruno** (`docs/postman/e2e_smoke_collection.json`, `docs/bruno/e2e_smoke.bru`).  
- `docs/e2e/smoke_playbook.md`: como configurar `.env`, executar local/CI, interpretar falhas e checagens rápidas no Grafana/Tempo.


---

## 🔧 Entregáveis
- Teste Go E2E + script wrapper + coleções Postman/Bruno + job CI + documentação.  
- Suite rodando em **local** e **CI**, exibindo **PASS/FAIL** e sumário consolidado.
