# 📂 Collections - Postman & Bruno

Esta pasta contém coleções de APIs organizadas para facilitar testes e integração com o sistema Suno Wallets.

## 🗂️ Estrutura

### **📝 Bruno Collections** (`/bruno/`)
Coleções organizadas por domínio, seguindo a estrutura DDD do projeto:

#### **🏥 Observability** (`/observability/`)
- `health_check.bru` - Health check geral
- `health_details.bru` - Detalhes de saúde do sistema
- `liveness.bru` - Verificação de liveness
- `readiness.bru` - Verificação de readiness
- `metrics.bru` - Métricas Prometheus

#### **🏦 B3 Integration** (`/b3/`)
- `complete_sync_ingestion.bru` - **🆕 Sincronização completa inteligente**
- `complete_sync_status_analysis.bru` - **🆕 Análise de status de sync**
- `complete_sync_dry_run.bru` - **🆕 Simulação de sync completo**
- `test_connection.bru` - Teste de conectividade B3
- `sync_run.bru` - Sincronização incremental
- `client_status.bru` - Status do cliente
- `client_last_sync.bru` - Última sincronização
- `raw_ingest.bru` - Ingestão de dados raw
- `normalize_run.bru` - Normalização de dados
- `reset_and_refetch.bru` - Reset e re-ingestão
- `transactions_preview.bru` - Preview de transações
- `positions_preview.bru` - Preview de posições
- `raw_date_range.bru` - Range de datas dos dados
- `summary.bru` - Resumo de dados
- `tickers.bru` - Lista de tickers
- `sync_window.bru` - Janela de sincronização

#### **🔧 B3 Admin** (`/b3_admin/`)
- `reactivation_analyze.bru` - Análise de reativação
- `reactivation_execute.bru` - Execução de reativação

#### **👤 B3 Client** (`/b3_client/`)
- `reactivation_status.bru` - Status de reativação

#### **🛡️ Admin** (`/admin/`)
- `profile.bru` - Perfil de cliente

#### **⚙️ Client Policy** (`/clientpolicy/`)
- `get_policy.bru` - Obter política do cliente
- `set_policy.bru` - Definir política do cliente
- `audit.bru` - Auditoria de políticas
- `dry_run.bru` - Teste de política

#### **🔄 Operations** (`/ops/`)
- `create_manual_op.bru` - Criar operação manual
- `list_manual_ops.bru` - Listar operações manuais
- `list_candidates.bru` - Candidatos de deduplicação
- `scan_dedup.bru` - Scan de deduplicação
- `timeline.bru` - Timeline de operações
- `timeline_export.bru` - Export de timeline
- `reconciliation_summary.bru` - Resumo de reconciliação

#### **🔍 Reconciliation** (`/reconciliation/`)
- `scan.bru` - Scan de reconciliação
- `list_inconsistencies.bru` - Listar inconsistências
- `get_inconsistency.bru` - Obter inconsistência específica
- `auto_fix.bru` - Auto-correção

#### **💰 Wallets** (`/wallets/`)
- `create_wallet.bru` - Criar carteira
- `list_wallets.bru` - Listar carteiras
- `get_wallet.bru` - Obter carteira
- `get_by_owner.bru` - Carteiras por proprietário
- `update_wallet.bru` - Atualizar carteira
- `delete_wallet.bru` - Deletar carteira

### **📮 Postman Collections** (`/postman/`)
Coleções Postman com exemplos e documentação detalhada:

- `postman_collection.json` - **🆕 Coleção principal atualizada com Complete Sync**
- `postman_b3_complete_sync.json` - **🆕 Coleção específica para Complete Sync**
- `postman_b3_data.json` - Dados B3
- `postman_b3_admin.json` - Administração B3
- `postman_backoffice_admin.json` - Backoffice administrativo
- `postman_client_policy.json` - Políticas de cliente
- `postman_manual_ops.json` - Operações manuais
- `postman_reconciliation.json` - Reconciliação
- `postman_timeline_summary.json` - Timeline e resumos

## 🚀 **Novidades - Complete Sync Orchestrator**

### **🧠 Endpoints Inteligentes Adicionados:**

#### **1. Complete Sync - Ingestion**
```http
POST /api/v1/b3/sync/complete-ingestion
```
**Funcionalidade:** Executa fluxo completo e inteligente de sincronização B3
- **Analisa automaticamente** se o cliente é novo ou existente
- **Escolhe estratégia ideal:** histórico completo vs incremental vs híbrido
- **Executa pipeline completo:** ingestão + normalização + reconciliação

#### **2. Complete Sync - Status Analysis**
```http
GET /api/v1/b3/sync/status-analysis?cpf=12345678901
```
**Funcionalidade:** Analisa status sem executar ações
- **Retorna estratégia recomendada** e estimativa de tempo
- **Verifica gaps** de sincronização
- **Útil para planejamento** e validação prévia

#### **3. Complete Sync - Dry Run**
```http
POST /api/v1/b3/sync/complete-ingestion (com dryRun: true)
```
**Funcionalidade:** Simula execução sem modificações
- **Valida estratégia** antes da execução real
- **Estima tempo** de processamento
- **Zero impacto** no sistema

## 🎯 **Estratégias Inteligentes**

### **🆕 Cliente Novo (FULL_HISTORICAL)**
- **Situação:** Primeiro acesso ao sistema
- **Ação:** Ingestão histórica completa desde 2019-11-01
- **Tempo:** 2-4 horas

### **✅ Cliente Atual (INCREMENTAL_ONLY)**
- **Situação:** Gap ≤ 30 dias
- **Ação:** Sincronização incremental desde última data
- **Tempo:** 1-5 minutos

### **🔄 Cliente Desatualizado (HYBRID_OPTIMIZED)**
- **Situação:** Gap > 30 dias
- **Ação:** Reativação inteligente (híbrida)
- **Tempo:** 5-30 minutos

## 🔧 **Como Usar**

### **Bruno:**
1. Abrir Bruno IDE
2. Importar pasta `/docs/collections/bruno/`
3. Configurar environment (Local/Production)
4. Executar requests

### **Postman:**
1. Importar arquivo JSON de interesse
2. Configurar variáveis de ambiente:
   - `baseUrl`: http://localhost:8080
   - `tenantId`: status_invest
   - `testCpf`: 12345678901
3. Executar coleção

## 📋 **Variáveis de Ambiente**

### **Obrigatórias:**
- `baseUrl` - URL base da API
- `tenantId` - ID do tenant (ex: status_invest)
- `testCpf` - CPF para testes (11 dígitos)

### **Opcionais:**
- `adminSecret` - Chave para endpoints administrativos
- `authToken` - Token de autenticação (se aplicável)

## 🎉 **Benefícios do Complete Sync**

- **🧠 Inteligência:** Decisão automática de estratégia
- **⚡ Performance:** Evita reprocessamento desnecessário
- **🔄 Completude:** Pipeline end-to-end em uma chamada
- **🛡️ Confiabilidade:** Dry-run, error handling, rollback
- **📈 Observabilidade:** Métricas e logs detalhados

---

> **💡 Dica:** Use sempre o **Complete Sync - Status Analysis** primeiro para entender a estratégia recomendada, depois execute o **Dry Run** para validar antes da execução real!
