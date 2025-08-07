Infrastructure Layer

Descrição
- Camada responsável por persistência (PostgreSQL/Redis), integrações externas, jobs e observabilidade.
- Implementa contratos definidos em domain/interfaces.

Estrutura
- repositories/: Implementações concretas dos repositórios do domínio.
- data/: Configuração de banco e Redis (pool, timeouts, ping).
- externalapis/: Clientes HTTP com timeouts e interfaces para mocks.
- backgroundjobs/: Workers/clients Asynq para execução assíncrona.
- observability/: Métricas/Tracing/Logs. Inclui endpoint /metrics (Prometheus).

Padrões
- Go em inglês, comentários em pt-BR.
- Todos acessos externos com timeouts.
- Nenhuma regra de negócio aqui.

Testes
- Testes de infraestrutura ficam em tests/infrastructure/...

