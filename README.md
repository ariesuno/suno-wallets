# 🏦 Suno Wallets

Sistema de carteiras digitais multi-tenant desenvolvido em Go, seguindo os princípios de Domain-Driven Design (DDD).

## 🚀 Características

- **Multi-tenant**: Suporte completo a múltiplos inquilinos
- **DDD**: Arquitetura baseada em Domain-Driven Design
- **Clean Architecture**: Separação clara entre camadas
- **Documentação**: Swagger/OpenAPI automático
- **Observabilidade**: Logs estruturados em JSON
- **Testes**: Framework testify com exemplos
- **Docker**: Containerização completa

## 🏗️ Arquitetura

```
src/
├── Api/              # Controllers, middlewares e rotas HTTP
├── Application/      # Casos de uso, serviços e DTOs
├── Domain/           # Entidades, value objects e interfaces
├── Infrastructure/   # Repositórios, integrações externas
├── Shared/          # Helpers, logs, utilitários
└── Tests/           # Testes unitários e de integração
```

## 🛠️ Stack Tecnológica

- **Linguagem**: Go 1.21+
- **Framework Web**: Gin
- **Banco de Dados**: PostgreSQL 15
- **Cache**: Redis 7
- **ORM**: GORM
- **Documentação**: Swagger/OpenAPI
- **Testes**: Testify
- **Containerização**: Docker & Docker Compose

## 📋 Pré-requisitos

- Go 1.21 ou superior
- Docker e Docker Compose
- PostgreSQL 15+ (ou usar via Docker)
- Redis 7+ (ou usar via Docker)

## 🚀 Instalação e Execução

### 1. Clone o repositório
```bash
git clone <repository-url>
cd suno-wallets
```

### 2. Configure as variáveis de ambiente
```bash
cp config.env.example .env
# Edite o arquivo .env conforme necessário
```

### 3. Execução com Docker (Recomendado)
```bash
# Subir todos os serviços
docker-compose up -d

# Ver logs da aplicação
docker-compose logs -f app

# Parar todos os serviços
docker-compose down
```

### 4. Execução local (desenvolvimento)
```bash
# Instalar dependências
go mod download

# Executar migrações (quando implementadas)
# go run cmd/migrate/main.go

# Executar aplicação
go run src/main.go
```

## 📚 Documentação da API

Após iniciar a aplicação, acesse:

- **Swagger UI**: http://localhost:8080/swagger/index.html
- **Health Check**: http://localhost:8080/health

## 🧪 Executando Testes

```bash
# Executar todos os testes
go test ./tests/...

# Executar testes com verbose
go test -v ./tests/...

# Executar testes de uma pasta específica
go test ./tests/unit/...

# Executar testes com coverage
go test -cover ./tests/...
```

## 🔧 Comandos Úteis

```bash
# Gerar documentação Swagger
swag init -g src/main.go -o docs

# Formatar código
go fmt ./...

# Executar linter
golangci-lint run

# Baixar dependências
go mod download

# Atualizar dependências
go mod tidy
```

## 🌐 Multi-tenancy

O sistema suporta multi-tenancy através do header `X-Tenant-ID`:

```bash
# Exemplo de requisição
curl -X GET "http://localhost:8080/api/v1/wallets" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440001" \
  -H "Content-Type: application/json"
```

## 📊 Estrutura do Banco de Dados

### Entidades Principais

- **Wallets**: Carteiras digitais dos usuários
- **Transactions**: Transações financeiras (a implementar)
- **Users**: Usuários do sistema (a implementar)

### Campos de Auditoria

Todas as entidades incluem:
- `id`: UUID único
- `tenant_id`: ID do inquilino (multi-tenancy)
- `created_at`: Data de criação
- `updated_at`: Data de atualização
- `deleted_at`: Data de exclusão (soft delete)
- `created_by`: Usuário que criou
- `updated_by`: Usuário que atualizou

## 🔐 Autenticação e Autorização

> 🚧 **Em desenvolvimento**: Sistema de autenticação JWT será implementado

## 📈 Observabilidade

### Logs Estruturados

```json
{
  "timestamp": "2023-12-01T15:30:00Z",
  "level": "info",
  "message": "Carteira criada com sucesso",
  "fields": {
    "wallet_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "550e8400-e29b-41d4-a716-446655440001",
    "user_id": "550e8400-e29b-41d4-a716-446655440002"
  }
}
```

### Monitoramento

- Health check endpoint: `/health`
- Métricas (a implementar): `/metrics`

## 🤝 Contribuição

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/nova-feature`)
3. Commit suas mudanças (`git commit -am 'Adiciona nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Abra um Pull Request

## 📝 Convenções

### Código
- **Inglês**: Nomes de arquivos, variáveis, funções
- **Português**: Comentários e documentação
- **Clean Code**: Funções pequenas, nomes descritivos
- **DDD**: Separação clara entre camadas

### Commits
```
tipo(escopo): descrição

feat(wallet): adiciona criação de carteiras
fix(auth): corrige validação de token
docs(readme): atualiza instruções de instalação
test(wallet): adiciona testes unitários
```

## 🐛 Troubleshooting

### Problemas Comuns

1. **Erro de conexão com banco**
   ```bash
   # Verificar se PostgreSQL está rodando
   docker-compose logs postgres
   ```

2. **Porta já em uso**
   ```bash
   # Alterar porta no .env ou parar processo
   lsof -ti:8080 | xargs kill -9
   ```

3. **Problemas com Go modules**
   ```bash
   go clean -modcache
   go mod download
   ```

## 📄 Licença

Este projeto está licenciado sob a Licença MIT - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 👥 Equipe

- **Backend**: Go, PostgreSQL, Redis
- **DevOps**: Docker, Docker Compose
- **Documentação**: Swagger/OpenAPI

---

> 💡 **Dica**: Para desenvolvimento, use o docker-compose com profile dev: `docker-compose --profile dev up`
