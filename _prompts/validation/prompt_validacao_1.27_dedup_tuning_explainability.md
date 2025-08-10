# Prompt de Validação 1.27v — Dedup Tuning & Explainability — **suno-wallets**

## 🎯 Objetivo
Validar que o **tuning de dedup** (config por tenant, modos `OFF|SHADOW|ENFORCED`), o **modo sombra**, a **explainability por candidato**, o **dry‑run** e a **avaliação com rótulos** estão implementados, auditáveis, performáticos e integrados à **policy (1.21)**, Backoffice (1.22) e pipeline do 1.19 — com **observabilidade**, **RBAC**, **multi‑tenant** e **sem PII**.

> Escopo: migrations (`dedup_config`, `dedup_eval_labels`, `dedup_eval_runs`, `dedup_candidate_explanations`), services/repos/controllers (config/dry‑run/explain/eval), integração com candidatos do 1.19, métricas Prometheus, logs, Swagger/Collections, testes unitários e de integração.


---

## ✅ Checklist de Validação

### 1) Arquitetura & DDD
- [ ] Artefatos em **inglês** com comentários **pt‑BR**:
  - `src/Domain/Dedup/dedup_config.go` (config structs, enums, thresholds, windows)
  - `src/Application/Dedup/dedup_explainer.go` (score + explicação por feature)
  - `src/Application/Dedup/dedup_tuner_service.go` (get/set/dryRun/evaluate)
  - `src/Infrastructure/Dedup/dedup_config_repository.go`
  - `src/Infrastructure/Dedup/dedup_eval_repository.go`
  - `src/Api/Controllers/Dedup/dedup_admin_controller.go`
  - Testes: `src/Tests/Unit/Dedup/...`, `src/Tests/Integration/Dedup/...`
- [ ] Controllers só **orquestram**; lógica em **Service**; BD via **Repository**.
- [ ] Integração com **candidates** do 1.19 (sem duplicar código).

### 2) Banco & Migrations
- [ ] `dedup_config` com `version` único por tenant, `mode`, `params(JSONB)`, `is_active`.
- [ ] `dedup_eval_labels` com `pair_key` único, `is_duplicate`, `source`, `notes?`.
- [ ] `dedup_eval_runs` com `config_version`, `dataset_info`, `metrics`.
- [ ] `dedup_candidate_explanations` com `score`, `threshold`, `features(JSONB)` (cache opcional).
- [ ] Índices: `ix_dedup_cfg_active`, `ix_dedup_labels`, `ix_dedup_eval_runs` criados.

### 3) Scoring & Explainability
- [ ] Normalização das **features** em faixa [-1..1].
- [ ] Score logístico `score = 1/(1+e^(−Σ w_i f_i))`.
- [ ] **Threshold efetivo**: prioridade `ByOperationType > ByAssetType > Global`.
- [ ] `explain` retorna **contribuição por feature** (`w_i * f_i`), `score`, `threshold`, `decision`.

### 4) Policy & Modos
- [ ] **HYBRID**: dedup ativo; **B3_ONLY|MANUAL_ONLY**: **skip** com métrica.
- [ ] `mode=OFF`: pipeline não decide, só lista candidatos.
- [ ] `mode=SHADOW`: registra **shadow divergences** sem alterar decisão real.
- [ ] `mode=ENFORCED`: aplica decisão conforme threshold.

### 5) Endpoints (Swagger tag: `Dedup — Admin`)
- [ ] `GET /admin/dedup/config` → versão ativa do tenant.
- [ ] `POST /admin/dedup/config` cria **nova versão** (não edita a anterior).
- [ ] `POST /admin/dedup/config/dry-run` simula impacto (janela, filtros).
- [ ] `GET /admin/dedup/candidates?cpf=&status=&minScore=&cursor=&limit=` com **keyset pagination**.
- [ ] `GET /admin/dedup/candidates/{id}/explain` retorna explicação detalhada.
- [ ] `POST /admin/dedup/eval/labels` faz upsert de rótulos.
- [ ] `POST /admin/dedup/eval/run` executa avaliação e retorna `precision/recall/F1/PR-curve`.
- [ ] `GET /admin/dedup/eval/runs` lista execuções.
- [ ] Códigos: 200/201/204/400/401/403/404/409/429/5xx; mensagens claras.
- [ ] **Swagger/Postman/Bruno** atualizados em `/docs/dedup/`.

### 6) Observabilidade
**Métricas Prometheus**
- [ ] `dedup_candidates_total{status}`
- [ ] `dedup_candidate_scores_bucket` (histogram)
- [ ] `dedup_decisions_total{decision}`
- [ ] `dedup_mode_info{mode}`
- [ ] `dedup_threshold_effective{level}`
- [ ] `dedup_shadow_divergences_total`
- [ ] `dedup_eval_precision{config_version}`, `dedup_eval_recall{...}`, `dedup_eval_f1{...}`
**Logs**
- [ ] JSON: `tenantId`, `configVersion`, `mode`, `candidateId/pairKey`, `score`, `threshold`, `decision`, `shadowDivergence?`, `durationMs`.
- [ ] **Sem PII**; mascarar quando necessário.

### 7) Segurança & LGPD
- [ ] **`X-Tenant-Id` obrigatório**; **RBAC**: `ADMIN` (set/dry-run/eval), `SUPPORT` (leitura/inspeção).
- [ ] Rate‑limit para rotas pesadas; validação de `pairKey/candidateId`.
- [ ] Respostas **sem CPF**; usar chaves estáveis.

### 8) Performance
- [ ] **Keyset** por `(tenant_id, score DESC, candidate_id)`; sem *offset*.
- [ ] `dry-run` e `eval` em **lotes** com limites/timeout configuráveis.
- [ ] *Caching* leve para `explain` recente.

### 9) Testes Automatizados
**Integration — `src/Tests/Integration/Dedup/dedup_tuning_explainability_test.go`**
- [ ] `POST /admin/dedup/config` cria versão e `GET /.../config` retorna a ativa.
- [ ] `dry-run` retorna `scoreHistogram`, `shadowDivergences` coerentes ao mudar threshold.
- [ ] `GET /.../candidates?...` pagina por **keyset** e respeita `minScore`.
- [ ] `GET /.../candidates/{id}/explain` soma de contribuições condiz com `z`/`score`.
- [ ] `policy=HYBRID` ativa; `B3_ONLY|MANUAL_ONLY` → **skip** e métrica aumenta.
- [ ] `POST /.../eval/labels` upsert; `POST /.../eval/run` retorna métricas consistentes.
**Unit**
- [ ] Normalização, pesos, score; escolha do threshold efetivo; comportamento `OFF/SHADOW/ENFORCED`.

### 10) Documentação
- [ ] Swagger + coleções Postman/Bruno versionadas.
- [ ] `docs/dedup/tuning_guide.md` (baseline → shadow → eval → enforce; leitura de métricas; rollback).


---

## 🧪 Passos de Validação Manual (rápidos)

1) **Criar versão (SHADOW)**  
```bash
curl -X POST "$BASE_URL/admin/dedup/config" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"mode":"SHADOW","thresholds":{"global":0.82},"window":{"maxDateDeltaDays":2,"maxQtyRelDiff":0.15,"maxPriceRelDiff":0.03},"weights":{"SameTicker":1.2,"OperationTypeMatch":0.8,"DateDeltaDays":-1.5,"QuantityRelDiff":-2.0,"PriceRelDiff":-2.0}}'
```

2) **Dry‑run**  
```bash
curl -X POST "$BASE_URL/admin/dedup/config/dry-run" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"from":"2024-01-01","to":"2024-12-31","assetTypes":["EQUITY"],"operationTypes":["BUY","SELL"]}'
```

3) **Inspeção & Explain**  
```bash
curl "$BASE_URL/admin/dedup/candidates?minScore=0.75&limit=20" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
curl "$BASE_URL/admin/dedup/candidates/<id>/explain" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN"
```

4) **Avaliação (labels + run)**  
```bash
curl -X POST "$BASE_URL/admin/dedup/eval/labels" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '[{"pairKey":"abc123","isDuplicate":true,"source":"OPERATOR"}]'

curl -X POST "$BASE_URL/admin/dedup/eval/run" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"from":"2024-01-01","to":"2024-03-31"}'
```

5) **Trocar para ENFORCED**  
```bash
curl -X POST "$BASE_URL/admin/dedup/config" \
  -H "Content-Type: application/json" -H "X-Tenant-Id: $TENANT" -H "Authorization: Bearer $TOKEN" \
  -d '{"mode":"ENFORCED","thresholds":{"global":0.82}}'
```

---

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.27 (Dedup Tuning & Explainability) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
