# Prompt de Validação 1.26v — Admin UI Minimal (Read‑only) — **suno-wallets**

## 🎯 Objetivo
Validar que a **Admin UI (somente leitura)** consome corretamente as APIs da Fase 1 (1.20, 1.21, 1.22, 1.23, 1.24, 1.25), respeita **multi‑tenant**, **LGPD** (CPF mascarado), usa **keyset pagination**, exibe **RBAC visual** (ações desabilitadas), permite **export** da timeline e apresenta um **painel de status/health** — com testes unitários e E2E (smoke), documentação e build.

## ✅ Checklist de Validação

### 1) Projeto & Stack
- [ ] Projeto criado em `/ui/admin` com **Next.js 14 (App Router) + TypeScript**, **TailwindCSS**, **shadcn/ui**, **TanStack Query**, **Axios**, **Zod**.
- [ ] Estrutura conforme planejado (páginas, `components`, `lib`, `hooks`, `types`, `tests`, `e2e`).
- [ ] Comentários no código em **pt‑BR**; nomes de arquivos/classes/funções em **inglês**.
- [ ] Scripts no `package.json`: `dev`, `build`, `start`, `test`, `e2e`.

### 2) Configuração & Env
- [ ] `.env.example` inclui:  
  - `NEXT_PUBLIC_API_BASE_URL`  
  - `NEXT_PUBLIC_TENANT_ID`  
  - `NEXT_PUBLIC_ENABLE_EXPORT`  
- [ ] Leitura de env em tempo de build e runtime OK (sem expor segredos adicionais).

### 3) Interceptors & Autorização
- [ ] **Axios** injeta `Authorization: Bearer <token>` e `X-Tenant-Id` em todas as requisições.
- [ ] Token armazenado em **sessionStorage** (não em URL, não em localStorage).
- [ ] Tratamento de erros 401/403/429 com *toasts* amigáveis e **redirect** para `/login` quando apropriado.
- [ ] **Nunca** logar token/PII no console.

### 4) Páginas (Read‑only)
- [ ] **/login**: formulário simples de token + tenant, mostra *role* estática (ex.: `SUPPORT`). Persistência em `sessionStorage`.
- [ ] **/search**: consome `GET /admin/backoffice/search?query=`; tabela com **CPF mascarado**, policy e última ingestão; link para Profile.
- [ ] **/clients/[cpf]/profile**: consome `GET /admin/backoffice/profile?cpf=` e `GET /admin/client/policy?cpf=`; cards 360° (b3, inconsistências, system ops, manual ops, dedup, performance, últimas ações). **Todos os botões de ação desabilitados** com *tooltip* “Disponível em etapa futura”.
- [ ] **/clients/[cpf]/timeline**: consome `GET /ops/timeline` com **keyset pagination** (`nextCursor`) e filtros; `respectPolicy` habilitado por padrão; **Export** via `GET /ops/timeline/export?format=csv|json&limit=...`.
- [ ] **/clients/[cpf]/inconsistencies**: lista por tipo/status (fonte do 1.17); links para abrir timeline filtrada.
- [ ] **/clients/[cpf]/catalog**: `GET /catalog/ca/history?ticker=` e `GET /catalog/ca/projection/status?cpf=`.
- [ ] **/health**: consome `GET /observability/status`; cards com versão, uptime, DB, B3 quotas, jobs, ledger, cache, trace sample rate.
- [ ] Estados de **loading**, **empty** e **error** visíveis em todas as páginas.

### 5) UX & Acessibilidade
- [ ] **CPF sempre mascarado** na UI (`***1234`); componente utilitário reutilizado.
- [ ] Botões de ação possuem `disabled`, *tooltip* e atributos `aria-disabled`/`title` coerentes.
- [ ] Foco visível; contraste AA; navegação por teclado nas tabelas/listas.
- [ ] `KeysetPager` com botão **“Load more”** e preservação de filtros.

### 6) Segurança & LGPD
- [ ] `X-Tenant-Id` obrigatório em todas as chamadas (validação no interceptor).
- [ ] **Sem** PII em logs do navegador; **sem** CPF completo em nenhuma tela.
- [ ] Não persistir credenciais além de `sessionStorage` (limpeza no logout).
- [ ] **Sem** executar POST/PUT/DELETE nesta etapa (somente GET/Export).

### 7) Integrações de API
- [ ] Todas as rotas citadas respondem com contratos esperados (via backend real ou *mocks* de rede no CI).
- [ ] Export dispara *download* e respeita `Content-Disposition` (nome de arquivo).
- [ ] Erros e timeouts exibem mensagens amigáveis (sem vazar detalhes internos).

### 8) Testes Automatizados
**Unit (Vitest + RTL)**
- [ ] `cpf.ts` mascara corretamente vários formatos de entrada.
- [ ] `pagination.ts` compõe/consome `cursor` e evita duplicação de itens.
- [ ] `api.ts` injeta headers e trata 401/403/429 conforme esperado.
- [ ] Componentes críticos (`DataTable`, `KeysetPager`, `ExportButton`, `RoleGuard`) cobertos.

**E2E (Playwright — smoke)**
- [ ] Fluxo: login → search → abrir profile → timeline (aplicar filtro) → export → health.
- [ ] Verificar que **todas as ações** (full fetch, incremental, recon-scan, etc.) aparecem **desabilitadas** com *tooltips*.
- [ ] Mocks de rede em CI (sem gravar dados fake) — validação de **contratos** (shape/fields).

### 9) Build & Deploy
- [ ] `pnpm build`/`npm run build` finaliza sem erros; aplicação inicia com `start`.
- [ ] Dockerfile funcional (produção) e **imagem leve**.
- [ ] `README.md` explica setup, env, execução de testes e E2E.

### 10) Documentação
- [ ] `docs/admin-ui/overview.md` com fluxos, prints de referência e como configurar base URL/tenant.
- [ ] Indicação de links para Grafana/Prometheus (texto estático configurável).

## 🧪 Passos de Validação Manual (rápidos)

1) **Setup local**
```
cp /ui/admin/.env.example /ui/admin/.env.local
# Ajuste NEXT_PUBLIC_API_BASE_URL e NEXT_PUBLIC_TENANT_ID
pnpm install && pnpm dev
```
Abra `http://localhost:3000/ui/admin`.

2) **Login**
- Informe `token` de teste e `tenant`; confirme que o papel aparece (ex.: SUPPORT).

3) **Search**
- Busque por um CPF; verifique **CPF mascarado** e link para Profile.

4) **Profile 360°**
- Confirme cards com dados coerentes (b3, inconsistências, system ops, manual ops, dedup, performance, últimas ações).
- Passe o mouse nos **botões de ação** e veja *tooltips* e `disabled`.

5) **Timeline**
- Aplique filtros; carregue a **2ª página** com **keyset (nextCursor)** sem itens duplicados.
- Clique em **Export** e valide o arquivo baixado.

6) **Inconsistencies / Catalog**
- Verifique listagem por tipo/status; abra histórico de CA e status de projeções.

7) **Health**
- Verifique cartões com o `/observability/status`.

## 📌 Resultado Esperado
Se **tudo** estiver conforme, responda:
```
Validação concluída: Etapa 1.26 (Admin UI — Read‑only) está 100% conforme.
```
Caso contrário, liste **exatamente** os itens pendentes e a **correção objetiva** para cada um.
