package entities

import (
	"time"

	"suno-wallets/src/domain/enums"
	"suno-wallets/src/domain/valueobjects"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User representa um usuário do sistema
type User struct {
	// Identificação primária
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`

	// Informações pessoais
	Email     *valueobjects.Email `json:"email" gorm:"embedded;embeddedPrefix:email_"`
	CPF       *valueobjects.CPF   `json:"cpf,omitempty" gorm:"embedded;embeddedPrefix:cpf_"`
	FirstName string              `json:"first_name" gorm:"type:varchar(100);not null"`
	LastName  string              `json:"last_name" gorm:"type:varchar(100);not null"`
	FullName  string              `json:"full_name" gorm:"type:varchar(255);not null"`

	// Informações de contato
	Phone       string     `json:"phone,omitempty" gorm:"type:varchar(20)"`
	DateOfBirth *time.Time `json:"date_of_birth,omitempty"`
	Nationality string     `json:"nationality,omitempty" gorm:"type:varchar(3);default:'BRA'"` // Código ISO 3166-1 alpha-3

	// Status e configurações
	Status           enums.UserStatus `json:"status" gorm:"type:varchar(50);not null;default:'pending'"`
	IsEmailVerified  bool             `json:"is_email_verified" gorm:"type:boolean;not null;default:false"`
	IsPhoneVerified  bool             `json:"is_phone_verified" gorm:"type:boolean;not null;default:false"`
	IsKYCCompleted   bool             `json:"is_kyc_completed" gorm:"type:boolean;not null;default:false"`
	TwoFactorEnabled bool             `json:"two_factor_enabled" gorm:"type:boolean;not null;default:false"`

	// Configurações de preferência
	PreferredLanguage string                 `json:"preferred_language" gorm:"type:varchar(5);default:'pt-BR'"`
	PreferredCurrency enums.CurrencyType     `json:"preferred_currency" gorm:"type:varchar(10);default:'BRL'"`
	TimeZone          string                 `json:"timezone" gorm:"type:varchar(50);default:'America/Sao_Paulo'"`
	NotificationPrefs map[string]interface{} `json:"notification_preferences,omitempty" gorm:"type:jsonb"`

	// Informações de segurança
	PasswordHash     string     `json:"-" gorm:"type:varchar(255);not null"` // Hash da senha
	LastLoginAt      *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP      string     `json:"last_login_ip,omitempty" gorm:"type:inet"`
	FailedLoginCount int        `json:"failed_login_count" gorm:"type:int;default:0"`
	LockedUntil      *time.Time `json:"locked_until,omitempty"`

	// Metadados e limites
	DailyTransactionLimit   *int64                 `json:"daily_transaction_limit,omitempty" gorm:"type:bigint"` // Em centavos
	MonthlyTransactionLimit *int64                 `json:"monthly_transaction_limit,omitempty" gorm:"type:bigint"`
	YearlyTransactionLimit  *int64                 `json:"yearly_transaction_limit,omitempty" gorm:"type:bigint"`
	RiskLevel               string                 `json:"risk_level" gorm:"type:varchar(20);default:'low'"` // low, medium, high
	Tags                    []string               `json:"tags,omitempty" gorm:"type:text[]"`
	Metadata                map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`

	// Campos de auditoria obrigatórios
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	CreatedBy uuid.UUID      `json:"created_by" gorm:"type:uuid"`
	UpdatedBy *uuid.UUID     `json:"updated_by,omitempty" gorm:"type:uuid"`

	// Relacionamentos (serão carregados conforme necessidade)
	// Wallets     []Wallet     `json:"wallets,omitempty" gorm:"foreignKey:OwnerID"`
	// Portfolios  []Portfolio  `json:"portfolios,omitempty" gorm:"foreignKey:OwnerID"`
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

	// Gerar nome completo
	u.FullName = u.GetFullName()

	return nil
}

// BeforeUpdate hook executado antes da atualização
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	// Atualizar nome completo
	u.FullName = u.GetFullName()

	return nil
}

// GetFullName retorna o nome completo do usuário
func (u *User) GetFullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return ""
	}
	return u.FirstName + " " + u.LastName
}

// IsActive verifica se o usuário está ativo
func (u *User) IsActive() bool {
	return u.Status.CanAccess()
}

// CanTrade verifica se o usuário pode realizar transações
func (u *User) CanTrade() bool {
	return u.Status.CanTrade() && !u.IsLocked()
}

// IsLocked verifica se a conta está bloqueada
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// IsAdult verifica se o usuário é maior de idade
func (u *User) IsAdult() bool {
	if u.DateOfBirth == nil {
		return false
	}

	now := time.Now()
	age := now.Year() - u.DateOfBirth.Year()

	// Verificar se ainda não fez aniversário este ano
	if now.YearDay() < u.DateOfBirth.YearDay() {
		age--
	}

	return age >= 18
}

// GetAge retorna a idade do usuário
func (u *User) GetAge() int {
	if u.DateOfBirth == nil {
		return 0
	}

	now := time.Now()
	age := now.Year() - u.DateOfBirth.Year()

	// Verificar se ainda não fez aniversário este ano
	if now.YearDay() < u.DateOfBirth.YearDay() {
		age--
	}

	return age
}

// GetDailyLimitInReais retorna o limite diário em reais
func (u *User) GetDailyLimitInReais() *float64 {
	if u.DailyTransactionLimit == nil {
		return nil
	}
	limit := float64(*u.DailyTransactionLimit) / 100
	return &limit
}

// GetMonthlyLimitInReais retorna o limite mensal em reais
func (u *User) GetMonthlyLimitInReais() *float64 {
	if u.MonthlyTransactionLimit == nil {
		return nil
	}
	limit := float64(*u.MonthlyTransactionLimit) / 100
	return &limit
}

// GetYearlyLimitInReais retorna o limite anual em reais
func (u *User) GetYearlyLimitInReais() *float64 {
	if u.YearlyTransactionLimit == nil {
		return nil
	}
	limit := float64(*u.YearlyTransactionLimit) / 100
	return &limit
}

// IncrementFailedLogin incrementa o contador de tentativas de login falhadas
func (u *User) IncrementFailedLogin() {
	u.FailedLoginCount++

	// Bloquear temporariamente após 5 tentativas
	if u.FailedLoginCount >= 5 {
		lockDuration := time.Duration(u.FailedLoginCount-4) * 15 * time.Minute
		lockUntil := time.Now().Add(lockDuration)
		u.LockedUntil = &lockUntil
	}
}

// ResetFailedLogin reseta o contador de tentativas de login falhadas
func (u *User) ResetFailedLogin() {
	u.FailedLoginCount = 0
	u.LockedUntil = nil
}

// UpdateLastLogin atualiza informações do último login
func (u *User) UpdateLastLogin(ipAddress string) {
	now := time.Now()
	u.LastLoginAt = &now
	u.LastLoginIP = ipAddress
	u.ResetFailedLogin()
}

// HasTag verifica se o usuário tem uma tag específica
func (u *User) HasTag(tag string) bool {
	for _, t := range u.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adiciona uma tag ao usuário
func (u *User) AddTag(tag string) {
	if !u.HasTag(tag) {
		u.Tags = append(u.Tags, tag)
	}
}

// RemoveTag remove uma tag do usuário
func (u *User) RemoveTag(tag string) {
	for i, t := range u.Tags {
		if t == tag {
			u.Tags = append(u.Tags[:i], u.Tags[i+1:]...)
			break
		}
	}
}

// IsHighRisk verifica se o usuário é de alto risco
func (u *User) IsHighRisk() bool {
	return u.RiskLevel == "high"
}

// RequiresKYC verifica se o usuário precisa completar KYC
func (u *User) RequiresKYC() bool {
	return u.Status == enums.UserStatusKYCRequired || !u.IsKYCCompleted
}

// GetEmailAddress retorna o endereço de email como string
func (u *User) GetEmailAddress() string {
	if u.Email == nil {
		return ""
	}
	return u.Email.Value()
}

// GetCPFNumber retorna o CPF como string
func (u *User) GetCPFNumber() string {
	if u.CPF == nil {
		return ""
	}
	return u.CPF.Value()
}

// Validate valida os dados do usuário
func (u *User) Validate() error {
	if u.TenantID == uuid.Nil {
		return NewValidationError("tenant_id é obrigatório")
	}

	if u.Email == nil {
		return NewValidationError("email é obrigatório")
	}

	if err := u.Email.Validate(); err != nil {
		return err
	}

	if u.FirstName == "" {
		return NewValidationError("primeiro nome é obrigatório")
	}

	if len(u.FirstName) > 100 {
		return NewValidationError("primeiro nome não pode ter mais de 100 caracteres")
	}

	if u.LastName == "" {
		return NewValidationError("último nome é obrigatório")
	}

	if len(u.LastName) > 100 {
		return NewValidationError("último nome não pode ter mais de 100 caracteres")
	}

	if !u.Status.IsValid() {
		return NewValidationError("status do usuário inválido")
	}

	if !u.PreferredCurrency.IsValid() {
		return NewValidationError("moeda preferida inválida")
	}

	// Validar CPF se fornecido
	if u.CPF != nil {
		if err := u.CPF.Validate(); err != nil {
			return err
		}
	}

	// Validar data de nascimento
	if u.DateOfBirth != nil {
		if u.DateOfBirth.After(time.Now()) {
			return NewValidationError("data de nascimento não pode ser no futuro")
		}

		// Verificar idade mínima (13 anos para compliance)
		minAge := time.Now().AddDate(-13, 0, 0)
		if u.DateOfBirth.After(minAge) {
			return NewValidationError("usuário deve ter pelo menos 13 anos")
		}
	}

	return nil
}
