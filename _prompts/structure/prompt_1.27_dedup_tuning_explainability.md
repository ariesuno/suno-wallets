# Prompt 1.27 — Dedup Tuning & Explainability — **suno-wallets**

Aprimore o módulo de **Deduplicação** (base 1.19) com **tuning de parâmetros**, **modo sombra (shadow)** e **explainability** completa (quebra por features e contribuição no score). Inclua **avaliação** com rótulos (precision/recall/F1/PR‑curve), **métricas Prometheus**, **endpoints admin** para `get/set/dry-run/explain/evaluate`, e integração com **policy** (1.21) e **Backoffice** (1.22).

> Nomes de arquivos/classes/funções em **inglês**; comentários em **pt‑BR**. **Sem dados fake**. Multi‑tenant estrito. Observabilidade obrigatória.  
> Dedup **somente** quando `Client Policy` (1.21) = `HYBRID`. Em `B3_ONLY` ou `MANUAL_ONLY`, registrar **skip** com métrica.


---

## 🎯 Objetivos
1) **Configuração por tenant** com **versionamento** e modos `OFF | SHADOW | ENFORCED`.  
2) **Explainability** por candidato (score final + contribuição de cada feature).  
3) **Tuning**: ajustar *thresholds* (globais e por *assetType*/`operationType`) e parâmetros de janelas/tolerâncias.  
4) **Dry‑run**: simular impacto (contagens, divergências) sem alterar decisões.  
5) **Avaliação**: carregar rótulos, rodar *eval* e expor métricas (precision, recall, F1, PR‑curve).  
6) **APIs** admin + métricas/logs; testes (unit/integration).


---

## 🧱 Arquitetura (DDD)

Crie/atualize (nomes em inglês, comentários pt‑BR):

- `src/Domain/Dedup/dedup_config.go`  
  Estruturas de configuração (por tenant):  
  ```go
  type DedupMode string // "OFF" | "SHADOW" | "ENFORCED"
  type FeatureWeights struct {
      SameTicker float64         // ticker exato
      CanonicalTickerMatch float64
      OperationTypeMatch float64 // BUY/SELL etc.
      DateDeltaDays float64      // penalização por distância em dias
      QuantityRelDiff float64    // penalização por diferença relativa na quantidade
      PriceRelDiff float64       // penalização por diferença relativa no preço
      BrokerMatch float64        // corretora/clearing
      TradeIdMatch float64       // IDs quando disponíveis
      SourceProximity float64    // B3 vs Manual criação próxima no tempo
  }
  type Thresholds struct {
      Global float64
      ByAssetType map[string]float64   // EQUITY/FII/ETF...
      ByOperationType map[string]float64 // BUY/SELL/...
  }
  type WindowTolerance struct {
      MaxDateDeltaDays int
      MaxQtyRelDiff float64
      MaxPriceRelDiff float64
  }
  type DedupConfig struct {
      Mode DedupMode
      Weights FeatureWeights
      Thresholds Thresholds
      Window WindowTolerance
      RespectPolicy bool // default true
  }
  ```

- `src/Application/Dedup/dedup_explainer.go`  
  Calcula **score** e retorna **explicação** por feature (ver fórmula abaixo).

- `src/Application/Dedup/dedup_tuner_service.go`  
  `get/set` config, `dryRun`, `evaluate`, `snapshotMetrics`.

- `src/Infrastructure/Dedup/dedup_config_repository.go`  
  Persistência + versionamento por tenant.

- `src/Infrastructure/Dedup/dedup_eval_repository.go`  
  Rótulos, execuções de avaliação e PR‑curve.

- `src/Api/Controllers/Dedup/dedup_admin_controller.go`  
  Endpoints admin (config, dry‑run, explain, eval, runs).

> Reuso dos **candidates**/resolvers do 1.19; este prompt adiciona *tuning* e *explainability*.


---

## 🗃️ Banco de Dados (migrations)

```sql
CREATE TABLE IF NOT EXISTS dedup_config (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  version INTEGER NOT NULL,
  mode TEXT NOT NULL CHECK (mode IN ('OFF','SHADOW','ENFORCED')),
  params JSONB NOT NULL,                     -- armazena DedupConfig (weights/thresholds/window/flags)
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by TEXT,
  UNIQUE (tenant_id, version)
);

-- Labels para avaliação (pares identificados do pipeline de candidatos, ou overrides manuais do 1.19)
CREATE TABLE IF NOT EXISTS dedup_eval_labels (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  pair_key TEXT NOT NULL,                    -- hash estável do par de operações
  is_duplicate BOOLEAN NOT NULL,
  source TEXT NOT NULL,                      -- OPERATOR|IMPORT|OTHER
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, pair_key)
);

-- Execuções de avaliação para uma versão/configuração/snapshot
CREATE TABLE IF NOT EXISTS dedup_eval_runs (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  config_version INTEGER NOT NULL,
  dataset_info JSONB,                        -- janela temporal, filtros, contagens
  metrics JSONB,                             -- precision, recall, f1, pr_curve[], thresholds[]
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Explicações (opcional cache) por candidato
CREATE TABLE IF NOT EXISTS dedup_candidate_explanations (
  candidate_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  score NUMERIC(6,5) NOT NULL,
  threshold NUMERIC(6,5) NOT NULL,
  features JSONB NOT NULL,                   -- {"SameTicker":1,"OperationTypeMatch":1,"DateDeltaDays":-0.12,...}
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_dedup_cfg_active ON dedup_config (tenant_id, is_active);
CREATE INDEX IF NOT EXISTS ix_dedup_labels ON dedup_eval_labels (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS ix_dedup_eval_runs ON dedup_eval_runs (tenant_id, created_at DESC);
```


---

## 🧠 Scoring & Explainability

**Fórmula sugerida**:  
- Normalizar *features* para faixa [-1..1] (match → positivo; divergência → negativo proporcional).  
- Score logístico: `z = Σ (w_i * f_i)`, `score = 1 / (1 + e^(−z))`.  
- **Explicação** = vetor `{feature: value, weight, contribution = w_i * f_i}` + `score`, `threshold`, `decision`.

**Exemplo de `explain` (resposta):**
```json
{
  "candidateId":"uuid",
  "score":0.87,
  "threshold":{"effective":0.82,"global":0.82,"byAssetType":null,"byOperationType":0.80},
  "decision":"MATCH",
  "features":[
    {"name":"SameTicker","value":1,"weight":1.20,"contribution":1.20},
    {"name":"OperationTypeMatch","value":1,"weight":0.80,"contribution":0.80},
    {"name":"DateDeltaDays","value":-0.10,"weight":-1.50,"contribution":0.15},
    {"name":"QuantityRelDiff","value":-0.05,"weight":-2.00,"contribution":0.10}
  ]
}
```

**Threshold efetivo**: usa `ByOperationType` > `ByAssetType` > `Global` (primeira disponível).


---

## 📡 Endpoints (Swagger tag: `Dedup — Admin`)

### Config & Tuning
- `GET /admin/dedup/config` → versão ativa do tenant (params + mode).
- `POST /admin/dedup/config` *(admin)*  
  Body:
  ```json
  {
    "mode":"SHADOW|ENFORCED|OFF",
    "weights":{...},
    "thresholds":{"global":0.82,"byAssetType":{"EQUITY":0.84},"byOperationType":{"BUY":0.80}},
    "window":{"maxDateDeltaDays":2,"maxQtyRelDiff":0.15,"maxPriceRelDiff":0.03},
    "respectPolicy": true
  }
  ```
  Cria **nova versão** (não editar a anterior).

- `POST /admin/dedup/config/dry-run` *(admin)*  
  Simula impacto da nova config/thresholds em um **conjunto** (`from..to`, `cpf?`, `assetTypes?`, `operationTypes?`).  
  **Resposta**:
  ```json
  {
    "candidatesTotal": 15234,
    "aboveThreshold": 3210,
    "shadowDivergences": 87,  // decisões diferentes da versão atual (se houver)
    "scoreHistogram":[[0.0,0.1,12],[0.1,0.2,54],...],
    "byOperationType":{"BUY":{"candidates":...,"above":...}},
    "byAssetType":{"EQUITY":{"candidates":...,"above":...}}
  }
  ```

### Explainability & Inspeção
- `GET /admin/dedup/candidates?cpf=&status=open|resolved&minScore=&cursor=&limit=`  
  Lista candidatos com **score** e *status* (keyset pagination).

- `GET /admin/dedup/candidates/{id}/explain`  
  Retorna **quebra por features**, `score`, `threshold efetivo` e **decisão**.

### Avaliação (labels & métricas)
- `POST /admin/dedup/eval/labels` *(admin)*  
  Upsert de rótulos `{pairKey,isDuplicate,source,notes?}` (lotes permitidos).

- `GET /admin/dedup/eval/labels?stats=1`  
  Estatísticas básicas: volume por `source`, datas, cobertura por `assetType`/`operationType`.

- `POST /admin/dedup/eval/run` *(admin)*  
  Executa avaliação da **versão ativa** ou `configVersion` especificada sobre o conjunto informado (`from..to`, filtros).  
  **Resposta**: `precision`, `recall`, `f1`, `support`, `prCurve[]`, `thresholds[]`.

- `GET /admin/dedup/eval/runs`  
  Lista execuções com métricas e metadados (paginado).

**Segurança**: `X-Tenant-Id` obrigatório; RBAC: `ADMIN` para *set/dry-run/eval*, `SUPPORT` leitura/inspeção.  
**LGPD**: **nunca** expor CPF em respostas/labels; usar chaves estáveis (`pairKey`/`candidateId`).


---

## 📊 Observabilidade

**Métricas Prometheus**
- `dedup_candidates_total{status="open|resolved"}`
- `dedup_candidate_scores_bucket` (histogram)  
- `dedup_decisions_total{decision="auto_match|auto_nonmatch|manual_merge|manual_ignore"}`
- `dedup_mode_info{mode}` (gauge: 0/1 por modo)
- `dedup_threshold_effective{level="global|assetType|operationType"}` (gauge)
- `dedup_shadow_divergences_total` (quando `mode=SHADOW`)
- `dedup_eval_precision`, `dedup_eval_recall`, `dedup_eval_f1` (gauges por `config_version`)

**Logs**
- JSON com `tenantId`, `configVersion`, `mode`, `candidateId`, `score`, `threshold`, `decision`, `shadowDivergence?`, `durationMs`.  
- **Sem PII**; nunca logar CPF/IDs originais de ordem/clearing.

**Alertas (sugestões)**
- p95 de `dedup_explain` > 500ms por 10m.  
- `dedup_shadow_divergences_total` > N por 1h após mudança de config (sinal de tuning arriscado).


---

## 🚦 Performance
- Keyset pagination por `(tenant_id, score DESC, candidate_id)`.
- Índices nos artefatos de candidatos (1.19) e nas novas tabelas.  
- *Caching* leve de explicações recentes (últimos N) por tenant.  
- `dry-run` e `eval` em **lotes** com limites configuráveis; *timeouts* e *circuit breakers*.


---

## 🔐 Segurança
- RBAC: `ADMIN` pode alterar config e rodar eval; `SUPPORT` apenas leitura/explicação.  
- `X-Tenant-Id` obrigatório; validação estrita de `pairKey`/`candidateId`.  
- Rate‑limit por IP/usuário em rotas pesadas (`dry-run`/`eval`).


---

## 🧪 Testes (obrigatório)

**Integration — `src/Tests/Integration/Dedup/dedup_tuning_explainability_test.go`**
- `POST /admin/dedup/config` cria **nova versão**; `GET /.../config` retorna ativa.  
- `dry-run` retorna histogram e `shadowDivergences` coerentes ao mudar `threshold`.  
- `GET /.../candidates?...` pagina por **keyset** e respeita `minScore`.  
- `GET /.../candidates/{id}/explain` soma de contribuições condiz com `z`/`score`.  
- `policy=HYBRID` → dedup ativo; `B3_ONLY|MANUAL_ONLY` → métrica `policy_skipped_total` (ou específica) cresce e dedup ignora.  
- `POST /.../eval/labels` upsert; `POST /.../eval/run` retorna métricas consistentes com labels.  

**Unit**
- Normalização de features, aplicação de pesos, cálculo de `z` e `score`.  
- Seleção do **threshold efetivo** (opType > assetType > global).  
- Modo `SHADOW`: registra divergências **sem** alterar decisão real.


---

## 📄 Documentação
- **Swagger** (tag `Dedup — Admin`) com exemplos sanitizados.  
- **Postman/Bruno** em `/docs/dedup/` (config/dry‑run/explain/eval).  
- `docs/dedup/tuning_guide.md`: guia rápido de tuning com passos: `baseline → shadow → eval → enforce`, leitura de métricas e *rollbacks* de versão.


---

## 🔧 Configurações (.env)
- `DEDUP_MODE=SHADOW`  
- `DEDUP_DEFAULT_SCORE_THRESHOLD=0.82`  
- `DEDUP_MAX_DATE_DELTA_DAYS=2`  
- `DEDUP_MAX_QTY_REL_DIFF=0.15`  
- `DEDUP_MAX_PRICE_REL_DIFF=0.03`  
- `DEDUP_DRY_RUN_LIMIT=100000`  
- `DEDUP_EVAL_TIMEOUT_SECONDS=120`  
- `OBS_MASK_CPF=true`


---

## ✅ Entregáveis
- Domain + Service + Repository + Controllers + migrations.  
- Testes unitários e de integração.  
- Logs, métricas, Swagger e collections.  
- **Aceite**: criar nova versão com `SHADOW`, rodar `dry-run`, inspecionar `explain`, subir alguns rótulos e executar `eval` obtendo métricas; alternar para `ENFORCED` e observar decisões refletidas sem regressão de performance.
