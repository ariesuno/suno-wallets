package clientpolicy

import (
	"context"
	dom "suno-wallets/src/domain/clientpolicy"
	obs "suno-wallets/src/infrastructure/observability"
)

// Comentários em pt-BR: serviço que lê cache → repo, grava e invalida cache e expõe helpers

type Service struct {
	repo  dom.Repository
	cache dom.Cache
}

func NewService(repo dom.Repository, cache dom.Cache) *Service {
	return &Service{repo: repo, cache: cache}
}

func (s *Service) GetMode(ctx context.Context, tenantID, cpf string) dom.Mode {
	if s.cache != nil {
		if pol, ok := s.cache.Get(tenantID, cpf); ok {
			obs.IncClientPolicyRead("cache", "hit")
			return pol.Mode
		}
		obs.IncClientPolicyRead("cache", "miss")
	}
	pol, err := s.repo.Get(tenantID, cpf)
	if err != nil || pol == nil {
		obs.IncClientPolicyRead("db", "empty")
		return dom.ModeHybrid
	}
	if s.cache != nil {
		s.cache.Set(tenantID, cpf, pol)
	}
	obs.IncClientPolicyRead("db", "ok")
	return pol.Mode
}

func (s *Service) Upsert(ctx context.Context, tenantID, cpf string, mode dom.Mode, reason, actor string) error {
	if err := s.repo.Upsert(tenantID, cpf, mode, reason, actor); err != nil {
		obs.IncClientPolicyWrite("error")
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(tenantID, cpf)
		obs.IncClientPolicyInvalidate()
	}
	obs.IncClientPolicyWrite("success")
	return nil
}

func (s *Service) ListAudit(ctx context.Context, tenantID, cpf string, limit int) ([]map[string]interface{}, error) {
	return s.repo.ListAudit(tenantID, cpf, limit)
}
