Fase 1 – Ingestão B3 (raw + normalização + observabilidade)
Fase 2 – Consolidação de Carteira (cálculo de posição, PM, etc.)
Fase 3 – Rentabilidade e Cotações (gráficos, valor de carteira)
Fase 4 – Transações Manuais + Ativos Externos (XP, BTG, etc.)
Fase 5 – Multi-Carteiras + Multi-Tenant
Fase 6 – Auditoria + Segurança + Performance
Fase 7 – Integração com Produtos + SDKs
Fase 8 – IA e Recomendações (baseado na carteira)


🧩 FASE 1 – Integração com a B3 (via ALF)
Objetivo: Criar a estrutura responsável por gerenciar as autorizações dos clientes com a B3, armazenar e consolidar os dados recebidos, com observabilidade, documentação, testes, validações e modularização.

✅ BLOCO 1 — Estrutura inicial e controle de autorização
Criação da estrutura multi-tenant com controle de TenantId

Criar todos os middlewares, modelos e helpers necessários para garantir que toda requisição e job respeitem o contexto do tenant.

Criação da tabela b3_authorizations

Estados possíveis: PENDING, ACTIVE, PENDING_DISCONNECT, DISCONNECTED, DELETED.

Registrar datas de criação, ativação, última sincronização e revogação.

Endpoints de gestão de autorização B3

Iniciar autorização

Verificar status de autorização (por CPF ou ID)

Revogar autorização

Listar todas as autorizações ativas/inativas do tenant

Middleware de validação de autorização ativa

Para uso interno em rotas de ingestão/sincronização

Retornar erro específico se a autorização estiver revogada ou expirada

Documentação Swagger + Postman + Bruno

Categorização dos endpoints como Auth → B3

Nomear claramente as ações: GET /b3/authorization-status, POST /b3/initiate-authorization, etc.

✅ BLOCO 2 — Ingestão de dados da B3
Criação da estrutura de raw_data_b3

Armazenar os arquivos brutos recebidos (ex: JSON de operações, notas de corretagem)

Metadados obrigatórios: tipo do arquivo, data de referência, hash do conteúdo, origem, status de ingestão

Serviço de ingestão por webhook

Criar endpoint POST /b3/ingest-raw que recebe dados da B3

Validar assinatura, tenant, autorização ativa

Salvar no bucket/local e registrar no banco

Job de processamento inicial do raw data

Identifica o tipo (movimentações, posições, etc)

Salva em tabelas específicas (ex: b3_trades, b3_positions)

Mantém versão original disponível para reprocessamento manual

Script de ingestão manual (por CLI/API)

Permitir que admin ou sistema faça reprocessamento de arquivos antigos

Passar ID do raw data ou intervalo de datas

✅ BLOCO 3 — Observabilidade e saúde do sistema
Endpoint de healthcheck por CPF

GET /b3/healthcheck?cpf=XXX

Retorna status da autorização e última data de sincronização

Dashboard de status por tenant

Endpoint GET /b3/status-summary

Retorna:

Nº de autorizações ativas

Última sincronização média

Quantidade de erros nas últimas 24h

Erros por tipo de arquivo

Logs, métricas e alertas

Logar eventos críticos: falhas de ingestão, revogação de autorização, erros de assinatura

Integrar com observabilidade padrão (Grafana/Prometheus/Loki)

Contar erros por tenant, endpoint, tipo de dado

✅ BLOCO 4 — Segurança, testes e auditoria
Validação de payloads com schemas

Validar entrada e estrutura dos dados vindos da B3

Geração de erros amigáveis (em português nos logs)

Auditoria de alterações

Logar quem iniciou/revogou autorizações

Registrar todos os acessos aos dados via endpoints internos

Cobertura de testes

Unitários com testify

Integração de endpoints com mocks

Validar comportamento esperado para cada status da autorização

✅ BLOCO 5 — Interfaces administrativas e manuais
Endpoint para listar arquivos raw por CPF ou data

Com filtros por tipo, status, intervalo de datas

Endpoint para reprocessamento forçado

Apenas para usuários com perfil admin no tenant

Documentado como rota de uso interno

Job manual para remoção de dados de clientes inativos

Criar CLI purge_client_data que só é executado sob comando explícito

Permitir passar argumentos como --cpf, --inactive_since=YYYY-MM-DD

