package normalize

import (
	"context"

	"suno-wallets/src/shared/helpers"
)

// Comentários em pt-BR: serviço de normalização LEGADO (mantido para compatibilidade)

type LegacyService struct {
	repo Repository
}

func NewLegacyService(repo Repository) *LegacyService {
	return &LegacyService{
		repo: repo,
	}
}

func (s *LegacyService) Run(ctx context.Context, p RunParams) (*Summary, error) {
	// Usar implementação DDD+SOLID sempre (delegando)
	helpers.LogInfo("legacy service delegating to DDD implementation", map[string]interface{}{
		"tenantID": p.TenantID,
		"cpf":      p.CPF[:3] + "*******",
		"dataType": p.DataType,
		"force":    p.Force,
	})

	// Delegar para serviço DDD
	appService := NewNormalizationService(s.repo)
	return appService.Run(ctx, p)
}
