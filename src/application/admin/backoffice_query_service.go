package admin

import (
	"context"

	"github.com/google/uuid"
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
	Search(ctx context.Context, tenantID, query string, limit int) ([]map[string]interface{}, error)
	ListActions(ctx context.Context, tenantID, cpf, action, status string, page, pageSize int) ([]map[string]interface{}, error)
	GetAction(ctx context.Context, tenantID string, id uuid.UUID) (map[string]interface{}, error)
	ExportLedger(ctx context.Context, tenantID, cpf string, excludeB3 bool, limit int) ([]map[string]interface{}, error)
	ExportLedgerStream(ctx context.Context, tenantID, cpf string, excludeB3 bool, limit int, callback func([]map[string]interface{}) error) error
}

type QueryService struct{ repo Repository }

func NewQueryService(r Repository) *QueryService { return &QueryService{repo: r} }

func (s *QueryService) Profile(ctx context.Context, tenantID, cpf string) (*Profile, error) {
	return s.repo.LoadProfile(ctx, tenantID, cpf)
}

func (s *QueryService) Search(ctx context.Context, tenantID, query string, limit int) ([]map[string]interface{}, error) {
	return s.repo.Search(ctx, tenantID, query, limit)
}

func (s *QueryService) ListActions(ctx context.Context, tenantID, cpf, action, status string, page, pageSize int) ([]map[string]interface{}, error) {
	return s.repo.ListActions(ctx, tenantID, cpf, action, status, page, pageSize)
}

func (s *QueryService) GetAction(ctx context.Context, tenantID string, id uuid.UUID) (map[string]interface{}, error) {
	return s.repo.GetAction(ctx, tenantID, id)
}

func (s *QueryService) ExportLedger(ctx context.Context, tenantID, cpf string, excludeB3 bool, limit int) ([]map[string]interface{}, error) {
	return s.repo.ExportLedger(ctx, tenantID, cpf, excludeB3, limit)
}

func (s *QueryService) ExportLedgerStream(ctx context.Context, tenantID, cpf string, excludeB3 bool, limit int, callback func([]map[string]interface{}) error) error {
	return s.repo.ExportLedgerStream(ctx, tenantID, cpf, excludeB3, limit, callback)
}
