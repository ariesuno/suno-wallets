package ops

import (
	"context"
)

// Comentários em pt-BR: service para calcular snapshot de reconciliação por CPF

type ReconSummary struct {
	CPFMasked  string                   `json:"cpfMasked"`
	Period     map[string]string        `json:"period"`
	DataSource string                   `json:"dataSourceMode"`
	B3         map[string]interface{}   `json:"b3"`
	Incons     map[string]interface{}   `json:"inconsistencies"`
	SystemOps  map[string]interface{}   `json:"systemOps"`
	Overrides  map[string]interface{}   `json:"userOverrides"`
	Tickers    []map[string]interface{} `json:"tickers"`
	Notes      []string                 `json:"notes"`
}

type ReconRepository interface {
	LoadSummary(ctx context.Context, tenantID string, cpf string) (*ReconSummary, error)
}

type ReconSummaryService struct{ repo ReconRepository }

func NewReconSummaryService(repo ReconRepository) *ReconSummaryService {
	return &ReconSummaryService{repo: repo}
}

func (s *ReconSummaryService) Get(ctx context.Context, tenantID string, cpf string) (*ReconSummary, error) {
	return s.repo.LoadSummary(ctx, tenantID, cpf)
}
