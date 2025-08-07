package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WalletStatus representa o status de uma carteira
type WalletStatus string

const (
	WalletStatusActive    WalletStatus = "active"
	WalletStatusInactive  WalletStatus = "inactive"
	WalletStatusBlocked   WalletStatus = "blocked"
	WalletStatusSuspended WalletStatus = "suspended"
)

// WalletType representa o tipo de carteira
type WalletType string

const (
	WalletTypePersonal  WalletType = "personal"
	WalletTypeBusiness  WalletType = "business"
	WalletTypeCorporate WalletType = "corporate"
)

// Wallet representa uma carteira digital no sistema
type Wallet struct {
	// Identificação primária
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`

	// Informações da carteira
	Name        string       `json:"name" gorm:"type:varchar(255);not null"`
	Description string       `json:"description" gorm:"type:text"`
	Type        WalletType   `json:"type" gorm:"type:varchar(50);not null"`
	Status      WalletStatus `json:"status" gorm:"type:varchar(50);not null"`

	// Informações financeiras
	Balance   int64  `json:"balance" gorm:"type:bigint;not null"` // Em centavos
	Currency  string `json:"currency" gorm:"type:varchar(3);not null"`
	IsDefault bool   `json:"is_default" gorm:"type:boolean;not null"`

	// Informações do proprietário
	OwnerID   uuid.UUID `json:"owner_id" gorm:"type:uuid;not null;index"`
	OwnerType string    `json:"owner_type" gorm:"type:varchar(50);not null"` // user, business, etc.

	// Configurações
	DailyLimit   *int64 `json:"daily_limit,omitempty" gorm:"type:bigint"`   // Em centavos
	MonthlyLimit *int64 `json:"monthly_limit,omitempty" gorm:"type:bigint"` // Em centavos
	AllowDebit   bool   `json:"allow_debit" gorm:"type:boolean;not null"`

	// Metadados (JSON para flexibilidade)
	Metadata datatypes.JSON `json:"metadata,omitempty"`

	// Campos de auditoria obrigatórios
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	CreatedBy uuid.UUID      `json:"created_by" gorm:"type:uuid"`
	UpdatedBy *uuid.UUID     `json:"updated_by,omitempty" gorm:"type:uuid"`

	// Relacionamentos (serão implementados conforme necessidade)
	// Transactions []Transaction `json:"transactions,omitempty" gorm:"foreignKey:WalletID"`
}

// TableName define o nome da tabela no banco de dados
func (Wallet) TableName() string {
	return "wallets"
}

// BeforeCreate hook executado antes da criação
func (w *Wallet) BeforeCreate(_ *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// IsActive verifica se a carteira está ativa
func (w *Wallet) IsActive() bool {
	return w.Status == WalletStatusActive
}

// CanDebit verifica se a carteira permite débito
func (w *Wallet) CanDebit() bool {
	return w.AllowDebit && w.IsActive()
}

// HasSufficientBalance verifica se há saldo suficiente para operação
func (w *Wallet) HasSufficientBalance(amount int64) bool {
	return w.Balance >= amount
}

// GetBalanceInReais retorna o saldo em reais (formato decimal)
func (w *Wallet) GetBalanceInReais() float64 {
	return float64(w.Balance) / 100
}

// GetDailyLimitInReais retorna o limite diário em reais
func (w *Wallet) GetDailyLimitInReais() *float64 {
	if w.DailyLimit == nil {
		return nil
	}
	limit := float64(*w.DailyLimit) / 100
	return &limit
}

// GetMonthlyLimitInReais retorna o limite mensal em reais
func (w *Wallet) GetMonthlyLimitInReais() *float64 {
	if w.MonthlyLimit == nil {
		return nil
	}
	limit := float64(*w.MonthlyLimit) / 100
	return &limit
}

// Validate valida os dados da carteira
func (w *Wallet) Validate() error {
	if w.TenantID == uuid.Nil {
		return NewValidationError("tenant_id é obrigatório")
	}

	if w.Name == "" {
		return NewValidationError("nome da carteira é obrigatório")
	}

	if len(w.Name) > 255 {
		return NewValidationError("nome da carteira não pode ter mais de 255 caracteres")
	}

	if w.OwnerID == uuid.Nil {
		return NewValidationError("owner_id é obrigatório")
	}

	if w.Currency == "" {
		w.Currency = "BRL" // Valor padrão
	}

	if len(w.Currency) != 3 {
		return NewValidationError("moeda deve ter exatamente 3 caracteres")
	}

	// Validar tipo de carteira
	validTypes := []WalletType{WalletTypePersonal, WalletTypeBusiness, WalletTypeCorporate}
	if !contains(validTypes, w.Type) {
		return NewValidationError("tipo de carteira inválido")
	}

	// Validar status
	validStatuses := []WalletStatus{WalletStatusActive, WalletStatusInactive, WalletStatusBlocked, WalletStatusSuspended}
	if !contains(validStatuses, w.Status) {
		return NewValidationError("status da carteira inválido")
	}

	return nil
}

// contains verifica se um slice contém um valor
func contains[T comparable](slice []T, item T) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
