package valueobjects

import (
	"fmt"
	"regexp"
	"strings"
)

// Email representa um endereço de email válido
type Email struct {
	value string
}

// emailRegex expressão regular para validação de email
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// NewEmail cria um novo value object Email
func NewEmail(email string) (*Email, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		return nil, NewValidationError("email não pode ser vazio")
	}

	if len(email) > 254 {
		return nil, NewValidationError("email não pode ter mais de 254 caracteres")
	}

	if !emailRegex.MatchString(email) {
		return nil, NewValidationError("formato de email inválido")
	}

	// Validações adicionais de segurança
	if strings.Contains(email, "..") {
		return nil, NewValidationError("email não pode conter pontos consecutivos")
	}

	if strings.HasPrefix(email, ".") || strings.HasSuffix(email, ".") {
		return nil, NewValidationError("email não pode começar ou terminar com ponto")
	}

	return &Email{value: email}, nil
}

// MustNewEmail cria um Email ou entra em pânico se inválido (para testes)
func MustNewEmail(email string) *Email {
	e, err := NewEmail(email)
	if err != nil {
		panic(fmt.Sprintf("email inválido: %v", err))
	}
	return e
}

// Value retorna o valor string do email
func (e *Email) Value() string {
	return e.value
}

// String implementa fmt.Stringer
func (e *Email) String() string {
	return e.value
}

// Equals verifica se dois emails são iguais
func (e *Email) Equals(other *Email) bool {
	if other == nil {
		return false
	}
	return e.value == other.value
}

// Domain retorna o domínio do email
func (e *Email) Domain() string {
	parts := strings.Split(e.value, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// LocalPart retorna a parte local do email (antes do @)
func (e *Email) LocalPart() string {
	parts := strings.Split(e.value, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

// IsGmail verifica se é um email do Gmail
func (e *Email) IsGmail() bool {
	return e.Domain() == "gmail.com"
}

// IsOutlook verifica se é um email do Outlook/Hotmail
func (e *Email) IsOutlook() bool {
	domain := e.Domain()
	return domain == "outlook.com" || domain == "hotmail.com" || domain == "live.com"
}

// IsCorporate verifica se é um email corporativo (não de provedores públicos)
func (e *Email) IsCorporate() bool {
	publicDomains := map[string]bool{
		"gmail.com":      true,
		"yahoo.com":      true,
		"hotmail.com":    true,
		"outlook.com":    true,
		"live.com":       true,
		"icloud.com":     true,
		"aol.com":        true,
		"protonmail.com": true,
		"mail.com":       true,
		"yandex.com":     true,
	}

	return !publicDomains[e.Domain()]
}

// Validate valida o email (método adicional para compatibilidade)
func (e *Email) Validate() error {
	if e.value == "" {
		return NewValidationError("email está vazio")
	}

	// Re-executar validação para garantir consistência
	_, err := NewEmail(e.value)
	return err
}

// MarshalJSON implementa json.Marshaler
func (e *Email) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, e.value)), nil
}

// UnmarshalJSON implementa json.Unmarshaler
func (e *Email) UnmarshalJSON(data []byte) error {
	// Remove aspas da string JSON
	emailStr := strings.Trim(string(data), `"`)

	email, err := NewEmail(emailStr)
	if err != nil {
		return err
	}

	e.value = email.value
	return nil
}
