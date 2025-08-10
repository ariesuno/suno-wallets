package sync

import (
	"context"
	"time"

	"strings"
	ingest "suno-wallets/src/application/b3/ingest"
	syncrepo "suno-wallets/src/infrastructure/b3/sync"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"

	"github.com/google/uuid"
)

// Comentários em pt-BR: serviço de sincronismo diário (incremental)

type IngestService interface {
	Ingest(ctx context.Context, p ingest.IngestParams) (*ingest.Summary, error)
}

type Repository interface {
	GetActiveByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]syncrepo.SyncState, error)
	GetByTenantCPF(ctx context.Context, tenantID uuid.UUID, cpf string) (*syncrepo.SyncState, error)
	Upsert(ctx context.Context, st *syncrepo.SyncState) error
	MarkResult(ctx context.Context, id uuid.UUID, success bool, needsReprocess bool, lastResult string, lastError *string, txSyncAt, posSyncAt *time.Time) error
}

type Service struct {
	repo   Repository
	ingest IngestService
}

func NewService(repo Repository, ingest IngestService) *Service {
	return &Service{repo: repo, ingest: ingest}
}

// Repo expõe o repositório para endpoints de consulta (status)
func (s *Service) Repo() Repository { return s.repo }

type RunScope string

type RunParams struct {
	TenantID   uuid.UUID
	Scope      RunScope // single|tenant
	CPF        string   // quando single
	DataTypes  []string // transactions, positions
	AssetTypes []string // e.g., equity
	Force      bool
	DryRun     bool
	Limit      int
}

type RunSummary struct {
	Clients int
	Success int
	Failed  int
	NewRAW  int
}

func (s *Service) Run(ctx context.Context, p RunParams) (*RunSummary, error) {
	sum := &RunSummary{}
	start := time.Now()
	var targets []syncrepo.SyncState
	if p.Scope == "single" {
		st, err := s.repo.GetByTenantCPF(ctx, p.TenantID, p.CPF)
		if err != nil {
			return nil, err
		}
		if st != nil {
			targets = append(targets, *st)
		}
	} else {
		lst, err := s.repo.GetActiveByTenant(ctx, p.TenantID, p.Limit)
		if err != nil {
			return nil, err
		}
		targets = lst
	}
	sum.Clients = len(targets)

	now := time.Now()
	for _, t := range targets {
		var txSyncAt, posSyncAt *time.Time
		needsReprocess := false
		ok := true
		var lastErr *string

		for _, dt := range p.DataTypes {
			switch dt {
			case "transactions":
				start := dayStart(nextTimeOrEpoch(t.LastTxSyncAt))
				end := now
				if start.Before(end) {
					res, err := s.ingest.Ingest(ctx, ingest.IngestParams{
						TenantID:  p.TenantID.String(),
						CPF:       t.CPF,
						DataType:  "transactions",
						AssetType: firstOrDefault(p.AssetTypes, "equity"),
						Start:     start.Format("2006-01-02"),
						End:       end.Format("2006-01-02"),
						FetchAll:  true,
						Force:     p.Force,
						DryRun:    p.DryRun,
					})
					if err != nil {
						ok = false
						se := err.Error()
						lastErr = &se
					} else {
						if res != nil && res.Saved > 0 {
							sum.NewRAW += res.Saved
							needsReprocess = true
						}
						txSyncAt = &now
					}
				}
			case "positions":
				start := dayStart(nextTimeOrEpoch(t.LastPosSyncAt))
				end := now
				if start.Before(end) {
					res, err := s.ingest.Ingest(ctx, ingest.IngestParams{
						TenantID:  p.TenantID.String(),
						CPF:       t.CPF,
						DataType:  "positions",
						AssetType: firstOrDefault(p.AssetTypes, "equity"),
						Start:     start.Format("2006-01-02"),
						End:       end.Format("2006-01-02"),
						FetchAll:  true,
						Force:     p.Force,
						DryRun:    p.DryRun,
					})
					if err != nil {
						ok = false
						se := err.Error()
						lastErr = &se
					} else {
						if res != nil && res.Saved > 0 {
							sum.NewRAW += res.Saved
							needsReprocess = true
						}
						posSyncAt = &now
					}
				}
			}
		}
		if ok {
			sum.Success++
		} else {
			sum.Failed++
		}
		_ = s.repo.MarkResult(ctx, t.ID, ok, needsReprocess, tern(ok, "OK", "ERROR"), lastErr, txSyncAt, posSyncAt)
		helpers.LogInfo("B3 daily sync processed", map[string]interface{}{
			"tenant_id": p.TenantID.String(), "cpf_masked": maskCPFLocal(t.CPF),
			"success": ok, "needs_reprocess": needsReprocess,
		})
	}
	observability.ObserveSync(sum.Clients, sum.Success, sum.Failed, sum.NewRAW, start)
	return sum, nil
}

func nextTimeOrEpoch(t *time.Time) time.Time {
	if t == nil {
		return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return t.Add(24 * time.Hour)
}
func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
func firstOrDefault(list []string, def string) string {
	if len(list) == 0 || list[0] == "" {
		return def
	}
	return list[0]
}
func tern[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

// máscara simples de CPF (apenas últimos 2 dígitos visíveis)
func maskCPFLocal(cpf string) string {
	n := len(cpf)
	if n <= 2 {
		return "**"
	}
	return strings.Repeat("*", n-2) + cpf[n-2:]
}
