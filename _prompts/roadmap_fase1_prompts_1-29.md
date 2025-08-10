# Roadmap — Fase 1 (B3 Ingest, Normalização, Reconciliação & Admin) — **suno-wallets**

**Atualizado:** 2025-08-10 • **Legenda:** ✅ Concluído · ⏳ Pendente

Este roadmap lista os **prompts 1.1 → 1.29** com um **resumo curto** de cada um.  
Por solicitação: **marcados como concluídos de 1.1 a 1.22**.

---

## Visão Geral
- **Concluídos:** 1.1–1.22 (22/29)
- **Pendentes:** 1.23–1.29

---

## Lista de Prompts

| ID | Título | Resumo | Status |
|---:|---|---|:---:|
| 1.1 | Repo Bootstrap, DDD & Cursor Guidelines | Estrutura inicial (DDD), módulos, lint, CI básico; nomes em inglês, comentários pt‑BR; pasta `/Tests` e docs. | ✅ |
| 1.2 | Database Base & RAW Tables | Migrations iniciais; tabelas `b3_raw_payloads_*`, índices por tenant/CPF/data; versions de schema. | ✅ |
| 1.3 | Config & Secrets Loader | `.env`, variáveis sensíveis, loader tipado; chaves B3 (ClientId/Secret/Cert). | ✅ |
| 1.4 | Errors, Logging & Tracing Base | Helper de erros, logs JSON (tenant, cpfMasked), OpenTelemetry inicial. | ✅ |
| 1.5 | Observability Baseline | Prometheus métricas core, health/liveness base, dashboards seeds e alertas sugeridos. | ✅ |
| 1.6 | B3 Auth & mTLS Client | OAuth2 (PKCE/Client Credentials), mTLS `.p12`, retries/backoff, headers obrigatórios. | ✅ |
| 1.7 | B3 Data Fetchers | Wrappers oficiais: **positions v3** e **transactions v2** com paginação e validações. | ✅ |
| 1.8 | Persistência de RAW | Serviço que guarda payloads B3 (auditoria/reprocesso), dedupe de páginas, metadados. | ✅ |
| 1.9 | Normalization Pipeline | Extrai JSON → tabelas normalizadas (transações/posições) sem tratamento de negócio. | ✅ |
| 1.10 | Historical Backfill (Full) | Job para novo cliente: coleta em lotes (mês a mês) desde o 1º dia da API B3. | ✅ |
| 1.11 | Canonical Ledger Builder | Consolida operações (B3 + manual + sintéticas) em ledger único, idempotente. | ✅ |
| 1.12 | Daily Incremental Fetch Job | Scheduler noturno para todos ativos; checkpoints e políticas de janela. | ✅ |
| 1.13 | Zero‑and‑Refetch (Admin) | Endpoint para resetar cálculo (preserva RAW) e refazer coleta completa. | ✅ |
| 1.14 | Incremental Re‑run (Admin) | Endpoint para reexecutar apenas a janela incremental a partir do último checkpoint. | ✅ |
| 1.15 | Health & Status API | `/health/ready` e `/observability/status` com quotas B3, DB, jobs e trace rate. | ✅ |
| 1.16 | Cache, Rate‑limit & Resiliência | Cache central (B3/DB), rate‑limit, circuit breaker e timeouts parametrizados. | ✅ |
| 1.17 | Inconsistency Detector (Read‑only) | Detecta “posição sem compra”, “venda sem compra”, e registra em `b3_inconsistencies`. | ✅ |
| 1.18 | System Synthetic Ops (Auto‑fix) | Gera operações neutras para fechar buracos pré‑B3; versionadas e auditáveis. | ✅ |
| 1.19 | Manual Ops & Dedup Core | CRUD seguro de transações manuais + dedupe básico orientado à policy. | ✅ |
| 1.20 | Timeline & Export API | Linha do tempo consolidada (keyset), filtros, `export` CSV/JSON. | ✅ |
| 1.21 | Client Policy & Modes | Política por cliente/tenant: `B3_ONLY`, `MANUAL_ONLY`, `HYBRID`; toggles e auditoria. | ✅ |
| 1.22 | Backoffice Admin APIs (MVP) | Busca/Perfil/Timeline/Status para suporte; pronto para Grafana/Bruno/Postman. | ✅ |
| 1.23 | Catalog: Identity & CA History | Catálogo de tickers/identidade e histórico de **Corporate Actions** (splits, renames). | ⏳ |
| 1.24 | CA Projection Engine | Preview/apply de CA versionada com **comicota** (arredondar para baixo) e neutralidade. | ⏳ |
| 1.25 | Health & Observability Finale | Consolida status, links de dashboards, métricas finais do bloco (p95, quotas). | ⏳ |
| 1.26 | Admin UI Minimal (Read‑only) | Next.js/Tailwind/shadcn UI só‑leitura: Profile 360°, Timeline, Inconsistências, Health. | ⏳ |
| 1.27 | Dedup Tuning & Explainability | Tuning por tenant (shadow/enforced), explain por feature, dry‑run e avaliação (PR‑curve). | ⏳ |
| 1.28 | LGPD & Retenção Manual (on‑demand) | Export/eliminação manual com dry‑run, dupla confirmação, recibo e tombstone/blocklist. | ⏳ |
| 1.29 | Smoke E2E — Reconciliação & CA | Suite E2E ponta‑a‑ponta: full→incremental→recon→auto‑fix→override→CA→export→OBS. | ⏳ |

---

## Próximos Passos Recomendados
1. **1.23–1.24**: concluir catálogo e motor de CA (pré‑requisito para consistência histórica).  
2. **1.25–1.26**: finalizar observabilidade e UI de suporte para acelerar operação.  
3. **1.27–1.28**: qualidade/privacidade (dedup avançado e LGPD).  
4. **1.29**: fechar com **E2E Smoke** e gates de performance.

> Todos os prompts devem manter: DDD, multi‑tenant, LGPD (CPF mascarado), **sem dados fake**, métricas/logs/tracing, Swagger/Bruno/Postman e testes (unit + integração).

