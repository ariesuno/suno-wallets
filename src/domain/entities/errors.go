package entities

import "errors"

// Erros de domínio customizados

// ValidationError representa um erro de validação de entidade
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// NewValidationError cria um novo erro de validação
func NewValidationError(message string) ValidationError {
	return ValidationError{
		Message: message,
	}
}

// NewFieldValidationError cria um novo erro de validação para um campo específico
func NewFieldValidationError(field, message string) ValidationError {
	return ValidationError{
		Field:   field,
		Message: message,
	}
}

// BusinessError representa um erro de regra de negócio
type BusinessError struct {
	Code    string
	Message string
}

func (e BusinessError) Error() string {
	return e.Message
}

// NewBusinessError cria um novo erro de negócio
func NewBusinessError(code, message string) BusinessError {
	return BusinessError{
		Code:    code,
		Message: message,
	}
}

// Erros pré-definidos do domínio
var (
	ErrWalletNotFound         = NewBusinessError("WALLET_NOT_FOUND", "Carteira não encontrada")
	ErrInsufficientBalance    = NewBusinessError("INSUFFICIENT_BALANCE", "Saldo insuficiente")
	ErrWalletInactive         = NewBusinessError("WALLET_INACTIVE", "Carteira inativa")
	ErrWalletBlocked          = NewBusinessError("WALLET_BLOCKED", "Carteira bloqueada")
	ErrDebitNotAllowed        = NewBusinessError("DEBIT_NOT_ALLOWED", "Débito não permitido")
	ErrInvalidTenant          = NewBusinessError("INVALID_TENANT", "Inquilino inválido")
	ErrUnauthorizedAccess     = NewBusinessError("UNAUTHORIZED_ACCESS", "Acesso não autorizado")
	ErrInvalidAmount          = NewBusinessError("INVALID_AMOUNT", "Valor inválido")
	ErrCurrencyMismatch       = NewBusinessError("CURRENCY_MISMATCH", "Moedas incompatíveis")
	ErrDailyLimitExceeded     = NewBusinessError("DAILY_LIMIT_EXCEEDED", "Limite diário excedido")
	ErrMonthlyLimitExceeded   = NewBusinessError("MONTHLY_LIMIT_EXCEEDED", "Limite mensal excedido")
	ErrDuplicateDefaultWallet = NewBusinessError("DUPLICATE_DEFAULT_WALLET", "Apenas uma carteira padrão é permitida por usuário")
)

// NotFoundError representa um erro de recurso não encontrado
type NotFoundError struct {
	Resource string
	ID       string
}

func (e NotFoundError) Error() string {
	return "recurso '" + e.Resource + "' com ID '" + e.ID + "' não encontrado"
}

// NewNotFoundError cria um novo erro de recurso não encontrado
func NewNotFoundError(resource, id string) NotFoundError {
	return NotFoundError{
		Resource: resource,
		ID:       id,
	}
}

// ConflictError representa um erro de conflito
type ConflictError struct {
	Resource string
	Message  string
}

func (e ConflictError) Error() string {
	return e.Message
}

// NewConflictError cria um novo erro de conflito
func NewConflictError(resource, message string) ConflictError {
	return ConflictError{
		Resource: resource,
		Message:  message,
	}
}

// IsValidationError verifica se o erro é de validação
func IsValidationError(err error) bool {
	var validationErr ValidationError
	return errors.As(err, &validationErr)
}

// IsBusinessError verifica se o erro é de negócio
func IsBusinessError(err error) bool {
	var businessErr BusinessError
	return errors.As(err, &businessErr)
}

// IsNotFoundError verifica se o erro é de recurso não encontrado
func IsNotFoundError(err error) bool {
	var notFoundErr NotFoundError
	return errors.As(err, &notFoundErr)
}

// IsConflictError verifica se o erro é de conflito
func IsConflictError(err error) bool {
	var conflictErr ConflictError
	return errors.As(err, &conflictErr)
}
