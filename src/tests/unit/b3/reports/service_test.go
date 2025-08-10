package reports_test

import (
	"context"
	"testing"
	"time"

	svc "suno-wallets/src/application/b3/reports"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	from, to *time.Time
	sum      *svc.SummaryOut
	tickers  []svc.TickerRow
}

func (f *fakeRepo) RawDateRange(ctx context.Context, tenantID uuid.UUID, cpf string) (from, to *time.Time, err error) {
	return f.from, f.to, nil
}
func (f *fakeRepo) Summary(ctx context.Context, tenantID uuid.UUID, cpf string, from, to time.Time) (*svc.SummaryOut, error) {
	return f.sum, nil
}
func (f *fakeRepo) Tickers(ctx context.Context, tenantID uuid.UUID, cpf string, from, to time.Time, limit, offset int) ([]svc.TickerRow, error) {
	return f.tickers, nil
}

func TestService_GetRawDateRange(t *testing.T) {
	r := &fakeRepo{}
	s := svc.NewService(r)
	tenant := uuid.New()
	out, err := s.GetRawDateRange(context.Background(), tenant, "00000000000")
	require.NoError(t, err)
	require.Equal(t, "00000000000", out["cpf"])
}

func TestService_GetSummary(t *testing.T) {
	r := &fakeRepo{sum: &svc.SummaryOut{}}
	s := svc.NewService(r)
	tenant := uuid.New()
	from, _ := time.Parse("2006-01-02", "2024-01-01")
	to, _ := time.Parse("2006-01-02", "2024-12-31")
	out, err := s.GetSummary(context.Background(), tenant, "00000000000", from, to)
	require.NoError(t, err)
	require.Equal(t, "00000000000", out.CPF)
	require.Equal(t, "Somatórios apenas se presentes no payload; sem PM/P&L.", out.Notes)
}

func TestService_GetTickers(t *testing.T) {
	r := &fakeRepo{tickers: []svc.TickerRow{{Ticker: "ABCD3", QuantityTotal: "15"}}}
	s := svc.NewService(r)
	tenant := uuid.New()
	from, _ := time.Parse("2006-01-02", "2024-01-01")
	to, _ := time.Parse("2006-01-02", "2024-12-31")
	out, err := s.GetTickers(context.Background(), tenant, "00000000000", from, to, 10, 0)
	require.NoError(t, err)
	require.Equal(t, "00000000000", out["cpf"])
	arr := out["tickers"].([]svc.TickerRow)
	require.Len(t, arr, 1)
	require.Equal(t, "ABCD3", arr[0].Ticker)
}
