package normalize

import (
	"context"
	"time"

	normrepo "suno-wallets/src/infrastructure/b3/normalize"
	"suno-wallets/src/infrastructure/b3/persistence"

	"github.com/google/uuid"
)

// Comentários em pt-BR: serviço de normalização (orquestra parsers e persistência)

type Repository interface {
	UpsertTransactions(ctx context.Context, items []normrepo.NormalizedTransaction) error
	UpsertPositions(ctx context.Context, items []normrepo.NormalizedPosition) error
	SelectPendingRaw(ctx context.Context, tenantID uuid.UUID, cpf, dataType, assetType string, start, end time.Time, force bool) ([]persistence.RawRecord, error)
	MarkRawNormalized(ctx context.Context, rawID uuid.UUID, count int) error
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

type RunParams struct {
	TenantID  uuid.UUID
	CPF       string
	DataType  string // transactions | positions
	AssetType string // equity
	Start     time.Time
	End       time.Time
	Force     bool
	DryRun    bool
}

type Summary struct{ Inserted, Updated, Skipped, Errors, RawProcessed int }

func (s *Service) Run(ctx context.Context, p RunParams) (*Summary, error) {
	sum := &Summary{}
	raws, err := s.repo.SelectPendingRaw(ctx, p.TenantID, p.CPF, p.DataType, p.AssetType, p.Start, p.End, p.Force)
	if err != nil {
		return nil, err
	}
	sum.RawProcessed = len(raws)
	for _, r := range raws {
		var inserted int
		switch p.DataType {
		case "transactions":
			items, _ := normrepo.NormalizeTransactions(p.TenantID, p.CPF, p.AssetType, r.ID, r.PayloadJSON)
			inserted = len(items)
			if !p.DryRun {
				if err := s.repo.UpsertTransactions(ctx, items); err != nil {
					sum.Errors++
					continue
				}
			}
		case "positions":
			items, _ := normrepo.NormalizePositions(p.TenantID, p.CPF, p.AssetType, r.ID, r.PayloadJSON)
			inserted = len(items)
			if !p.DryRun {
				if err := s.repo.UpsertPositions(ctx, items); err != nil {
					sum.Errors++
					continue
				}
			}
		}
		sum.Inserted += inserted
		if !p.DryRun {
			_ = s.repo.MarkRawNormalized(ctx, r.ID, inserted)
		}
	}
	return sum, nil
}
