package entities

import (
	"time"

	"suno-wallets/src/shared/helpers"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserStatus representa o status de um usuário
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBlocked  UserStatus = "blocked"
	UserStatusPending  UserStatus = "pending"
)

// User representa um usuário no sistema
type User struct {
	// Identificação primária
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`

	// Informações do usuário
	Name   string     `json:"name" gorm:"type:varchar(255);not null"`
	Email  string     `json:"email" gorm:"type:varchar(255);not null;index"`
	Phone  string     `json:"phone" gorm:"type:varchar(20)"`
	Status UserStatus `json:"status" gorm:"type:varchar(50);not null;default:'active'"`

	// Metadados (JSON para flexibilidade)
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`

	// Campos de auditoria obrigatórios
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	CreatedBy uuid.UUID      `json:"created_by" gorm:"type:uuid"`
	UpdatedBy *uuid.UUID     `json:"updated_by,omitempty" gorm:"type:uuid"`
}

// TableName define o nome da tabela no banco de dados
func (User) TableName() string {
	return "users"
}

// BeforeCreate hook executado antes da criação
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// IsActive verifica se o usuário está ativo
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// CanPerformActions verifica se o usuário pode realizar ações no sistema
func (u *User) CanPerformActions() bool {
	return u.Status == UserStatusActive
}

// GetDisplayName retorna o nome de exibição do usuário
func (u *User) GetDisplayName() string {
	if u.Name != "" {
		return u.Name
	}
	return u.Email
}

// Validate valida os dados do usuário
func (u *User) Validate() error {
	if u.TenantID == uuid.Nil {
		return NewValidationError("tenant_id é obrigatório")
	}

	if u.Name == "" {
		return NewValidationError("nome é obrigatório")
	}

	if len(u.Name) > 255 {
		return NewValidationError("nome não pode ter mais de 255 caracteres")
	}

	if u.Email == "" {
		return NewValidationError("email é obrigatório")
	}

	if len(u.Email) > 255 {
		return NewValidationError("email não pode ter mais de 255 caracteres")
	}

	// Validação básica de email
	if !isValidEmail(u.Email) {
		return NewValidationError("formato de email inválido")
	}

	// Validar telefone se fornecido
	if u.Phone != "" && len(u.Phone) > 20 {
		return NewValidationError("telefone não pode ter mais de 20 caracteres")
	}

	// Validar status
	validStatuses := []UserStatus{UserStatusActive, UserStatusInactive, UserStatusBlocked, UserStatusPending}
	if !helpers.Contains(validStatuses, u.Status) {
		return NewValidationError("status do usuário inválido")
	}

	return nil
}

// isValidEmail validação básica de formato de email
func isValidEmail(email string) bool {
	// Implementação básica - em produção usar regex mais robusta
	emailRunes := []rune(email)
	return len(email) > 3 && 
		   helpers.ContainsRune(emailRunes, '@') && 
		   helpers.ContainsRune(emailRunes, '.')
}
