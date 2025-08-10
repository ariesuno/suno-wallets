package reports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Comentários em pt-BR: Service de relatórios (somente leitura), orquestra repositório e validações

type Repository interface {
	RawDateRange(ctx context.Context, tenantID uuid.UUID, cpf string) (from, to *time.Time, err error)
	Summary(ctx context.Context, tenantID uuid.UUID, cpf string, from, to time.Time) (*SummaryOut, error)
	Tickers(ctx context.Context, tenantID uuid.UUID, cpf string, from, to time.Time, limit, offset int) ([]TickerRow, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

type SummaryOut struct {
	CPF                    string `json:"cpf"`
	From                   string `json:"from"`
	To                     string `json:"to"`
	MonthsWithTransactions int    `json:"monthsWithTransactions"`
	TotalTransactions      int    `json:"totalTransactions"`
	TickersCount           int    `json:"tickersCount"`
	PositionsCount         int    `json:"positionsCount"`
	GrossValueBRLSum       string `json:"grossValueBRLSum"`
	Notes                  string `json:"notes"`
}

type TickerRow struct {
	Ticker        string `json:"ticker"`
	QuantityTotal string `json:"quantityTotal"`
}

func (s *Service) GetRawDateRange(ctx context.Context, tenantID uuid.UUID, cpf string) (map[string]any, error) {
	from, to, err := s.repo.RawDateRange(ctx, tenantID, cpf)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"cpf": cpf, "dataTypes": []string{"transactions", "positions"}, "assetTypes": []string{"equity"}}
	if from != nil {
		out["from"] = from.Format("2006-01-02")
	}
	if to != nil {
		out["to"] = to.Format("2006-01-02")
	}
	return out, nil
}

func (s *Service) GetSummary(ctx context.Context, tenantID uuid.UUID, cpf string, from, to time.Time) (*SummaryOut, error) {
	sum, err := s.repo.Summary(ctx, tenantID, cpf, from, to)
	if err != nil {
		return nil, err
	}
	sum.CPF = cpf
	sum.From = from.Format("2006-01-02")
	sum.To = to.Format("2006-01-02")
	if sum.Notes == "" {
		sum.Notes = "Somatórios apenas se presentes no payload; sem PM/P&L."
	}
	return sum, nil
}

func (s *Service) GetTickers(ctx context.Context, tenantID uuid.UUID, cpf string, from, to time.Time, limit, offset int) (map[string]any, error) {
	rows, err := s.repo.Tickers(ctx, tenantID, cpf, from, to, limit, offset)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cpf":     cpf,
		"from":    from.Format("2006-01-02"),
		"to":      to.Format("2006-01-02"),
		"tickers": rows,
	}, nil
}
