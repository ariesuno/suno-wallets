package ops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"os"
	"strconv"
	"time"

	obs "suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"

	"github.com/google/uuid"
)

// Comentários em pt-BR: serviço de deduplicação B3_RAW × USER_MANUAL

type DedupPolicy struct {
	PreferSource           string
	AutoMergeThreshold     float64
	AlertThreshold         float64
	DateToleranceDays      int
	QuantityToleranceRatio float64
	GrossToleranceRatio    float64
}

type LedgerOp struct {
	ID            uuid.UUID
	Ticker        string
	OperationDate time.Time
	OperationType string
	Quantity      float64
	UnitPrice     *float64
	Gross         *float64
	BrokerCode    *string
	OrderID       *string
}

type DedupCandidate struct {
	ID                   uuid.UUID
	TenantID             uuid.UUID
	CPF                  string
	PrimaryOperationID   uuid.UUID
	CandidateOperationID uuid.UUID
	Score                float64
	Status               string
	Rationale            map[string]interface{}
	PairKey              string
	DedupeKey            string
}

type DedupRepository interface {
	GetPolicy(ctx context.Context, tenantID uuid.UUID, cpf string) (*DedupPolicy, error)
	ListLedgerOps(ctx context.Context, tenantID uuid.UUID, cpf string, source string, since, to *time.Time, tickers []string) ([]LedgerOp, error)
	UpsertCandidate(ctx context.Context, c DedupCandidate) (bool, error)
	ListCandidates(ctx context.Context, tenantID uuid.UUID, cpf string, status string, minScore, maxScore *float64, tickers []string, page, pageSize int) ([]DedupCandidate, error)
	ResolveMerge(ctx context.Context, tenantID uuid.UUID, candidateIDs []uuid.UUID, prefer string) error
	ResolveOverride(ctx context.Context, tenantID uuid.UUID, candidateIDs []uuid.UUID) error
	ResolveIgnore(ctx context.Context, tenantID uuid.UUID, candidateIDs []uuid.UUID) error
}

type ScanRequest struct {
	CPF      string
	Since    *time.Time
	To       *time.Time
	Tickers  []string
	ScanMode string // ON_WRITE|BATCH
	DryRun   bool
}

type ResolveRequest struct {
	Action       string // MERGE|OVERRIDE|IGNORE
	CandidateIDs []uuid.UUID
	Prefer       string // opcional: B3_RAW|USER_MANUAL
}

type DedupService struct{ repo DedupRepository }

func (s *DedupService) ListCandidates(ctx context.Context, tenantID uuid.UUID, cpf, status string, page, pageSize int) ([]DedupCandidate, error) {
	return s.repo.ListCandidates(ctx, tenantID, cpf, status, nil, nil, nil, page, pageSize)
}

func NewDedupService(repo DedupRepository) *DedupService { return &DedupService{repo: repo} }

func (s *DedupService) Scan(ctx context.Context, tenantID uuid.UUID, req ScanRequest) (int, error) {
	started := time.Now()
	log := helpers.GetLoggerWithFields(map[string]interface{}{
		"service":  "dedup_scan",
		"tenantId": tenantID.String(),
		"cpf":      maskCPF(req.CPF),
		"dryRun":   req.DryRun,
		"mode":     req.ScanMode,
	})
	policy := s.loadPolicyDefaults()
	if p, err := s.repo.GetPolicy(ctx, tenantID, req.CPF); err == nil && p != nil {
		policy = *p
	}

	b3Ops, err := s.repo.ListLedgerOps(ctx, tenantID, req.CPF, "B3_RAW", req.Since, req.To, req.Tickers)
	if err != nil {
		obs.ObserveDedupScan("error", started)
		return 0, err
	}
	manOps, err := s.repo.ListLedgerOps(ctx, tenantID, req.CPF, "USER_MANUAL", req.Since, req.To, req.Tickers)
	if err != nil {
		obs.ObserveDedupScan("error", started)
		return 0, err
	}

	created := 0
	for _, b3 := range b3Ops {
		for _, mo := range manOps {
			if b3.Ticker != mo.Ticker {
				continue
			}
			if !withinDays(b3.OperationDate, mo.OperationDate, policy.DateToleranceDays) {
				continue
			}
			score, rationale := s.scorePair(b3, mo, policy)
			if score < policy.AlertThreshold {
				continue
			}
			cand := DedupCandidate{
				TenantID:             tenantID,
				CPF:                  req.CPF,
				PrimaryOperationID:   b3.ID,
				CandidateOperationID: mo.ID,
				Score:                score,
				Status:               "OPEN",
				Rationale:            rationale,
			}
			cand.PairKey = hashJoin(tenantID.String(), req.CPF, b3.ID.String(), mo.ID.String())
			cand.DedupeKey = hashJoin(b3.Ticker, b3.OperationType, b3.OperationDate.Format("2006-01-02"), f2s(b3.Quantity), f2s(zeroIfNil(b3.Gross)), mo.OperationType, f2s(mo.Quantity), f2s(zeroIfNil(mo.Gross)))
			if !req.DryRun {
				if ok, err := s.repo.UpsertCandidate(ctx, cand); err == nil && ok {
					created++
				} else if err != nil {
					obs.ObserveDedupScan("error", started)
					return created, err
				}
				if score >= policy.AutoMergeThreshold {
					_ = s.repo.ResolveMerge(ctx, tenantID, []uuid.UUID{cand.ID}, policy.PreferSource)
				}
			}
		}
	}
	obs.IncDedupCandidates("OPEN", created)
	obs.ObserveDedupScan("success", started)
	log.Info("dedup_scan_finished", map[string]interface{}{"created": created, "durationMs": time.Since(started).Milliseconds()})
	return created, nil
}

func maskCPF(cpf string) string {
	if len(cpf) < 3 {
		return "***"
	}
	return "***" + cpf[len(cpf)-3:]
}

func (s *DedupService) Resolve(ctx context.Context, tenantID uuid.UUID, req ResolveRequest) error {
	switch req.Action {
	case "MERGE":
		prefer := req.Prefer
		if prefer == "" {
			prefer = "B3_RAW"
		}
		if err := s.repo.ResolveMerge(ctx, tenantID, req.CandidateIDs, prefer); err != nil {
			return err
		}
		obs.IncDedupResolutions("MERGE", len(req.CandidateIDs))
	case "OVERRIDE":
		if err := s.repo.ResolveOverride(ctx, tenantID, req.CandidateIDs); err != nil {
			return err
		}
		obs.IncDedupResolutions("OVERRIDE", len(req.CandidateIDs))
	case "IGNORE":
		if err := s.repo.ResolveIgnore(ctx, tenantID, req.CandidateIDs); err != nil {
			return err
		}
		obs.IncDedupResolutions("IGNORE", len(req.CandidateIDs))
	default:
		return nil
	}
	return nil
}

// --- auxiliares ---

func (s *DedupService) loadPolicyDefaults() DedupPolicy {
	return DedupPolicy{
		PreferSource:           envOr("B3_RAW", "DEDUP_PREFER_SOURCE"),
		AutoMergeThreshold:     envOrFloat(0.92, "DEDUP_AUTO_MERGE_THRESHOLD"),
		AlertThreshold:         envOrFloat(0.70, "DEDUP_ALERT_THRESHOLD"),
		DateToleranceDays:      int(envOrFloat(2, "DEDUP_DATE_TOLERANCE_DAYS")),
		QuantityToleranceRatio: envOrFloat(0.005, "DEDUP_QTY_TOLERANCE_RATIO"),
		GrossToleranceRatio:    envOrFloat(0.005, "DEDUP_GROSS_TOLERANCE_RATIO"),
	}
}

func (s *DedupService) scorePair(b3 LedgerOp, mo LedgerOp, p DedupPolicy) (float64, map[string]interface{}) {
	// features
	qtyDelta := math.Abs(mo.Quantity-b3.Quantity) / maxf(1e-9, b3.Quantity)
	qtyMatch := clamp01(1 - qtyDelta/p.QuantityToleranceRatio)
	grossB3 := zeroIfNil(b3.Gross)
	grossMo := zeroIfNil(mo.Gross)
	grossBase := maxf(1e-9, grossB3)
	grossMatch := clamp01(1 - math.Abs(grossMo-grossB3)/grossBase/p.GrossToleranceRatio)
	sideMatch := 0.0
	if mo.OperationType == b3.OperationType {
		sideMatch = 1.0
	}
	// extras: broker/order simples (placeholder: 0)
	extras := 0.0
	rationale := map[string]interface{}{"qtyDelta": qtyDelta, "grossDelta": math.Abs(grossMo - grossB3), "side": sideMatch, "weights": map[string]float64{"quantity": 0.45, "gross": 0.35, "side": 0.15, "extras": 0.05}}
	// pesos
	score := 0.45*qtyMatch + 0.35*grossMatch + 0.15*sideMatch + 0.05*extras
	rationale["score"] = score
	return score, rationale
}

func withinDays(a, b time.Time, days int) bool {
	if days <= 0 {
		days = 0
	}
	d := a.Sub(b)
	if d < 0 {
		d = -d
	}
	return d.Hours() <= float64(days*24)
}

func hashJoin(parts ...string) string {
	h := sha256.Sum256([]byte(stringsJoin(parts, "|")))
	return hex.EncodeToString(h[:])
}

func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}

func f2s(f float64) string      { return fmtFloat(f) }
func fmtFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
func zeroIfNil(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func envOr(def, key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func envOrFloat(def float64, key string) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
