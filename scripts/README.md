# Scripts Utilitários

Esta pasta contém scripts auxiliares para desenvolvimento e manutenção do projeto.

## 📁 Estrutura

```
scripts/
├── db/                     # Scripts de banco de dados
│   ├── truncate_all_tables.sql    # Limpa todas as tabelas para testes
│   └── check_tables_status.sql    # Verifica status das tabelas
└── README.md              # Este arquivo
```

## 🗄️ Scripts de Banco (`db/`)

### `truncate_all_tables.sql`
**Finalidade**: Limpa todas as tabelas do banco para permitir testes do zero.

**⚠️ CUIDADO**: Apaga TODOS os dados! Use apenas em desenvolvimento.

**Como usar**:
```bash
# Executar truncate completo
docker exec -i suno-wallets-postgres-1 psql -U suno_user -d suno_wallets -f /dev/stdin < scripts/db/truncate_all_tables.sql
```

**Características**:
- ✅ Ordem correta de dependências (evita erros de FK)
- ✅ Desabilita temporariamente verificações de FK
- ✅ Reinicia sequences (RESTART IDENTITY)
- ✅ Verificação automática pós-execução
- ✅ Relatório detalhado de sucesso/falha

### `check_tables_status.sql`
**Finalidade**: Verifica rapidamente o status de todas as tabelas.

**Como usar**:
```bash
# Verificar contagem de registros
docker exec -i suno-wallets-postgres-1 psql -U suno_user -d suno_wallets -f /dev/stdin < scripts/db/check_tables_status.sql
```

**Características**:
- 📊 Mostra contagem por categoria de tabelas
- 🎯 Resumo geral do banco
- 📋 Organizado por função (B3, Reconciliação, Archive, etc.)

## 🚀 Fluxo de Uso Comum

```bash
# 1. Verificar estado atual
docker exec -i suno-wallets-postgres-1 psql -U suno_user -d suno_wallets -f /dev/stdin < scripts/db/check_tables_status.sql

# 2. Limpar para teste
docker exec -i suno-wallets-postgres-1 psql -U suno_user -d suno_wallets -f /dev/stdin < scripts/db/truncate_all_tables.sql

# 3. Executar seus testes de coleta B3...

# 4. Verificar resultados
docker exec -i suno-wallets-postgres-1 psql -U suno_user -d suno_wallets -f /dev/stdin < scripts/db/check_tables_status.sql
```

## 📝 Convenções

- **Pasta `db/`**: Scripts relacionados ao banco de dados
- **Nomenclatura**: `verbo_objeto.sql` (ex: `truncate_all_tables.sql`)
- **Documentação**: Comentários detalhados dentro dos scripts
- **Segurança**: Sempre incluir verificações de ambiente quando apropriado

## 🔮 Scripts Futuros

Possíveis adições para esta pasta:
- `scripts/db/seed_test_data.sql` - Dados de teste padrão
- `scripts/db/backup_development.sh` - Backup do ambiente dev
- `scripts/deploy/` - Scripts de deploy
- `scripts/monitoring/` - Scripts de monitoramento
