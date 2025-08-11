package e2e_test

import (
	"context"
	"testing"
	"time"

	appe2e "suno-wallets/src/application/b3/e2e"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fakeResetRepo implementa appe2e.ResetRepository para testes
type fakeResetRepo struct {
	lock bool
}

func (r *fakeResetRepo) TryAcquireLock(ctx context.Context, tenantID string, cpf string) (bool, error) {
	return r.lock, nil
}
func (r *fakeResetRepo) ReleaseLock(ctx context.Context, tenantID string, cpf string) error {
	return nil
}
func (r *fakeResetRepo) Reset(ctx context.Context, tenantID string, cpf string, mode string, archivedBy string) (*appe2e.ResetResult, error) {
	return &appe2e.ResetResult{}, nil
}

func TestOrchestrator_DryRun_ReturnsPlan(t *testing.T) {
	repo := &fakeResetRepo{lock: true}
	orch := appe2e.NewOrchestrator(repo, nil, nil)

	tenant := uuid.New()
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)

	out, err := orch.Run(context.Background(), appe2e.Params{
		TenantID:   tenant.String(),
		CPF:        "00000000000",
		AssetTypes: []string{"equity"},
		DataTypes:  []string{"transactions", "positions"},
		Start:      start,
		End:        end,
		DryRun:     true,
		Mode:       "archive",
	})

	require.NoError(t, err)
	// Janeiro, Fevereiro, Março => 3 janelas
	require.Equal(t, 3, out.Raw.MonthsProcessed)
	require.True(t, out.DryRun)
}

func TestOrchestrator_LockNotAcquired_ReturnsError(t *testing.T) {
	repo := &fakeResetRepo{lock: false}
	orch := appe2e.NewOrchestrator(repo, nil, nil)

	tenant := uuid.New()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	out, err := orch.Run(context.Background(), appe2e.Params{
		TenantID:   tenant.String(),
		CPF:        "00000000000",
		AssetTypes: []string{"equity"},
		DataTypes:  []string{"transactions"},
		Start:      start,
		End:        end,
		DryRun:     false,
		Mode:       "archive",
	})

	require.Error(t, err)
	require.Nil(t, out)
}
