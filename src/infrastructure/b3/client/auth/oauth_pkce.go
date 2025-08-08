package auth

import "context"

// PKCEProvider permite injetar/renovar tokens PKCE (ALF)
type PKCEProvider interface {
	// Exchange e Refresh são stubs a serem conectados ao consentimento externo
	Exchange(ctx context.Context, code string) (accessToken string, refreshToken string, err error)
	Refresh(ctx context.Context, refreshToken string) (newAccessToken string, newRefreshToken string, err error)
}
