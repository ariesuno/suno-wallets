# 📐 Diretrizes de Desenvolvimento para o projeto `suno-wallets`

Estas diretrizes devem ser aplicadas em **todos os prompts e arquivos** gerados dentro do projeto `suno-wallets`, no ambiente Cursor.

---

## 🌍 Idioma e Convenções

- ✅ **Código sempre em inglês** (nomes de arquivos, variáveis, funções, pacotes, estruturas, etc.)
- 🗣️ **Comentários sempre em português do Brasil** (claro, explicativo e direto)
- 🧠 Não usar nomes genéricos (ex: `data`, `service`, `utils`) — seja sempre específico no domínio

---

## 🧱 Organização de Pastas (DDD – Domain-Driven Design)

Use uma estrutura de projeto baseada em DDD, organizada em camadas:

```
/src
  /Api               # Interfaces HTTP (controllers, routes, middlewares)
  /Application       # Regras de uso (use cases, services, DTOs)
  /Domain            # Regras de negócio (entidades, value objects, enums, interfaces)
  /Infrastructure    # Implementações (repositórios, jobs, integrações externas)
  /Shared            # Utilitários, logs, contexto, constantes, helpers
  /Tests             # Testes (unitários e de integração)
```

> ✅ Cada camada deve conter apenas suas responsabilidades.

---

## 🧪 Testes

- Todos os arquivos de teste devem ser criados na pasta `/Tests`, separados por tipo:
  - `/Tests/Unit/` para testes unitários
  - `/Tests/Integration/` para testes de integração
- Use **testify** como framework de teste
- Nome dos arquivos de teste devem seguir o padrão `nome_do_arquivo_test.go`

---

## 📄 Documentação

- Cada rota, use case ou job deve ser documentado com:
  - Descrição em comentário no início do arquivo (em português)
  - Assinatura e exemplo de uso da função
  - Observações relevantes sobre validações e limites
- Geração automática de documentação para APIs (Swagger), Postman e Bruno

---

## 📊 Observabilidade e Logs

- Todo log estruturado deve conter `tenant_id`, `user_id` (se aplicável), tipo de ação e status
- Logs devem ser criados via helper centralizado (`/Shared/Helpers/log.go`)
- Rastreabilidade obrigatória em jobs, ingestão, rotas e erros

---

## 🧹 Clean Code e Padronização

- Modularize o código em arquivos pequenos, específicos e reutilizáveis
- Nunca repetir código: crie helpers ou funções comuns quando necessário
- Tipagem obrigatória — evite tipos genéricos como `interface{}` sempre que possível
- Validação e tratamento de erros obrigatórios

---

## 🔐 Segurança e Auditoria

- Toda entidade sensível deve incluir campos de auditoria (`created_at`, `updated_at`, `deleted_at`, `created_by`, etc.)
- Uso obrigatório de `tenant_id` nas queries, filtros e logs
- Nunca retorne erros crus para o cliente — use erros tratados e mensagens claras

---

Essas diretrizes são **obrigatórias** e devem ser respeitadas mesmo que o prompt inicial não mencione explicitamente.

> Em caso de conflito entre o prompt e esta diretriz, esta diretriz prevalece.
