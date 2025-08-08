package infrastructure_test

import (
	"testing"

	"suno-wallets/src/infrastructure/database"
	"suno-wallets/src/infrastructure/migration"
	"suno-wallets/src/shared/config"

	"github.com/stretchr/testify/require"
)

// TestPostgresConnection executa conexão e migração básicas
func TestPostgresConnection(t *testing.T) {
	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "suno_user",
		DBPassword: "suno_password",
		DBName:     "suno_wallets",
		DBSSLMode:  "disable",
		LogLevel:   "error",
	}

	db, err := database.Connect(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB, _ := db.DB(); _ = sqlDB.Close() })

	// Verificar conexão
	var one int
	require.NoError(t, db.Raw("SELECT 1").Scan(&one).Error)
	require.Equal(t, 1, one)
	// Criar extensões
	require.NoError(t, migration.CreateExtensions(db))
}
