package infrastructure_test

import (
	"context"
	"testing"

	"suno-wallets/src/domain/entities"

	infrarepos "suno-wallets/src/infrastructure/repositories"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newInMemoryDB cria um DB SQLite em memória somente para testes unitários
func newInMemoryDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Ajustes para SQLite: remover tipos específicos não suportados
	require.NoError(t, db.Migrator().AutoMigrate(&entities.Wallet{}))
	return db
}

func TestWalletRepository_CreateAndGet(t *testing.T) {
	db := newInMemoryDB(t)
	repo := infrarepos.NewWalletRepository(db)
	_ = repo // Verificar implementação da interface

	ctx := context.Background()
	tenantID := uuid.New()
	ownerID := uuid.New()
	creator := uuid.New()

	wallet := &entities.Wallet{
		TenantID:  tenantID,
		Name:      "Test Wallet",
		Type:      entities.WalletTypePersonal,
		Status:    entities.WalletStatusActive,
		Currency:  "BRL",
		OwnerID:   ownerID,
		OwnerType: "user",
		CreatedBy: creator,
	}

	require.NoError(t, repo.Create(ctx, wallet))

	got, err := repo.GetByID(ctx, wallet.ID, tenantID)
	require.NoError(t, err)
	require.Equal(t, wallet.ID, got.ID)
}
