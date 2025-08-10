package admin

import (
	"context"
)

// Comentários em pt-BR: service para montar o profile 360 do cliente agregando dados

type Profile struct {
	CPFMasked      string                   `json:"cpfMasked"`
	DataSourceMode string                   `json:"dataSourceMode"`
	B3             map[string]interface{}   `json:"b3"`
	Reconciliation map[string]interface{}   `json:"reconciliation"`
	Performance    map[string]interface{}   `json:"performance"`
	LastActions    []map[string]interface{} `json:"lastActions"`
}

type Repository interface {
	LoadProfile(ctx context.Context, tenantID, cpf string) (*Profile, error)
}

type QueryService struct{ repo Repository }

func NewQueryService(r Repository) *QueryService { return &QueryService{repo: r} }

func (s *QueryService) Profile(ctx context.Context, tenantID, cpf string) (*Profile, error) {
	return s.repo.LoadProfile(ctx, tenantID, cpf)
}
