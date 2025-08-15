package common

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// ValidatedRequest representa uma requisição que pode ser validada
type ValidatedRequest interface {
	Validate() []ValidationError
}

// ValidationError representa um erro de validação específico
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// ValidateAndBind valida e faz bind de uma requisição JSON
func ValidateAndBind[T ValidatedRequest](c *gin.Context, req T) bool {
	// Bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors := formatValidationErrors(err); len(validationErrors) > 0 {
			HandleBadRequest(c, "Dados de entrada inválidos", validationErrors)
		} else {
			HandleValidationError(c, err, "Formato JSON inválido")
		}
		return false
	}

	// Validação customizada
	if customErrors := req.Validate(); len(customErrors) > 0 {
		HandleBadRequest(c, "Erro de validação", customErrors)
		return false
	}

	return true
}

// formatValidationErrors formata erros de validação do gin/validator
func formatValidationErrors(err error) []ValidationError {
	var validationErrors []ValidationError

	if validatorErr, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validatorErr {
			validationError := ValidationError{
				Field: fieldError.Field(),
				Tag:   fieldError.Tag(),
				Value: fieldError.Param(),
			}

			// Mensagens customizadas baseadas na tag
			switch fieldError.Tag() {
			case "required":
				validationError.Message = "Campo obrigatório"
			case "email":
				validationError.Message = "Email inválido"
			case "min":
				validationError.Message = "Valor muito pequeno (mínimo: " + fieldError.Param() + ")"
			case "max":
				validationError.Message = "Valor muito grande (máximo: " + fieldError.Param() + ")"
			case "len":
				validationError.Message = "Tamanho deve ser exatamente " + fieldError.Param()
			case "uuid":
				validationError.Message = "UUID inválido"
			case "cpf":
				validationError.Message = "CPF inválido"
			default:
				validationError.Message = "Valor inválido"
			}

			validationErrors = append(validationErrors, validationError)
		}
	}

	return validationErrors
}

// ValidateUUID valida se uma string é um UUID válido
func ValidateUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

// ValidateCPF valida formato de CPF brasileiro
func ValidateCPF(cpf string) bool {
	// Remove caracteres especiais
	cpf = regexp.MustCompile(`[^0-9]`).ReplaceAllString(cpf, "")

	// Verifica se tem 11 dígitos
	if len(cpf) != 11 {
		return false
	}

	// Verifica se não são todos dígitos iguais
	if regexp.MustCompile(`^(\d)\1{10}$`).MatchString(cpf) {
		return false
	}

	// Validação dos dígitos verificadores
	return validateCPFDigits(cpf)
}

// validateCPFDigits valida os dígitos verificadores do CPF
func validateCPFDigits(cpf string) bool {
	// Primeiro dígito verificador
	sum := 0
	for i := 0; i < 9; i++ {
		digit := int(cpf[i] - '0')
		sum += digit * (10 - i)
	}

	remainder := sum % 11
	checkDigit1 := 0
	if remainder >= 2 {
		checkDigit1 = 11 - remainder
	}

	if int(cpf[9]-'0') != checkDigit1 {
		return false
	}

	// Segundo dígito verificador
	sum = 0
	for i := 0; i < 10; i++ {
		digit := int(cpf[i] - '0')
		sum += digit * (11 - i)
	}

	remainder = sum % 11
	checkDigit2 := 0
	if remainder >= 2 {
		checkDigit2 = 11 - remainder
	}

	return int(cpf[10]-'0') == checkDigit2
}

// ValidateEmail valida formato de email
func ValidateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// SanitizeString remove caracteres perigosos de strings
func SanitizeString(input string) string {
	// Remove caracteres de controle
	input = regexp.MustCompile(`[\x00-\x1f\x7f]`).ReplaceAllString(input, "")

	// Limita tamanho
	if len(input) > 1000 {
		input = input[:1000]
	}

	// Remove espaços extras
	input = strings.TrimSpace(input)
	input = regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")

	return input
}

// ValidateEnum valida se um valor está em uma lista de valores permitidos
func ValidateEnum(value string, allowedValues []string) bool {
	for _, allowed := range allowedValues {
		if value == allowed {
			return true
		}
	}
	return false
}

// ValidateDateFormat valida formato de data (YYYY-MM-DD)
func ValidateDateFormat(date string) bool {
	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	return dateRegex.MatchString(date)
}
