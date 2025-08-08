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
