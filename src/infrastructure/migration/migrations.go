package migration

import (
	"fmt"

	"gorm.io/gorm"

	"suno-wallets/src/domain/entities"
	"suno-wallets/src/shared/helpers"
)

// AutoMigrate executa as migrações automáticas do GORM
func AutoMigrate(db *gorm.DB) error {
	helpers.LogInfo("Iniciando migrações automáticas", map[string]interface{}{})

	// Lista de entidades para migração
	models := []interface{}{
		&entities.Wallet{},
		// Adicionar novas entidades aqui conforme necessário
	}

	// Executar migrações
	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			helpers.LogError("Falha na migração", err, map[string]interface{}{
				"model": fmt.Sprintf("%T", model),
			})
			return err
		}
	}

	helpers.LogInfo("Migrações automáticas concluídas com sucesso", map[string]interface{}{
		"models_count": len(models),
	})

	return nil
}

// CreateExtensions cria extensões necessárias no PostgreSQL
func CreateExtensions(db *gorm.DB) error {
	helpers.LogInfo("Criando extensões PostgreSQL", map[string]interface{}{})

	extensions := []string{
		"CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";",
		"CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";",
	}

	for _, ext := range extensions {
		if err := db.Exec(ext).Error; err != nil {
			helpers.LogError("Falha ao criar extensão", err, map[string]interface{}{
				"extension": ext,
			})
			return err
		}
	}

	helpers.LogInfo("Extensões PostgreSQL criadas com sucesso", map[string]interface{}{})
	return nil
}

// CreateIndexes cria índices adicionais necessários
func CreateIndexes(db *gorm.DB) error {
	helpers.LogInfo("Criando índices adicionais", map[string]interface{}{})

	indexes := []string{
		// Índices para performance de consultas multi-tenant
		"CREATE INDEX IF NOT EXISTS idx_wallets_tenant_owner ON wallets(tenant_id, owner_id);",
		"CREATE INDEX IF NOT EXISTS idx_wallets_tenant_status ON wallets(tenant_id, status);",
		"CREATE INDEX IF NOT EXISTS idx_wallets_tenant_type ON wallets(tenant_id, type);",
		"CREATE INDEX IF NOT EXISTS idx_wallets_owner_default ON wallets(owner_id, is_default) WHERE is_default = true;",

		// Índices para campos de auditoria
		"CREATE INDEX IF NOT EXISTS idx_wallets_created_at ON wallets(created_at);",
		"CREATE INDEX IF NOT EXISTS idx_wallets_updated_at ON wallets(updated_at);",
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			helpers.LogError("Falha ao criar índice", err, map[string]interface{}{
				"index": idx,
			})
			return err
		}
	}

	helpers.LogInfo("Índices adicionais criados com sucesso", map[string]interface{}{})
	return nil
}
