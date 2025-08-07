Domain Layer

Descrição
- Esta camada concentra as regras de negócio do sistema (DDD).
- É independente de infraestrutura. Não deve importar pacotes de camadas externas.

Estrutura
- entities/: Entidades do domínio com estado e comportamento.
- valueobjects/: Objetos de valor imutáveis com validação.
- enums/: Conjuntos fixos de valores.
- interfaces/: Contratos de repositórios e serviços.
- repositories/: Interfaces específicas (ex.: wallet_repository.go). Opcional mover para interfaces/ se preferir uma única pasta de contratos.

Diretrizes
- Código em inglês; comentários em pt-BR.
- Toda entidade deve ter validação, campos de auditoria e TenantID.
- Value Objects devem ser imutáveis, com construtores validadores (ex.: Money, Email, CPF).
- Não usar lógica de infraestrutura (DB, HTTP) nesta camada.

Testes
- Testes unitários ficam em tests/unit/domain/.
- Nomes dos testes em inglês.

Como estender
- Criar novas entidades em entities/ com métodos de negócio e Validate().
- Criar value objects em valueobjects/ com validações rigorosas e APIs imutáveis.
- Adicionar enums em enums/ para estados e tipos fixos.
- Definir contratos em interfaces/; implementar na camada infrastructure/.

