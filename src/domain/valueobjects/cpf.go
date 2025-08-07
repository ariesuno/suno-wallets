package valueobjects

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// CPF representa um CPF (Cadastro de Pessoa Física) válido
type CPF struct {
	value string // CPF armazenado apenas com números
}

// cpfRegex expressão regular para limpar CPF
var cpfRegex = regexp.MustCompile(`[^\d]`)

// NewCPF cria um novo value object CPF
func NewCPF(cpf string) (*CPF, error) {
	// Remove caracteres não numéricos
	cleanCPF := cpfRegex.ReplaceAllString(cpf, "")

	if cleanCPF == "" {
		return nil, NewValidationError("CPF não pode ser vazio")
	}

	if len(cleanCPF) != 11 {
		return nil, NewValidationError("CPF deve ter 11 dígitos")
	}

	// Verifica se todos os dígitos são iguais (CPF inválido)
	if isAllSameDigit(cleanCPF) {
		return nil, NewValidationError("CPF com todos os dígitos iguais é inválido")
	}

	// Valida os dígitos verificadores
	if !isValidCPF(cleanCPF) {
		return nil, NewValidationError("CPF com dígitos verificadores inválidos")
	}

	return &CPF{value: cleanCPF}, nil
}

// MustNewCPF cria um CPF ou entra em pânico se inválido (para testes)
func MustNewCPF(cpf string) *CPF {
	c, err := NewCPF(cpf)
	if err != nil {
		panic(fmt.Sprintf("CPF inválido: %v", err))
	}
	return c
}

// Value retorna o valor string do CPF (apenas números)
func (c *CPF) Value() string {
	return c.value
}

// String implementa fmt.Stringer - retorna CPF formatado
func (c *CPF) String() string {
	return c.Formatted()
}

// Formatted retorna o CPF formatado (XXX.XXX.XXX-XX)
func (c *CPF) Formatted() string {
	if len(c.value) != 11 {
		return c.value
	}

	return fmt.Sprintf("%s.%s.%s-%s",
		c.value[0:3],
		c.value[3:6],
		c.value[6:9],
		c.value[9:11],
	)
}

// Equals verifica se dois CPFs são iguais
func (c *CPF) Equals(other *CPF) bool {
	if other == nil {
		return false
	}
	return c.value == other.value
}

// Validate valida o CPF (método adicional para compatibilidade)
func (c *CPF) Validate() error {
	if c.value == "" {
		return NewValidationError("CPF está vazio")
	}

	// Re-executar validação para garantir consistência
	_, err := NewCPF(c.value)
	return err
}

// GetRegion retorna a região fiscal do CPF baseada no primeiro dígito
func (c *CPF) GetRegion() string {
	if len(c.value) < 1 {
		return "Desconhecido"
	}

	switch c.value[0] {
	case '1':
		return "DF, GO, MS, MT, TO"
	case '2':
		return "AC, AM, AP, PA, RO, RR"
	case '3':
		return "CE, MA, PI"
	case '4':
		return "AL, PB, PE, RN"
	case '5':
		return "BA, SE"
	case '6':
		return "MG"
	case '7':
		return "ES, RJ"
	case '8':
		return "SP"
	case '9':
		return "PR, SC"
	case '0':
		return "RS"
	default:
		return "Desconhecido"
	}
}

// MarshalJSON implementa json.Marshaler
func (c *CPF) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, c.value)), nil
}

// UnmarshalJSON implementa json.Unmarshaler
func (c *CPF) UnmarshalJSON(data []byte) error {
	// Remove aspas da string JSON
	cpfStr := strings.Trim(string(data), `"`)

	cpf, err := NewCPF(cpfStr)
	if err != nil {
		return err
	}

	c.value = cpf.value
	return nil
}

// isAllSameDigit verifica se todos os dígitos são iguais
func isAllSameDigit(cpf string) bool {
	first := cpf[0]
	for _, digit := range cpf {
		if byte(digit) != first {
			return false
		}
	}
	return true
}

// isValidCPF valida os dígitos verificadores do CPF
func isValidCPF(cpf string) bool {
	// Converte string para slice de inteiros
	digits := make([]int, 11)
	for i, r := range cpf {
		digit, err := strconv.Atoi(string(r))
		if err != nil {
			return false
		}
		digits[i] = digit
	}

	// Calcula primeiro dígito verificador
	sum := 0
	for i := 0; i < 9; i++ {
		sum += digits[i] * (10 - i)
	}

	remainder := sum % 11
	firstVerifier := 0
	if remainder >= 2 {
		firstVerifier = 11 - remainder
	}

	if digits[9] != firstVerifier {
		return false
	}

	// Calcula segundo dígito verificador
	sum = 0
	for i := 0; i < 10; i++ {
		sum += digits[i] * (11 - i)
	}

	remainder = sum % 11
	secondVerifier := 0
	if remainder >= 2 {
		secondVerifier = 11 - remainder
	}

	return digits[10] == secondVerifier
}
