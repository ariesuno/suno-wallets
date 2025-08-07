package memory_test

import (
	"context"
	"testing"

	"suno-wallets/src/domain/entities"
	"suno-wallets/src/domain/repositories"
	"suno-wallets/src/domain/repositories/memory"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// UserMemoryRepositoryTestSuite suite para testes do repositório em memória de usuários
type UserMemoryRepositoryTestSuite struct {
	suite.Suite
	repository repositories.UserRepository
	ctx        context.Context
	tenantID   uuid.UUID
	createdBy  uuid.UUID
}

// SetupTest configura o ambiente antes de cada teste
func (suite *UserMemoryRepositoryTestSuite) SetupTest() {
	suite.repository = memory.NewUserMemoryRepository()
	suite.ctx = context.Background()
	suite.tenantID = uuid.New()
	suite.createdBy = uuid.New()
}

// TestCreateUser testa criação de usuário
func (suite *UserMemoryRepositoryTestSuite) TestCreateUser() {
	user := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Silva",
		Email:     "joao@example.com",
		Phone:     "+5511999999999",
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, user)
	require.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), uuid.Nil, user.ID)
	assert.False(suite.T(), user.CreatedAt.IsZero())
	assert.False(suite.T(), user.UpdatedAt.IsZero())
}

// TestCreateDuplicateEmail testa erro ao criar usuário com email duplicado
func (suite *UserMemoryRepositoryTestSuite) TestCreateDuplicateEmail() {
	email := "joao@example.com"

	// Criar primeiro usuário
	user1 := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Silva",
		Email:     email,
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, user1)
	require.NoError(suite.T(), err)

	// Tentar criar segundo usuário com mesmo email (deve falhar)
	user2 := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Santos",
		Email:     email,
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err = suite.repository.Create(suite.ctx, user2)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsConflictError(err))
}

// TestCreateValidation testa validação durante criação
func (suite *UserMemoryRepositoryTestSuite) TestCreateValidation() {
	user := &entities.User{
		TenantID: uuid.Nil, // TenantID inválido
		Name:     "João Silva",
		Email:    "joao@example.com",
	}

	err := suite.repository.Create(suite.ctx, user)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsValidationError(err))
}

// TestFindByID testa busca de usuário por ID
func (suite *UserMemoryRepositoryTestSuite) TestFindByID() {
	// Criar usuário
	user := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Silva",
		Email:     "joao@example.com",
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Buscar usuário
	found, err := suite.repository.FindByID(suite.ctx, user.ID, suite.tenantID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, found.ID)
	assert.Equal(suite.T(), user.Name, found.Name)
	assert.Equal(suite.T(), user.Email, found.Email)
}

// TestFindByIDNotFound testa busca de usuário inexistente
func (suite *UserMemoryRepositoryTestSuite) TestFindByIDNotFound() {
	nonExistentID := uuid.New()

	_, err := suite.repository.FindByID(suite.ctx, nonExistentID, suite.tenantID)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsNotFoundError(err))
}

// TestFindByIDTenantIsolation testa isolamento entre tenants
func (suite *UserMemoryRepositoryTestSuite) TestFindByIDTenantIsolation() {
	otherTenantID := uuid.New()

	// Criar usuário no tenant atual
	user := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Silva",
		Email:     "joao@example.com",
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Tentar buscar usando outro tenant (deve falhar)
	_, err = suite.repository.FindByID(suite.ctx, user.ID, otherTenantID)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsNotFoundError(err))
}

// TestFindByEmail testa busca de usuário por email
func (suite *UserMemoryRepositoryTestSuite) TestFindByEmail() {
	email := "joao@example.com"

	// Criar usuário
	user := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Silva",
		Email:     email,
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Buscar usuário por email
	found, err := suite.repository.FindByEmail(suite.ctx, email, suite.tenantID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, found.ID)
	assert.Equal(suite.T(), user.Email, found.Email)
}

// TestFindByEmailNotFound testa busca por email inexistente
func (suite *UserMemoryRepositoryTestSuite) TestFindByEmailNotFound() {
	_, err := suite.repository.FindByEmail(suite.ctx, "inexistente@example.com", suite.tenantID)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsNotFoundError(err))
}

// TestUpdateUser testa atualização de usuário
func (suite *UserMemoryRepositoryTestSuite) TestUpdateUser() {
	// Criar usuário
	user := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Silva",
		Email:     "joao@example.com",
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	originalCreatedAt := user.CreatedAt

	// Atualizar usuário
	user.Name = "João Santos"
	user.Phone = "+5511888888888"
	updatedBy := uuid.New()
	user.UpdatedBy = &updatedBy

	err = suite.repository.Update(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Verificar atualização
	found, err := suite.repository.FindByID(suite.ctx, user.ID, suite.tenantID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "João Santos", found.Name)
	assert.Equal(suite.T(), "+5511888888888", found.Phone)
	assert.Equal(suite.T(), originalCreatedAt, found.CreatedAt) // CreatedAt deve ser preservado
	assert.True(suite.T(), found.UpdatedAt.After(found.CreatedAt))
}

// TestUpdateUserNotFound testa atualização de usuário inexistente
func (suite *UserMemoryRepositoryTestSuite) TestUpdateUserNotFound() {
	user := &entities.User{
		ID:        uuid.New(),
		TenantID:  suite.tenantID,
		Name:      "João Silva",
		Email:     "joao@example.com",
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Update(suite.ctx, user)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsNotFoundError(err))
}

// TestUpdateUserEmailConflict testa conflito de email na atualização
func (suite *UserMemoryRepositoryTestSuite) TestUpdateUserEmailConflict() {
	// Criar dois usuários
	user1 := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Silva",
		Email:     "joao1@example.com",
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	user2 := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Santos",
		Email:     "joao2@example.com",
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, user1)
	require.NoError(suite.T(), err)

	err = suite.repository.Create(suite.ctx, user2)
	require.NoError(suite.T(), err)

	// Tentar atualizar user2 com email do user1 (deve falhar)
	user2.Email = "joao1@example.com"

	err = suite.repository.Update(suite.ctx, user2)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsConflictError(err))
}

// TestDeleteUser testa exclusão de usuário
func (suite *UserMemoryRepositoryTestSuite) TestDeleteUser() {
	// Criar usuário
	user := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Silva",
		Email:     "joao@example.com",
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err := suite.repository.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Deletar usuário
	err = suite.repository.Delete(suite.ctx, user.ID, suite.tenantID)
	require.NoError(suite.T(), err)

	// Verificar se foi deletado (soft delete)
	_, err = suite.repository.FindByID(suite.ctx, user.ID, suite.tenantID)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsNotFoundError(err))
}

// TestDeleteUserNotFound testa exclusão de usuário inexistente
func (suite *UserMemoryRepositoryTestSuite) TestDeleteUserNotFound() {
	nonExistentID := uuid.New()

	err := suite.repository.Delete(suite.ctx, nonExistentID, suite.tenantID)
	assert.Error(suite.T(), err)
	assert.True(suite.T(), entities.IsNotFoundError(err))
}

// TestListUsers testa listagem de usuários
func (suite *UserMemoryRepositoryTestSuite) TestListUsers() {
	// Criar múltiplos usuários
	users := []*entities.User{
		{
			TenantID:  suite.tenantID,
			Name:      "João Silva",
			Email:     "joao1@example.com",
			Status:    entities.UserStatusActive,
			CreatedBy: suite.createdBy,
		},
		{
			TenantID:  suite.tenantID,
			Name:      "Maria Santos",
			Email:     "maria@example.com",
			Status:    entities.UserStatusActive,
			CreatedBy: suite.createdBy,
		},
		{
			TenantID:  suite.tenantID,
			Name:      "Pedro Oliveira",
			Email:     "pedro@example.com",
			Status:    entities.UserStatusInactive,
			CreatedBy: suite.createdBy,
		},
	}

	for _, user := range users {
		err := suite.repository.Create(suite.ctx, user)
		require.NoError(suite.T(), err)
	}

	// Listar todos os usuários
	params := repositories.ListUserParams{
		TenantID: suite.tenantID,
		Page:     1,
		Limit:    10,
	}

	found, total, err := suite.repository.List(suite.ctx, params)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), found, 3)
	assert.Equal(suite.T(), int64(3), total)
}

// TestListUsersWithFilters testa listagem com filtros
func (suite *UserMemoryRepositoryTestSuite) TestListUsersWithFilters() {
	// Criar usuários com diferentes status
	users := []*entities.User{
		{
			TenantID:  suite.tenantID,
			Name:      "João Silva",
			Email:     "joao@example.com",
			Status:    entities.UserStatusActive,
			CreatedBy: suite.createdBy,
		},
		{
			TenantID:  suite.tenantID,
			Name:      "Maria Santos",
			Email:     "maria@example.com",
			Status:    entities.UserStatusInactive,
			CreatedBy: suite.createdBy,
		},
	}

	for _, user := range users {
		err := suite.repository.Create(suite.ctx, user)
		require.NoError(suite.T(), err)
	}

	// Filtrar apenas usuários ativos
	status := entities.UserStatusActive
	params := repositories.ListUserParams{
		TenantID: suite.tenantID,
		Status:   &status,
		Page:     1,
		Limit:    10,
	}

	found, total, err := suite.repository.List(suite.ctx, params)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), found, 1)
	assert.Equal(suite.T(), int64(1), total)
	assert.Equal(suite.T(), entities.UserStatusActive, found[0].Status)
}

// TestListUsersWithSearch testa listagem com busca
func (suite *UserMemoryRepositoryTestSuite) TestListUsersWithSearch() {
	// Criar usuários
	users := []*entities.User{
		{
			TenantID:  suite.tenantID,
			Name:      "João Silva",
			Email:     "joao@example.com",
			Status:    entities.UserStatusActive,
			CreatedBy: suite.createdBy,
		},
		{
			TenantID:  suite.tenantID,
			Name:      "Maria Santos",
			Email:     "maria@example.com",
			Status:    entities.UserStatusActive,
			CreatedBy: suite.createdBy,
		},
	}

	for _, user := range users {
		err := suite.repository.Create(suite.ctx, user)
		require.NoError(suite.T(), err)
	}

	// Buscar por "João"
	params := repositories.ListUserParams{
		TenantID: suite.tenantID,
		Search:   "João",
		Page:     1,
		Limit:    10,
	}

	found, total, err := suite.repository.List(suite.ctx, params)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), found, 1)
	assert.Equal(suite.T(), int64(1), total)
	assert.Contains(suite.T(), found[0].Name, "João")
}

// TestExistsByEmail testa verificação de existência por email
func (suite *UserMemoryRepositoryTestSuite) TestExistsByEmail() {
	email := "joao@example.com"

	// Verificar antes de criar (não deve existir)
	exists, err := suite.repository.ExistsByEmail(suite.ctx, email, suite.tenantID, nil)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), exists)

	// Criar usuário
	user := &entities.User{
		TenantID:  suite.tenantID,
		Name:      "João Silva",
		Email:     email,
		Status:    entities.UserStatusActive,
		CreatedBy: suite.createdBy,
	}

	err = suite.repository.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Verificar depois de criar (deve existir)
	exists, err = suite.repository.ExistsByEmail(suite.ctx, email, suite.tenantID, nil)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// Verificar excluindo o próprio ID (não deve existir)
	exists, err = suite.repository.ExistsByEmail(suite.ctx, email, suite.tenantID, &user.ID)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

// TestInSuite executa a suite de testes
func TestUserMemoryRepositorySuite(t *testing.T) {
	suite.Run(t, new(UserMemoryRepositoryTestSuite))
}
