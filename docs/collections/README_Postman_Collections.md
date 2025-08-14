# Suno Wallets - Collections do Postman

## 📁 Organização Atualizada

A collection do Postman foi reorganizada em um arquivo unificado e bem estruturado: `Suno_Wallets_Complete_Collection.json`

### 🔄 Migração das Collections Antigas

As collections anteriores foram consolidadas em uma única collection organizadas por funcionalidade:

| Collection Antiga | Nova Organização |
|-------------------|------------------|
| `postman_collection.json` | ✅ Integrada na collection completa |
| `postman_b3_admin.json` | ✅ Seção "🔧 B3 - Administração" |
| `postman_b3_complete_sync.json` | ✅ Seção "🎯 B3 - Sincronização Completa" |
| `postman_b3_data.json` | ✅ Seções de Preview e Ingestão |
| `postman_backoffice_admin.json` | ✅ Seção "🏢 Backoffice Admin" |
| `postman_client_policy.json` | ✅ Seção "⚙️ Políticas de Cliente" |
| `postman_manual_ops.json` | ✅ Seção "🛠️ Operações Manuais" |
| `postman_reconciliation.json` | ✅ Seção "🔍 B3 - Reconciliação" |
| `postman_timeline_summary.json` | ✅ Seção "📊 Timeline & Sumários" |

## 📋 Estrutura da Nova Collection

### 🔍 Observabilidade
- Health Check, Readiness, Liveness
- Métricas Prometheus
- Documentação Swagger

### 👛 Carteiras
- CRUD completo de carteiras
- Gestão por proprietário

### 🔐 B3 - Autenticação & Conectividade
- Verificação de autenticação B3
- Testes de conectividade
- Validação de CPF

### 📊 B3 - Dados & Preview
- Preview de transações e posições
- Fetch sem persistência

### ⚙️ B3 - Ingestão & Normalização
- Ingestão histórica RAW
- Normalização de dados

### 🔄 B3 - Sincronização
- Sincronização básica/diária
- Status e últimas sincronizações
- Janela de sincronização

### 🎯 B3 - Sincronização Completa
- Orquestração inteligente
- Análise de status
- Dry runs

### ⚡ B3 - Processamento Assíncrono
- Jobs em fila
- Status de processamento
- Worker pool management

### 🔧 B3 - Administração
- Reset e refetch E2E
- Sincronização incremental forçada

### 🔄 B3 - Reativação
- Análise e execução de reativação
- Estratégias híbridas

### 📈 B3 - Relatórios
- Intervalos de dados RAW
- Resumos financeiros
- Lista de tickers

### 🔍 B3 - Reconciliação
- Scan de inconsistências
- Auto-fix de problemas
- Gestão de inconsistências

### 🛠️ Operações Manuais
- CRUD de operações manuais
- Gestão completa

### 🔄 Deduplicação
- Scan de duplicatas
- Resolução de candidatos

### 📊 Timeline & Sumários
- Timeline operacional
- Exportações
- Sumários de reconciliação

### ⚙️ Políticas de Cliente
- Gestão de políticas de fonte de dados
- Auditoria e dry runs

### 🏢 Backoffice Admin
- Perfis de cliente
- Busca avançada
- Ações administrativas
- Exportação de ledger

## 🚀 Como Usar

### 1. Importar a Collection

1. No Postman, clique em **Import**
2. Selecione o arquivo `Suno_Wallets_Complete_Collection.json`
3. A collection será importada com todas as pastas organizadas

### 2. Configurar Ambiente (IMPORTANTE)

**Para que as variáveis sejam preenchidas automaticamente, você DEVE importar um ambiente:**

1. Vá em **Environments** no Postman (ícone de engrenagem)
2. Clique em **Import** 
3. Selecione um dos arquivos de ambiente:
   - `Local.postman_environment.json` - Para desenvolvimento local
   - `Development.postman_environment.json` - Para ambiente de dev
   - `Production.postman_environment.json` - Para produção
4. **Ative o ambiente** clicando no dropdown no canto superior direito
5. Selecione o ambiente importado (ex: "Suno Wallets - Local")

> **⚠️ IMPORTANTE**: Sem um ambiente ativo, as variáveis como `{{baseUrl}}` e `{{tenantId}}` não serão substituídas!

### 3. Verificar/Ajustar Variáveis

Após importar o ambiente, você pode verificar e ajustar as variáveis conforme necessário:

1. No ambiente ativo, clique no ícone de "olho" 👁️ para ver as variáveis
2. Clique em **Edit** para modificar os valores se necessário

As variáveis incluídas são:

```json
{
  "baseUrl": "http://localhost:8080",
  "tenantId": "status_invest", 
  "testCpf": "12345678901",
  "adminSecret": ""
}
```

### 4. Ambientes Disponíveis

Recomenda-se criar ambientes específicos para cada contexto:

#### Ambiente Local
```json
{
  "baseUrl": "http://localhost:8080",
  "tenantId": "status_invest",
  "testCpf": "12345678901"
}
```

#### Ambiente Desenvolvimento
```json
{
  "baseUrl": "https://dev-api.suno-wallets.com",
  "tenantId": "dev_tenant",
  "testCpf": "11111111111"
}
```

#### Ambiente Produção
```json
{
  "baseUrl": "https://api.suno-wallets.com",
  "tenantId": "production_tenant",
  "testCpf": "99999999999"
}
```

## ✅ Cobertura de Endpoints

Todos os endpoints da aplicação estão cobertos na nova collection:

### Endpoints de Saúde ✅
- `GET /health` ✅
- `GET /ready` ✅
- `GET /live` ✅
- `GET /health/live` ✅
- `GET /health/ready` ✅
- `GET /health/details` ✅
- `GET /metrics` ✅
- `GET /swagger/*any` ✅

### Carteiras ✅
- `POST /api/v1/wallets` ✅
- `GET /api/v1/wallets` ✅
- `GET /api/v1/wallets/:id` ✅
- `PUT /api/v1/wallets/:id` ✅
- `DELETE /api/v1/wallets/:id` ✅
- `GET /api/v1/wallets/owner/:owner_id` ✅

### B3 - Conectividade ✅
- `GET /api/v1/b3/health/auth` ✅
- `GET /api/v1/b3/test/connection` ✅
- `GET /api/v1/b3/test/cpf-quick` ✅

### B3 - Dados ✅
- `GET /api/v1/b3/fetch/transactions/preview` ✅
- `GET /api/v1/b3/transactions/:cpf` ✅
- `GET /api/v1/b3/fetch/positions/preview` ✅
- `POST /api/v1/b3/fetch/historical` ✅
- `POST /api/v1/b3/normalize/run` ✅

### B3 - Sincronização ✅
- `POST /api/v1/b3/sync/run` ✅
- `GET /api/v1/b3/client/status` ✅
- `GET /api/v1/b3/client/last-sync` ✅
- `POST /api/v1/b3/sync/complete-ingestion` ✅
- `GET /api/v1/b3/sync/status-analysis` ✅
- `GET /api/v1/b3/client/sync-window` ✅

### B3 - Processamento Assíncrono ✅
- `POST /api/v1/b3/async/sync/complete-ingestion` ✅
- `GET /api/v1/b3/async/jobs/:jobId/status` ✅
- `GET /api/v1/b3/async/jobs` ✅

### B3 - Administração ✅
- `POST /api/v1/b3/admin/reset-and-refetch` ✅
- `POST /api/v1/b3/admin/incremental-from-last` ✅
- `POST /api/v1/b3/admin/reactivation/analyze` ✅
- `POST /api/v1/b3/admin/reactivation/execute` ✅
- `GET /api/v1/b3/client/reactivation/status` ✅

### B3 - Relatórios ✅
- `GET /api/v1/b3/client/raw-date-range` ✅
- `GET /api/v1/b3/client/summary` ✅
- `GET /api/v1/b3/client/tickers` ✅

### B3 - Reconciliação ✅
- `POST /api/v1/b3/reconciliation/scan` ✅
- `GET /api/v1/b3/reconciliation/inconsistencies` ✅
- `GET /api/v1/b3/reconciliation/inconsistencies/:id` ✅
- `POST /api/v1/b3/reconciliation/auto-fix` ✅
- `POST /api/v1/b3/reconciliation/auto-fix/:id` ✅

### Operações Manuais ✅
- `POST /api/v1/ops/manual` ✅
- `PUT /api/v1/ops/manual/:id` ✅
- `DELETE /api/v1/ops/manual/:id` ✅
- `GET /api/v1/ops/manual` ✅

### Timeline & Deduplicação ✅
- `GET /api/v1/ops/timeline` ✅
- `GET /api/v1/ops/timeline/export` ✅
- `GET /api/v1/ops/reconciliation/summary` ✅
- `POST /api/v1/ops/dedup/scan` ✅
- `POST /api/v1/ops/dedup/resolve` ✅
- `GET /api/v1/ops/dedup/candidates` ✅

### Políticas & Admin ✅
- `GET /api/v1/admin/client/policy` ✅
- `POST /api/v1/admin/client/policy` ✅
- `GET /api/v1/admin/client/policy/audit` ✅
- `POST /api/v1/admin/client/policy/dry-run` ✅
- `GET /api/v1/admin/backoffice/profile` ✅
- `GET /api/v1/admin/backoffice/search` ✅
- `GET /api/v1/admin/backoffice/actions` ✅
- `GET /api/v1/admin/backoffice/actions/:id` ✅
- `GET /api/v1/admin/backoffice/export/ledger` ✅
- `POST /api/v1/admin/backoffice/actions/request` ✅
- `POST /api/v1/admin/backoffice/actions/confirm` ✅

## 🎯 Benefícios da Nova Organização

1. **Organização Lógica**: Endpoints agrupados por funcionalidade com emojis para fácil identificação
2. **Cobertura Completa**: Todos os 64 endpoints da aplicação estão incluídos
3. **Documentação Rica**: Cada endpoint tem descrição detalhada
4. **Testes Globais**: Scripts de teste automatizados para verificar saúde das respostas
5. **Variáveis Centralizadas**: Fácil configuração para diferentes ambientes
6. **Consistência**: Padronização de headers e estruturas

## 🔧 Manutenção

- Para adicionar novos endpoints, siga a estrutura organizada por funcionalidade
- Mantenha as variáveis centralizadas atualizadas
- Use as convenções de nomenclatura estabelecidas
- Documente novos endpoints adequadamente

## 📁 Arquivos de Referência

- **Collection Completa**: `Suno_Wallets_Complete_Collection.json`
- **Collections Antigas** (mantidas para referência): pasta `/docs/collections/postman/`
- **Documentação**: Este arquivo README

> **Nota**: As collections antigas foram mantidas na pasta para referência histórica, mas recomenda-se usar apenas a nova collection unificada.
