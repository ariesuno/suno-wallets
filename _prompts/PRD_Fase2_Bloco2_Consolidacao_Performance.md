# PRD — Bloco 2 (Fase 2) — Consolidação de Carteira & Performance — **suno-wallets**

Este PRD orienta o Cursor (e a squad) a implementar a **Fase 2** do projeto `suno-wallets`.  
Reforços globais: nomes de arquivos/classes/funções em **inglês**; comentários em **pt‑BR**; **DDD**, **multi‑tenant**, **LGPD**, **observabilidade**, **testes**, **Swagger + Postman + Bruno** em todos os prompts.

## Contexto e Objetivos
- Consolidar dados de **ledger** (B3, manuais, sintéticos e CA) em **posições**, **snapshots diários** e **métricas de performance** (PnL, TWR, MWR).
- Processar **incrementalmente** (sem refazer histórico inteiro), mantendo **idempotência**, **auditoria** e **alto desempenho**.
- Expor **APIs** de consulta (resumo, posições, séries) e relatórios exportáveis.
- Fornecer **backoffice read‑only** para suporte/monitoramento operacional do bloco 2.

## Escopo (Inclui)
- Cálculo de custo médio, lotes e PnL.
- Snapshots EOD por ativo e por carteira.
- Projeção/aplicação de Corporate Actions (com *comicota* — **arredondar para baixo**).
- Proventos (dividendos/JCP) EOD; entradas manuais.
- Performance (TWR/MWR) e comparação com benchmarks (IBOV/IFIX).
- Reprocessamento direcionado e otimizações (partições, MV).
- APIs de leitura e exports; E2E com SLOs.

## Fora de Escopo (Agora)
- UI final para usuário investidor (aplicação cliente).  
- Intraday pricing, cálculos intraday.  
- Impostos/fisco e IRPF.  
- Recomendações ou otimização de carteira.

## Requisitos de Qualidade (globais)
- **SLO**: p95 das APIs ≤ 1.5s em ambiente padrão; ingest/processos assíncronos com *backoff* e *circuit breaker*.
- **Idempotência**: reexecuções não duplicam; uso de **versionamento** para CA e checkpoints por janela.
- **Observabilidade**: Prometheus, logs JSON correlacionados (requestId/traceId), tracing OTel; dashboards e alertas.
- **LGPD**: sem PII em métricas/labels; CPF **mascarado** em logs; respeitar tombstone/blocklist (1.28).
- **Multi‑tenant**: `X-Tenant-Id` obrigatório; filtragem por tenant no BD; RBAC em rotas admin.
- **Testes**: unitários + integração + e2e smoke do bloco 2 (2.20). Cobrir casos de erro e race.

## Arquitetura & DDD (pastas sugeridas)
```
/src
  /Domain
    /Portfolio       # entidades: Position, Lot, Snapshot, CashFlow, PnL, BenchmarkRef, FxRateRef, CaApplied
  /Application
    /Portfolio       # services: CostBasisEngine, SnapshotService, PnLService, PerformanceService, Reprocessor
  /Infrastructure
    /Pricing         # price/FX providers, caches
    /Portfolio       # repos: positions, snapshots, pnl, series; migrations
  /Api
    /Controllers/Portfolio  # endpoints 2.12–2.14
  /Tests
    /Unit/Portfolio
    /Integration/Portfolio
```

## Modelos de Dados (alto nível)
- `positions_daily`: tenant, cpf_hash, date, ticker, qty, avg_price, market_value, realized_pnl_today, unrealized_pnl, ca_version, updated_at.
- `lots`: tenant, cpf_hash, ticker, lot_id, qty, unit_cost, opened_at, closed_at, source.
- `pnl_daily`: tenant, cpf_hash, date, realized, unrealized, fees, dividends, contributions, withdrawals.
- `portfolio_series`: chaveada por (tenant, cpf_hash, metric, date) → valuation, twr, mwr, contributions.
- `benchmarks_eod`: index, date, close, return.
- `prices_eod`: ticker, date, close, source, loaded_at.
- `fx_eod`: pair, date, rate, source.

> Campos **cpf_hash** (nunca CPF puro); índices por `(tenant_id, cpf_hash, date)`; partições por **date** onde fizer sentido.

## Roadmap de Prompts (2.x)

### 2.1 — Domain Model da Consolidação
- Objetivo: definir entidades, VOs e contratos do domínio de consolidação; regras de integridade; interfaces de repositórios.
- Entregas: modelos, interfaces, erros de domínio, migrações iniciais (tabelas base).  
- DoD: testes unitários de invariantes; Swagger dos contratos internos (se expostos).

### 2.2 — Price Provider (Fechamento EOD)
- Objetivo: cliente/caching para EOD de EQUITY/FII/ETF (BRL).
- Entregas: `PriceProvider`, cache TTL, retries/backoff, métricas `price_requests_total`, `price_cache_hits_total`.
- DoD: integração fakeable; testes de latência/timeout; sem dados dummy gravados.

### 2.3 — FX Provider (USD/BRL etc.)
- Objetivo: taxas EOD (PTAX/alternativo) e cache EOD.
- Entregas: `FxProvider`, normalização para BRL; métricas `fx_requests_total`.
- DoD: precisão decimal, datas úteis vs corridas, testes de borda.

### 2.4 — Engine de Custo Médio & Lotes
- Objetivo: cálculo de custo médio (BUY/SELL), abertura/fechamento de `lots`, taxas opcionais.
- Entradas: ledger consolidado (1.20), CA aplicadas.
- Saídas: `positions` + `lots` intermediários; eventos de fechamento/realized PnL parcial.
- DoD: unit tests determinísticos, *property-based* para BUY/SELL/partial closes.

### 2.5 — Snapshots Diários & Incrementais
- Objetivo: gerar `daily snapshots` EOD e incremental por janela afetada.
- Entregas: `SnapshotService`, política de reprocesso (dirty window), *checkpoints*.
- DoD: idempotência; testes de reprocessamento por ticker/intervalo.

### 2.6 — Realized/Unrealized P&L
- Objetivo: cálculo diário de PnL realizado e não-realizado, com taxas e proventos.
- DoD: reconciliação básica com ledger e snapshots; testes de exemplos.

### 2.7 — Day Trade & Tagging
- Objetivo: marcar operações same‑day e ajustar custo médio por regra de day trade.
- DoD: testes de detecção (compra/venda mesmo dia) e impacto em PnL.

### 2.8 — Corporate Actions no Cálculo
- Objetivo: aplicar CA de versão N (1.24) nos cálculos; respeitar **floor** (sem fração).
- DoD: idempotência por versão; *fraction adjustments* coerentes.

### 2.9 — Proventos (Dividendos/JCP)
- Objetivo: registrar proventos EOD; refletir em `cashflows` e PnL.
- DoD: inputs manuais e de catálogo; conciliação mínima por data ex‑direito/pagamento.

### 2.10 — Métricas de Performance (TWR/MWR)
- Objetivo: séries TWR (composição) e MWR/IRR; normalização em BRL.
- DoD: testes com **golden cases**; precisão decimal; séries contínuas por data.

### 2.11 — Benchmarks (IBOV/IFIX)
- Objetivo: armazenar EOD e comparar retorno da carteira vs índice.
- DoD: API para retorno relativo; drawdown básico por janela.

### 2.12 — API de Consolidação — Resumo
- Objetivo: `GET /portfolio/summary?date=` e `/positions?date=`.
- DoD: resposta sem PII; paginação por keyset; export CSV.

### 2.13 — API de Consolidação — Séries Temporais
- Objetivo: `GET /portfolio/timeseries/valuation`, `/twrr`, `/contributions`.
- DoD: filtros por janela e agregações; export; limites de amostragem.

### 2.14 — Reprocessamento Direcionado
- Objetivo: reprocessar **ticker**, **intervalo** ou após **CA/provento**; auditoria e checkpoints.
- DoD: evitar *fan‑out*; rate‑limit; métricas de *backfill*.

### 2.15 — Consistência B3 vs Snapshot
- Objetivo: comparar posição B3 v3 *as‑of* vs snapshot; explicar desvios.
- DoD: relatório com causas prováveis (CA, atrasos, arredondamentos).

### 2.16 — Otimizações & MV
- Objetivo: partições por data/tenant, índices, **materialized views** para snapshots/PnL.
- DoD: *EXPLAIN ANALYZE* antes/depois; p95 estável; custos reduzidos.

### 2.17 — Backoffice Read‑only — Consolidação
- Objetivo: telas para resumo diário, posições, divergências e séries; ações desabilitadas.
- DoD: integração com 1.26 (read‑only); CPF mascarado; *keyset*.

### 2.18 — Test Harness & Golden Cases
- Objetivo: cenários fixos para validar custo médio, CA, proventos, PnL e séries.
- DoD: testes determinísticos + *property-based*; *fixtures* versionados.

### 2.19 — Relatórios Operacionais
- Objetivo: exports por período (posições, P&L realizado, proventos, TWR/MWR).
- DoD: CSV/JSON com `Content-Disposition` e paginação por janelas.

### 2.20 — E2E Consolidação & SLOs
- Objetivo: suite E2E do bloco 2; gates p95, throughput e consistência B3.
- DoD: artifacts de sumário; CI gate; idempotência de ponta a ponta.

## Dependências e Ordem Recomendada
- 2.1 → 2.2/2.3 → 2.4 → 2.5 → 2.6/2.7 → 2.8/2.9 → 2.10/2.11 → 2.12/2.13 → 2.14 → 2.15 → 2.16 → 2.17 → 2.18 → 2.19 → 2.20.

## Critérios Gerais de DoD por Prompt
- Código DDD com nomes em inglês e comentários pt‑BR.
- Migrações e índices entregues e aplicáveis (*down* seguro quando couber).
- Métricas Prometheus, logs JSON com correlação, tracing OTel.
- Swagger + coleções Postman/Bruno atualizadas em `/docs/<area>/`.
- Testes unitários e de integração; *mocks* de rede no CI; sem dados fake gravados no BD.
- RBAC e `X-Tenant-Id` aplicados; sem PII em métricas/labels; CPF mascarado em logs.
- README/PLAYBOOK de cada área com **setup** e **troubleshooting**.

## Métricas e Alertas Sugeridos (novos no bloco 2)
- `cost_basis_runs_total{status}`, `snapshot_runs_total{status}`, `pnl_calc_runs_total{status}`
- `portfolio_series_points_total{metric}`
- `price_requests_total`, `price_cache_hits_total`, `fx_requests_total`
- `portfolio_api_p95_seconds` (histogram) por rota
- Alertas: p95 > limite por 15 min; erro em backfills; divergência B3 x snapshot acima do limite.

## Riscos e Mitigações
- **Cardinalidade elevada** em métricas: limitar labels; evitar CPF/ticker → usar agregações por rota/job/tenant.
- **Reprocessos caros**: *dirty windows* e reprocesso direcionado (2.14); MV (2.16).
- **Dados incompletos de preço/FX**: *fallbacks* e políticas de *stale data* documentadas.

## Variáveis de Ambiente (exemplos)
- `PRICE_EOD_BASE_URL`, `FX_EOD_BASE_URL`, `FX_BASE_CURRENCY=BRL`
- `PORTFOLIO_P95_LIMIT=1.5`, `OBS_MASK_CPF=true`
- `SNAPSHOT_DIRTY_WINDOW_DAYS=7`, `REPROCESS_RATE_LIMIT=5`

## Aceite Final do Bloco 2
- E2E 2.20 **PASS**, com artifacts e gates atendidos.
- Dashboards atualizados com séries/latências do bloco 2.
- APIs 2.12/2.13 estáveis e documentadas; exports funcionais.
- Divergência B3 x snapshot controlada e explicada (2.15).
