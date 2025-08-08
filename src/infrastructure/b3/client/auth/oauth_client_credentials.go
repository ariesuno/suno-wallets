package auth

import (
	"context"
	"time"
)

// TokenResponse representa uma resposta de token simplificada
type TokenResponse struct {
	AccessToken string
	ExpiresIn   int64 // em segundos
	ObtainedAt  time.Time
}

// ClientCredentialsProvider define interface para obter tokens via client credentials
type ClientCredentialsProvider interface {
	GetToken(ctx context.Context) (*TokenResponse, error)
}
