# Prompt 1.26 — Admin UI Minimal (Read‑only) — **suno-wallets**

Implemente uma **UI administrativa mínima, somente leitura**, para **investigação operacional** usando as APIs já entregues (1.20, 1.21, 1.22, 1.23, 1.24, 1.25).  
**Sem ações destrutivas** aqui (botões de ação devem aparecer desabilitados com *tooltips*), foco em **consulta, diagnóstico e export**.

> Código/nomes em **inglês**; comentários em **pt‑BR**. **Sem dados fake**. Multi‑tenant estrito. **CPF sempre mascarado na UI**.  
> O front‑end deve ser isolado em `/ui/admin` (monorepo) ou em um *workspace* separado, com build independente.


---

## 🎯 Objetivos
1) **Pesquisar** cliente por CPF (mascarado) e navegar para um **Profile 360°** (read‑only).  
2) Visualizar **Timeline** unificada com **keyset pagination** e filtros.  
3) Ver **Reconciliation Summary**, **Inconsistencies**, **Policy**, **CA Status**.  
4) **Export** (CSV/JSON) da timeline usando o endpoint 1.20.  
5) Painel de **Saúde/Status** do serviço consumindo `/observability/status` (1.25).  
6) **RBAC visual**: exibir papéis e **desabilitar** ações quando não permitidas (nesta etapa, todas as ações ficam desabilitadas).


---

## 🧱 Stack & Estrutura
- **Next.js 14 (App Router) + TypeScript**
- **TailwindCSS** + **shadcn/ui** (botões, cards, tabelas)  
- **TanStack Query** (cache/fetch) + **Axios** (interceptors)
- **Lucide** (ícones) + **Zod** (validação de schemas)
- Testes: **Vitest + React Testing Library** (unit), **Playwright** (e2e smoke)

**Pasta (monorepo ou subpasta)**:
```
/ui/admin
  /src
    /app
      /(public)
        /login
        /search
        /health
      /(client)
        /clients/[cpf]/profile
        /clients/[cpf]/timeline
        /clients/[cpf]/inconsistencies
        /clients/[cpf]/catalog
    /components
    /lib
      api.ts          # axios + interceptors
      auth.ts         # auth stub (read-only)
      cpf.ts          # mask util
      pagination.ts   # keyset helpers
      rbac.ts         # roles read-only
    /hooks
    /types
    /tests
  /e2e
  .env.example
  next.config.js
  tailwind.config.ts
  README.md
```

> **Comentários do código em pt‑BR** explicando cada componente/página/serviço.


---

## 🔌 Integrações com APIs existentes

### Interceptors (Axios)
- Adicionar `Authorization: Bearer <token>` (armazenado em `sessionStorage`), e `X-Tenant-Id` vindo de `.env`/config.  
- `onError`: mapear 401/403/429 e exibir *toasts* com mensagens amigáveis (sem PII).

### Endpoints consumidos
- **1.22 — Backoffice Admin**  
  - `GET /admin/backoffice/search?query=` → página **/search**  
  - `GET /admin/backoffice/profile?cpf=` → **Profile 360°**
- **1.20 — Timeline & Export**  
  - `GET /ops/timeline` com keyset (`cursor`) + filtros; **Export** em `GET /ops/timeline/export`
- **1.21 — Client Policy**  
  - `GET /admin/client/policy?cpf=` (mostrar *badge* do modo atual)
- **1.17 — Inconsistencies**  
  - `GET /ops/inconsistencies?cpf=` (ou fonte equivalente definida no 1.17)
- **1.23 — Catalog**  
  - `GET /catalog/ca/history?ticker=` (aba “Catalog & CA History”)
- **1.24 — CA Projection**  
  - `GET /catalog/ca/projection/status?cpf=` (somente leitura)
- **1.25 — Status/Health**  
  - `GET /observability/status` (dashboard **/health**)

> **Importante**: nenhum POST/PUT/DELETE nesta etapa; somente GETs e **download** de export via link.


---

## 🖥️ Páginas & UX

### Login (mock, read‑only)
- Form simples de **token** + **tenant** (armazenar em `sessionStorage`); exibir *role* (ex.: `SUPPORT`) como rótulo estático.  
- **Sem** back-end de auth nesta etapa.

### Search `/search`
- Campo único de busca (CPF ou parte dele, nome se disponível).  
- Tabela com resultados: CPF **mascarado**, data da última ingestão B3, policy atual, link para **Profile**.

### Profile 360° `/clients/[cpf]/profile`
- Cards com: policy, últimas janelas B3, inconsistências abertas (contagem), system ops criadas, manual ops, dedup (contagens), tempo médio de full/incremental, últimas ações.  
- Botões de **ação** (full fetch, incremental, recon-scan, auto-fix, dedup, zero-and-refetch) **desabilitados**, com tooltip “Disponível na etapa de ações (futuro)”.

### Timeline `/clients/[cpf]/timeline`
- Filtros: período, tickers, sources, assetTypes; **respectPolicy** (on por padrão).  
- Tabela paginada por **keyset** (`nextCursor`).  
- **Export**: botão que usa `/ops/timeline/export?format=csv|json&limit=...`.  
- Mostrar colunas: `operationDate`, `canonicalTicker/originalTicker`, `operationType`, `source`, `quantity`, `unitPrice?`, `reasonCode`, `caVersionApplied?`.

### Inconsistencies `/clients/[cpf]/inconsistencies`
- Lista por tipo/status, com *badges* e datas. Linka para **Timeline** filtrada.

### Catalog & CA History `/clients/[cpf]/catalog`
- Campo “ticker” → lista histórico de eventos de identidade.  
- Bloco “CA Projection status (read‑only)” por versão.

### Health `/health`
- Cards do `/observability/status`: versão, uptime, DB, B3 quotas, jobs, ledger, cache, trace sample rate.  
- Links rápidos para Grafana (texto estático configurável).


---

## 🧩 Componentes & Libs
- `DataTable` (shadcn + TanStack Table) com **virtualização** (se necessário) e linhas compactas.  
- `KeysetPager` com botão **“Load more”** usando `nextCursor`.  
- `CpfMasked` utilitário (`***1234`).  
- `RoleGuard` (só read‑only; exibe tooltips em ações).  
- `ExportButton` com *spinner* e *Content-Disposition* filename.

**Acessibilidade**: focos visíveis, `aria-*` nos botões desabilitados, contraste AA.


---

## 🔐 Segurança & LGPD
- **Nunca** renderizar CPF completo em telas; usar máscara `***1234`.  
- **Sem** gravar tokens em logs; **sem** tokens na URL.  
- `X-Tenant-Id` obrigatório em todas as chamadas; validar presença no interceptor.  
- Tratar 401/403 com *redirect* para `/login` (sem perder estado de navegação).


---

## 🧪 Testes
**Unit (Vitest + RTL)**
- `CpfMasked` mascara corretamente.
- `KeysetPager` compõe `cursor` com paginação incremental.
- `api.ts` injeta headers e trata 401/403/429.

**E2E (Playwright — smoke)**
- Fluxo: login → search → abrir profile → timeline (aplicar filtro) → export.  
- Verificar que **ações estão desabilitadas** e tooltips aparecem.  
- Health page carrega campos principais de `/observability/status`.

> Em CI, rodar e2e com *mocks* de rede (sem dados fake gravados) — interceptar chamadas e validar contratos.


---

## ⚙️ Configuração & Build
- `.env.example`:
  - `NEXT_PUBLIC_API_BASE_URL=http://localhost:8080`
  - `NEXT_PUBLIC_TENANT_ID=<uuid>`
  - `NEXT_PUBLIC_ENABLE_EXPORT=true`
- Scripts:
  - `dev`, `build`, `start`, `test`, `e2e`
- Dockerfile de produção (Node 20 + distroless/alpine) e `nginx.conf` opcional para *reverse-proxy*.

---

## 📄 Documentação
- `README.md` com **setup rápido**, variáveis de ambiente, como rodar testes e e2e.  
- `docs/admin-ui/overview.md` (fluxos de navegação, prints de referência, acessos a Grafana/Prometheus).


---

## ✅ Entregáveis
- App Next.js completo em `/ui/admin` (read‑only).  
- Testes unitários + e2e (smoke).  
- **Sem** ações de escrita nesta etapa; todos os botões de ação desabilitados.  
- **Aceite**: é possível buscar um CPF, abrir o profile 360°, navegar Timeline com **keyset**, exportar CSV e visualizar o status do sistema — **sem vazar PII** e com `X-Tenant-Id`/token corretos em todas as requisições.
