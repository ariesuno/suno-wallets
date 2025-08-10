package reconciliation_test

import (
	"context"
	"testing"
	"time"

	apprecon "suno-wallets/src/application/b3/reconciliation"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	scans map[string]int
}

func (f *fakeRepo) ScanOpeningBalanceMissing(ctx context.Context, tenantID uuid.UUID, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]apprecon.Finding, error) {
	n := f.scans["OPENING_BALANCE_MISSING"]
	out := make([]apprecon.Finding, n)
	for i := 0; i < n; i++ {
		out[i] = apprecon.Finding{Ticker: "ABC"}
	}
	return out, nil
}
func (f *fakeRepo) ScanSellWithoutBuy(ctx context.Context, tenantID uuid.UUID, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]apprecon.Finding, error) {
	n := f.scans["SELL_WITHOUT_BUY"]
	out := make([]apprecon.Finding, n)
	for i := 0; i < n; i++ {
		out[i] = apprecon.Finding{Ticker: "XYZ"}
	}
	return out, nil
}
func (f *fakeRepo) ScanPositionTxDivergence(ctx context.Context, tenantID uuid.UUID, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]apprecon.Finding, error) {
	n := f.scans["POSITION_TX_DIVERGENCE"]
	out := make([]apprecon.Finding, n)
	for i := 0; i < n; i++ {
		out[i] = apprecon.Finding{Ticker: "DEF"}
	}
	return out, nil
}
func (f *fakeRepo) UpsertFindings(ctx context.Context, tenantID uuid.UUID, cpf string, findings []apprecon.Finding) error {
	return nil
}
func (f *fakeRepo) ListInconsistencies(ctx context.Context, tenantID uuid.UUID, cpf, status, typ, ticker string, from, to *time.Time, page, pageSize int) ([]apprecon.Inconsistency, error) {
	return nil, nil
}
func (f *fakeRepo) GetInconsistency(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*apprecon.Inconsistency, error) {
	return nil, nil
}

func TestRecon_Scan_Totals(t *testing.T) {
	repo := &fakeRepo{scans: map[string]int{"OPENING_BALANCE_MISSING": 2, "SELL_WITHOUT_BUY": 1, "POSITION_TX_DIVERGENCE": 3}}
	svc := apprecon.NewService(repo)
	tenant := uuid.New()
	totals, err := svc.Scan(context.Background(), tenant, apprecon.ScanRequest{CPF: "12345678901", DryRun: true})
	require.NoError(t, err)
	require.Equal(t, 2, totals["OPENING_BALANCE_MISSING"])
	require.Equal(t, 1, totals["SELL_WITHOUT_BUY"])
	require.Equal(t, 3, totals["POSITION_TX_DIVERGENCE"])
}
