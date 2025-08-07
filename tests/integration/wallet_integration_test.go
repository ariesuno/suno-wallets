package integration_test

import (
	"context"
	"testing"

	"suno-wallets/src/domain/entities"
	"suno-wallets/src/infrastructure/database"
	"suno-wallets/src/infrastructure/migration"
	"suno-wallets/src/infrastructure/repositories"
	"suno-wallets/src/shared/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// WalletIntegrationTestSuite suite para testes de integração de carteiras
type WalletIntegrationTestSuite struct {
	suite.Suite
	db         *gorm.DB
	repository repositories.WalletRepository
	ctx        context.Context
	tenantID   uuid.UUID
	ownerID    uuid.UUID
	createdBy  uuid.UUID
}

// SetupSuite configura o ambiente de teste
func (suite *WalletIntegrationTestSuite) SetupSuite() {
	// Configurar banco de dados de teste
	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "suno_user",
		DBPassword: "suno_password",
		DBName:     "suno_wallets_test",
		DBSSLMode:  "disable",
		LogLevel:   "error",
	}

	db, err := database.Connect(cfg)
	require.NoError(suite.T(), err)

	// Executar migrações
	err = migration.AutoMigrate(db)
	require.NoError(suite.T(), err)

	suite.db = db
	suite.repository = repositories.NewWalletRepository(db)
	suite.ctx = context.Background()
	suite.tenantID = uuid.New()
	suite.ownerID = uuid.New()
	suite.createdBy = uuid.New()
}

// TearDownSuite limpa o ambiente de teste
func (suite *WalletIntegrationTestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

// SetupTest limpa dados antes de cada teste
func (suite *WalletIntegrationTestSuite) SetupTest() {
	// Limpar tabela de carteiras para cada teste
	suite.db.Exec("DELETE FROM wallets WHERE tenant_id = ?", suite.tenantID)
}

// TestCreateWallet testa criação de carteira
func (suite *WalletIntegrationTestSuite) TestCreateWallet() {
	wallet := &entities.Wallet{
		TenantID:    suite.tenantID,
		Name:        "Carteira de Teste",
		Description: "Descrição da carteira de teste",
		Type:        entities.WalletTypePersonal,
		Currency:    "BRL",
		OwnerID:     suite.ownerID,
		OwnerType:   "user",
		CreatedBy:   suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, wallet)
	require.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), uuid.Nil, wallet.ID)
	assert.Equal(suite.T(), entities.WalletStatusActive, wallet.Status)
}

// TestCreateDuplicateDefaultWallet testa erro ao criar carteira padrão duplicada
func (suite *WalletIntegrationTestSuite) TestCreateDuplicateDefaultWallet() {
	// Criar primeira carteira padrão
	wallet1 := &entities.Wallet{
		TenantID:  suite.tenantID,
		Name:      "Carteira Padrão 1",
		Type:      entities.WalletTypePersonal,
		Currency:  "BRL",
		OwnerID:   suite.ownerID,
		OwnerType: "user",
		IsDefault: true,
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, wallet1)
	require.NoError(suite.T(), err)

	// Tentar criar segunda carteira padrão (deve falhar)
	wallet2 := &entities.Wallet{
		TenantID:  suite.tenantID,
		Name:      "Carteira Padrão 2",
		Type:      entities.WalletTypePersonal,
		Currency:  "BRL",
		OwnerID:   suite.ownerID,
		OwnerType: "user",
		IsDefault: true,
		CreatedBy: suite.createdBy,
	}

	err = suite.repository.Create(suite.ctx, wallet2)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), entities.ErrDuplicateDefaultWallet, err)
}

// TestGetWalletByID testa busca de carteira por ID
func (suite *WalletIntegrationTestSuite) TestGetWalletByID() {
	// Criar carteira
	wallet := &entities.Wallet{
		TenantID:  suite.tenantID,
		Name:      "Carteira para Busca",
		Type:      entities.WalletTypePersonal,
		Currency:  "BRL",
		OwnerID:   suite.ownerID,
		OwnerType: "user",
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, wallet)
	require.NoError(suite.T(), err)

	// Buscar carteira
	found, err := suite.repository.GetByID(suite.ctx, wallet.ID, suite.tenantID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), wallet.ID, found.ID)
	assert.Equal(suite.T(), wallet.Name, found.Name)
}

// TestGetWalletByIDNotFound testa busca de carteira inexistente
func (suite *WalletIntegrationTestSuite) TestGetWalletByIDNotFound() {
	nonExistentID := uuid.New()

	_, err := suite.repository.GetByID(suite.ctx, nonExistentID, suite.tenantID)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsNotFoundError(err))
}

// TestGetWalletsByOwner testa busca de carteiras por proprietário
func (suite *WalletIntegrationTestSuite) TestGetWalletsByOwner() {
	// Criar múltiplas carteiras
	wallets := []*entities.Wallet{
		{
			TenantID:  suite.tenantID,
			Name:      "Carteira 1",
			Type:      entities.WalletTypePersonal,
			Currency:  "BRL",
			OwnerID:   suite.ownerID,
			OwnerType: "user",
			IsDefault: true,
			CreatedBy: suite.createdBy,
		},
		{
			TenantID:  suite.tenantID,
			Name:      "Carteira 2",
			Type:      entities.WalletTypeBusiness,
			Currency:  "USD",
			OwnerID:   suite.ownerID,
			OwnerType: "user",
			CreatedBy: suite.createdBy,
		},
	}

	for _, wallet := range wallets {
		err := suite.repository.Create(suite.ctx, wallet)
		require.NoError(suite.T(), err)
	}

	// Buscar carteiras do proprietário
	found, err := suite.repository.GetByOwner(suite.ctx, suite.ownerID, suite.tenantID)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), found, 2)

	// Verificar se a carteira padrão vem primeiro
	assert.True(suite.T(), found[0].IsDefault)
}

// TestUpdateWallet testa atualização de carteira
func (suite *WalletIntegrationTestSuite) TestUpdateWallet() {
	// Criar carteira
	wallet := &entities.Wallet{
		TenantID:  suite.tenantID,
		Name:      "Carteira Original",
		Type:      entities.WalletTypePersonal,
		Currency:  "BRL",
		OwnerID:   suite.ownerID,
		OwnerType: "user",
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, wallet)
	require.NoError(suite.T(), err)

	// Atualizar carteira
	wallet.Name = "Carteira Atualizada"
	wallet.Description = "Nova descrição"
	updatedBy := uuid.New()
	wallet.UpdatedBy = &updatedBy

	err = suite.repository.Update(suite.ctx, wallet)
	require.NoError(suite.T(), err)

	// Verificar atualização
	found, err := suite.repository.GetByID(suite.ctx, wallet.ID, suite.tenantID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Carteira Atualizada", found.Name)
	assert.Equal(suite.T(), "Nova descrição", found.Description)
	assert.Equal(suite.T(), updatedBy, *found.UpdatedBy)
}

// TestDeleteWallet testa exclusão de carteira
func (suite *WalletIntegrationTestSuite) TestDeleteWallet() {
	// Criar carteira
	wallet := &entities.Wallet{
		TenantID:  suite.tenantID,
		Name:      "Carteira para Deletar",
		Type:      entities.WalletTypePersonal,
		Currency:  "BRL",
		OwnerID:   suite.ownerID,
		OwnerType: "user",
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, wallet)
	require.NoError(suite.T(), err)

	// Deletar carteira
	err = suite.repository.Delete(suite.ctx, wallet.ID, suite.tenantID)
	require.NoError(suite.T(), err)

	// Verificar se foi deletada (soft delete)
	_, err = suite.repository.GetByID(suite.ctx, wallet.ID, suite.tenantID)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsNotFoundError(err))
}

// TestUpdateBalance testa atualização de saldo
func (suite *WalletIntegrationTestSuite) TestUpdateBalance() {
	// Criar carteira
	wallet := &entities.Wallet{
		TenantID:  suite.tenantID,
		Name:      "Carteira para Saldo",
		Type:      entities.WalletTypePersonal,
		Currency:  "BRL",
		OwnerID:   suite.ownerID,
		OwnerType: "user",
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, wallet)
	require.NoError(suite.T(), err)

	newBalance := int64(50000) // R$ 500,00

	// Atualizar saldo
	err = suite.repository.UpdateBalance(suite.ctx, wallet.ID, suite.tenantID, newBalance)
	require.NoError(suite.T(), err)

	// Verificar saldo atualizado
	found, err := suite.repository.GetByID(suite.ctx, wallet.ID, suite.tenantID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), newBalance, found.Balance)
}

// TestTenantIsolation testa isolamento entre tenants
func (suite *WalletIntegrationTestSuite) TestTenantIsolation() {
	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Criar carteira no tenant 1
	wallet1 := &entities.Wallet{
		TenantID:  tenant1,
		Name:      "Carteira Tenant 1",
		Type:      entities.WalletTypePersonal,
		Currency:  "BRL",
		OwnerID:   suite.ownerID,
		OwnerType: "user",
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, wallet1)
	require.NoError(suite.T(), err)

	// Tentar buscar carteira do tenant 1 usando tenant 2 (deve falhar)
	_, err = suite.repository.GetByID(suite.ctx, wallet1.ID, tenant2)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsNotFoundError(err))

	// Cleanup
	suite.db.Exec("DELETE FROM wallets WHERE tenant_id IN (?, ?)", tenant1, tenant2)
}

// TestInSuite executa a suite de testes
func TestWalletIntegrationSuite(t *testing.T) {
	suite.Run(t, new(WalletIntegrationTestSuite))
}
