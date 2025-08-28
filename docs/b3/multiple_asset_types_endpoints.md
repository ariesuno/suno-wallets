# 📊 Suporte Completo a Múltiplos Tipos de Ativo B3

## 🎯 Visão Geral

Este documento descreve a implementação completa do suporte a todos os tipos de ativos disponíveis na API B3, expandindo além do suporte inicial apenas a **equities**.

## 🔗 Endpoints B3 Suportados

### 1. **Equities (Ações)**
```
GET /equities/investors/{documentNumber}
GET /position/v3/equities/investors/{documentNumber}  # Endpoint especial para posições
```

### 2. **Fixed Income (Renda Fixa)**
```
GET /fixed-income/investors/{documentNumber}
```

### 3. **Treasury Bonds (Títulos do Tesouro)**
```
GET /treasury-bonds/investors/{documentNumber}
```

### 4. **Derivatives (Derivativos)**
```
GET /derivatives/investors/{documentNumber}
```

### 5. **Securities Lending (Empréstimos de Valores Mobiliários)**
```
GET /securities-lending/investors/{documentNumber}
```

## 🏗️ Arquitetura Implementada

### Domain Layer
- **`enums.B3AssetType`**: Enum específico para tipos de ativo B3
- **Mapeamento automático**: Conversão entre tipos genéricos e B3-específicos
- **Validações**: Suporte a transações/posições por tipo

### Infrastructure Layer
- **Cliente B3 expandido**: Métodos para todos os tipos de ativo
- **Endpoints dinâmicos**: Roteamento automático baseado no tipo
- **Compatibilidade**: Mantém funcionamento com código existente

### Application Layer
- **Processo de ingestão**: Usa endpoints específicos por tipo
- **Sync assíncrono**: Processa todos os tipos automaticamente
- **Orchestração**: Suporte completo no fluxo E2E

## 🔧 Uso da API

### Processo Async/Sync (Recomendado)

Por padrão, o processo async agora inclui **todos** os tipos de ativo:

```bash
curl -X POST "http://localhost:8080/api/v1/b3/async/sync/complete-ingestion" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: status_invest" \
  -d '{
    "cpf": "12345678901",
    "assetTypes": ["equities", "fixed-income", "treasury-bonds", "derivatives"],
    "dataTypes": ["transactions", "positions"],
    "includeReconciliation": true,
    "dryRun": false
  }'
```

### Processo Sync Específico

Para processar apenas um tipo específico:

```bash
# Apenas renda fixa
curl -X POST "http://localhost:8080/api/v1/b3/async/sync/complete-ingestion" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: status_invest" \
  -d '{
    "cpf": "12345678901",
    "assetTypes": ["fixed-income"],
    "dataTypes": ["transactions", "positions"]
  }'
```

### Preview/Test de Endpoints

Para testar conectividade com tipos específicos:

```bash
# Teste de ações (mantém compatibilidade)
curl "http://localhost:8080/api/v1/b3/fetch/transactions/preview?cpf=12345678901&assetType=equities"

# Teste de renda fixa
curl "http://localhost:8080/api/v1/b3/fetch/transactions/preview?cpf=12345678901&assetType=fixed-income"
```

## 📋 Tipos de Ativo Suportados

| Tipo | String API | Descrição | Transações | Posições |
|------|------------|-----------|:----------:|:--------:|
| `equities` | `"equities"` | Ações | ✅ | ✅ |
| `fixed-income` | `"fixed-income"` | Renda Fixa | ✅ | ✅ |
| `treasury-bonds` | `"treasury-bonds"` | Títulos do Tesouro | ✅ | ✅ |
| `derivatives` | `"derivatives"` | Derivativos | ✅ | ✅ |
| `securities-lending` | `"securities-lending"` | Empréstimos | ✅ | ✅ |

## 🔄 Migração e Compatibilidade

### ✅ Compatibilidade Mantida
- Código existente usando `"equity"` continua funcionando
- Endpoints v2/v3 existentes preservados
- Processo async/sync backwards compatible

### 🆕 Novos Defaults
- **Antes**: Apenas `["equity"]`
- **Agora**: `["equities", "fixed-income", "treasury-bonds", "derivatives"]`
- **Configurável**: Pode especificar tipos específicos conforme necessário

### 📈 Benefícios
1. **Cobertura completa**: Todos os dados disponíveis na B3
2. **Performance**: Processamento paralelo por tipo
3. **Flexibilidade**: Escolha de tipos específicos conforme necessidade
4. **Escalabilidade**: Fácil adição de novos tipos no futuro

## 🧪 Testes

### Testes Unitários
```bash
go test ./src/tests/unit/b3/endpoints/...
```

### Testes de Integração
```bash
# Configurar variáveis de ambiente
export RUN_INTEGRATION_TESTS=true
export B3_TEST_CPF=12345678901
export B3_URL_DATA=...
export B3_CLIENT_ID=...

go test ./src/tests/integration/b3/...
```

## 🔍 Monitoramento

### Métricas por Tipo de Ativo
- `b3_requests_total{asset_type="equities"}`
- `b3_requests_total{asset_type="fixed-income"}`
- `b3_requests_total{asset_type="treasury-bonds"}`
- etc.

### Logs Estruturados
```json
{
  "level": "info",
  "message": "B3 request completed",
  "asset_type": "fixed-income",
  "endpoint": "/fixed-income/investors/12345678901",
  "status_code": 200,
  "duration_ms": 1250
}
```

## 🚀 Próximos Passos

1. **Monitoramento específico** por tipo de ativo
2. **Otimizações** baseadas em padrões de uso
3. **Cache inteligente** por tipo e período
4. **Relatórios detalhados** por categoria de ativo

## 📚 Referências

- [Documentação API B3](https://developers.b3.com.br/)
- [Enum B3AssetType](../src/domain/enums/b3_asset_type.go)
- [Cliente B3 Endpoints](../src/infrastructure/b3/client/endpoints.go)
- [Testes de Integração](../src/tests/integration/b3/)

---

**Status**: ✅ Implementado e testado  
**Versão**: v1.0  
**Data**: 2024-01-15
