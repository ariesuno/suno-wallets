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

### 🔑 Variáveis de ambiente (.env) e precedência

- Local (go run): usamos `godotenv.Overload()`, então variáveis do `.env` têm precedência sobre variáveis já exportadas no ambiente.
- Docker Compose: usamos `env_file: .env` no serviço `app`, então o container recebe as variáveis do `.env` automaticamente.
- Certificados B3: no container os arquivos são montados em `/app/certs` (volume read-only). Ajuste os caminhos no `.env` quando rodar em container, por exemplo:
  - `B3_CERT_P12_PATH=/app/certs/b3_certificatep12filepath.p12`
  - `B3_LEGACY_CERT_PATH=/app/certs/b3_certificatefilepath.cer`
  - Em execução local direta (sem Docker), use `certs/...`.

## 📚 Documentação da API

Após iniciar a aplicação, acesse:

- **Swagger UI**: http://localhost:8080/swagger/index.html
- **Health Check**: http://localhost:8080/health
  - **Metrics (Prometheus)**: http://localhost:8080/metrics
  - **Grafana**: http://localhost:3000
  - **Prometheus**: http://localhost:9090

### Collections (Postman e Bruno)

- **Postman**: arquivo `postman/B3 Preview.postman_collection.json`
- **Bruno**: arquivos `.bru` em `bruno/b3/`
  - `bruno/b3/transactions_preview.bru`
  - `bruno/b3/positions_preview.bru`

Exemplo (Bruno CLI):

```bash
bru run bruno/b3/transactions_preview.bru
bru run bruno/b3/positions_preview.bru
```

## 🧪 Executando Testes

```bash
# Executar todos os testes
go test ./tests/...

# Executar testes com verbose
go test -v ./tests/...

# Executar testes de uma pasta específica
go test ./tests/unit/...

# Executar testes de infraestrutura e observabilidade
go test ./tests/infrastructure/...
go test ./tests/integration/observability/...

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
- Métricas: `/metrics`

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

4. **Prometheus/Grafana**
   - Stack via docker-compose (Prometheus, Grafana, Loki, Promtail, node-exporter, cadvisor)
   - Métricas em `/metrics`
   - Dashboards via Grafana (datasource Prometheus)

## 🏦 B3 Official Client
- Certificados esperados em `certs/`:
  - `b3_certificate12filepath.p12`
  - `b3_certificatefilepath.cer` (opcional)
- Variáveis:
  - `B3_URL_DATA`, `B3_OAUTH_TOKEN_URL`, `B3_CLIENT_ID`, `B3_CLIENT_SECRET`, `B3_CERT_P12_PATH`, `B3_CERT_PASSPHRASE`
- Exemplo:
  ```go
  cfg := &b3cfg.B3Config{URLData: os.Getenv("B3_URL_DATA"), CertP12Path: os.Getenv("B3_CERT_P12_PATH"), CertPassphrase: os.Getenv("B3_CERT_PASSPHRASE"), Timeout: 5*time.Second}
  cli, _ := b3client.NewB3OfficialClient(cfg, getBearer)
  resp, err := cli.MakeRequest(ctx, http.MethodGet, "/position/v3/...", q, cpf, true)
  _ = resp; _ = err
  ```

### cURL (exemplos)
- Posições v3 (exige OAuth e CPF):
```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "X-Investor-CPF: 12345678901" \
     -H "Content-Type: application/json" \
     "$B3_URL_DATA/position/v3/equities/investors/12345678901?referenceStartDate=2024-01-01&referenceEndDate=2024-01-31&page=1"
```

- Transações v2:
```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "X-Investor-CPF: 12345678901" \
     -H "Content-Type: application/json" \
     "$B3_URL_DATA/assets-trading/v2/equity/12345678901?referenceStartDate=2024-01-01&referenceEndDate=2024-01-31&page=1"
```

### Injeção de Tokens PKCE
- O cliente aceita função `getBearer(ctx)` para injetar tokens obtidos via consentimento externo (PKCE/ALF).

### Health
- Endpoint opcional: `GET /api/v1/b3/health/auth` (não valida contra B3; apenas verifica presença de configuração)

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
