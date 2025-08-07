package valueobjects

// ValidationError representa um erro de validação de value object
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
