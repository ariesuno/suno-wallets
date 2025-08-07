package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"suno-wallets/src/domain/entities"
	"suno-wallets/src/domain/repositories"

	"github.com/google/uuid"
)

// userMemoryRepository implementação em memória do repositório de usuários
type userMemoryRepository struct {
	users map[uuid.UUID]*entities.User
	mutex sync.RWMutex
}

// NewUserMemoryRepository cria uma nova instância do repositório em memória
func NewUserMemoryRepository() repositories.UserRepository {
	return &userMemoryRepository{
		users: make(map[uuid.UUID]*entities.User),
		mutex: sync.RWMutex{},
	}
}

// Create cria um novo usuário
func (r *userMemoryRepository) Create(ctx context.Context, user *entities.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verificar se já existe usuário com o mesmo email
	for _, existingUser := range r.users {
		if existingUser.Email == user.Email && 
		   existingUser.TenantID == user.TenantID && 
		   existingUser.DeletedAt.Time.IsZero() {
			return entities.NewConflictError("user", "usuário com este email já existe")
		}
	}

	// Gerar ID se não fornecido
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	// Definir timestamps
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Clonar o usuário para evitar modificações externas
	userCopy := *user
	r.users[user.ID] = &userCopy

	return nil
}

// FindByID busca um usuário por ID e tenant
func (r *userMemoryRepository) FindByID(ctx context.Context, id, tenantID uuid.UUID) (*entities.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	user, exists := r.users[id]
	if !exists || user.TenantID != tenantID || !user.DeletedAt.Time.IsZero() {
		return nil, entities.NewNotFoundError("user", id.String())
	}

	// Retornar cópia para evitar modificações externas
	userCopy := *user
	return &userCopy, nil
}

// FindByEmail busca um usuário por email e tenant
func (r *userMemoryRepository) FindByEmail(ctx context.Context, email string, tenantID uuid.UUID) (*entities.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, user := range r.users {
		if user.Email == email && 
		   user.TenantID == tenantID && 
		   user.DeletedAt.Time.IsZero() {
			// Retornar cópia para evitar modificações externas
			userCopy := *user
			return &userCopy, nil
		}
	}

	return nil, entities.NewNotFoundError("user", email)
}

// Update atualiza um usuário existente
func (r *userMemoryRepository) Update(ctx context.Context, user *entities.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	existingUser, exists := r.users[user.ID]
	if !exists || existingUser.TenantID != user.TenantID || !existingUser.DeletedAt.Time.IsZero() {
		return entities.NewNotFoundError("user", user.ID.String())
	}

	// Verificar se o email não está sendo usado por outro usuário
	for id, otherUser := range r.users {
		if id != user.ID && 
		   otherUser.Email == user.Email && 
		   otherUser.TenantID == user.TenantID && 
		   otherUser.DeletedAt.Time.IsZero() {
			return entities.NewConflictError("user", "email já está sendo usado por outro usuário")
		}
	}

	// Preservar campos de criação e atualizar timestamp
	user.CreatedAt = existingUser.CreatedAt
	user.CreatedBy = existingUser.CreatedBy
	user.UpdatedAt = time.Now().UTC()

	// Atualizar usuário
	userCopy := *user
	r.users[user.ID] = &userCopy

	return nil
}

// Delete remove um usuário (soft delete)
func (r *userMemoryRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	user, exists := r.users[id]
	if !exists || user.TenantID != tenantID || !user.DeletedAt.Time.IsZero() {
		return entities.NewNotFoundError("user", id.String())
	}

	// Soft delete
	user.DeletedAt.Time = time.Now().UTC()
	user.DeletedAt.Valid = true

	return nil
}

// List lista usuários com paginação e filtros
func (r *userMemoryRepository) List(ctx context.Context, params repositories.ListUserParams) ([]*entities.User, int64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var filteredUsers []*entities.User

	// Aplicar filtros
	for _, user := range r.users {
		// Filtrar por tenant e não deletados
		if user.TenantID != params.TenantID || !user.DeletedAt.Time.IsZero() {
			continue
		}

		// Filtrar por status se especificado
		if params.Status != nil && user.Status != *params.Status {
			continue
		}

		// Filtrar por busca se especificado
		if params.Search != "" {
			searchLower := strings.ToLower(params.Search)
			if !strings.Contains(strings.ToLower(user.Name), searchLower) &&
			   !strings.Contains(strings.ToLower(user.Email), searchLower) {
				continue
			}
		}

		filteredUsers = append(filteredUsers, user)
	}

	total := int64(len(filteredUsers))

	// Aplicar ordenação (simplificada)
	// Em uma implementação real, seria mais robusta

	// Aplicar paginação
	if params.Limit > 0 {
		start := (params.Page - 1) * params.Limit
		end := start + params.Limit

		if start >= len(filteredUsers) {
			return []*entities.User{}, total, nil
		}

		if end > len(filteredUsers) {
			end = len(filteredUsers)
		}

		filteredUsers = filteredUsers[start:end]
	}

	// Retornar cópias para evitar modificações externas
	result := make([]*entities.User, len(filteredUsers))
	for i, user := range filteredUsers {
		userCopy := *user
		result[i] = &userCopy
	}

	return result, total, nil
}

// ExistsByEmail verifica se já existe usuário com o email
func (r *userMemoryRepository) ExistsByEmail(ctx context.Context, email string, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for id, user := range r.users {
		if user.Email == email && 
		   user.TenantID == tenantID && 
		   user.DeletedAt.Time.IsZero() {
			// Se deve excluir um ID específico, verificar
			if excludeID != nil && id == *excludeID {
				continue
			}
			return true, nil
		}
	}

	return false, nil
}

// GetAllUsers retorna todos os usuários (método auxiliar para testes)
func (r *userMemoryRepository) GetAllUsers() map[uuid.UUID]*entities.User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[uuid.UUID]*entities.User)
	for id, user := range r.users {
		userCopy := *user
		result[id] = &userCopy
	}

	return result
}

// Clear limpa todos os usuários (método auxiliar para testes)
func (r *userMemoryRepository) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.users = make(map[uuid.UUID]*entities.User)
}
