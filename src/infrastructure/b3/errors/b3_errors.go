package errors

import "errors"

// Erros padronizados do cliente B3
var (
	ErrInvalidCPF       = errors.New("invalid cpf: must be 11 digits")
	ErrTLSConfig        = errors.New("failed to build mTLS configuration")
	ErrTokenAcquisition = errors.New("failed to acquire oauth2 token")
	ErrHTTPBuildRequest = errors.New("failed to build http request")
	ErrHTTPDoRequest    = errors.New("failed to execute http request")
)

// B3Error representa um erro retornado pela B3 com status HTTP
type B3Error struct {
	Status  int
	Message string
}

func (e *B3Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "b3 error"
}

// IsInternal indica se é um erro 5xx
func (e *B3Error) IsInternal() bool { return e.Status >= 500 }

// NewB3Error helper para construir erro com status e mensagem
func NewB3Error(status int, msg string) *B3Error { return &B3Error{Status: status, Message: msg} }
