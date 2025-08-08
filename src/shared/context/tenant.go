package contextx

import (
	"context"
)

// Comentários em pt-BR: helpers para armazenar/recuperar tenant_id do contexto
type tenantKeyType string

const tenantKey tenantKeyType = "tenant_id"

func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

func TenantIDFromContext(ctx context.Context) string {
	if v := ctx.Value(tenantKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
