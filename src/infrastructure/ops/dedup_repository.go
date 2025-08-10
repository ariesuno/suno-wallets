package ops

import (
	"context"
	"encoding/json"
	appops "suno-wallets/src/application/ops"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Comentários em pt-BR: repository de dedup candidates e resoluções

type DedupRepo struct{ db *gorm.DB }

func NewDedupRepo(db *gorm.DB) *DedupRepo { return &DedupRepo{db: db} }

func (r *DedupRepo) GetPolicy(ctx context.Context, tenantID uuid.UUID, cpf string) (*appops.DedupPolicy, error) {
    type row struct {
        PreferSource           string
        AutoMergeThreshold     float64
        AlertThreshold         float64
        DateToleranceDays      int
        QuantityToleranceRatio float64
        GrossToleranceRatio    float64
    }
    var rw row
    if err := r.db.WithContext(ctx).Raw(`SELECT prefer_source, auto_merge_threshold, alert_threshold, date_tolerance_days, quantity_tolerance_ratio, gross_tolerance_ratio FROM dedup_policies WHERE tenant_id = ? AND cpf = ? LIMIT 1`, tenantID, cpf).Scan(&rw).Error; err != nil {
        return nil, nil
    }
    if rw.PreferSource == "" {
        return nil, nil
    }
    return &appops.DedupPolicy{PreferSource: rw.PreferSource, AutoMergeThreshold: rw.AutoMergeThreshold, AlertThreshold: rw.AlertThreshold, DateToleranceDays: rw.DateToleranceDays, QuantityToleranceRatio: rw.QuantityToleranceRatio, GrossToleranceRatio: rw.GrossToleranceRatio}, nil
}

func (r *DedupRepo) ListLedgerOps(ctx context.Context, tenantID uuid.UUID, cpf string, source string, since, to *time.Time, tickers []string) ([]appops.LedgerOp, error) {
	qb := r.db.WithContext(ctx).Table("b3_operations_ledger").Select("id, ticker, operation_date, operation_type, quantity, unit_price").Where("tenant_id = ? AND cpf = ? AND source = ? AND is_active = true", tenantID, cpf, source)
	var out []appops.LedgerOp
	if err := qb.Scan(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *DedupRepo) UpsertCandidate(ctx context.Context, c appops.DedupCandidate) (bool, error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	js, _ := json.Marshal(c.Rationale)
	res := r.db.WithContext(ctx).Exec(`
      INSERT INTO dedup_candidates (id, tenant_id, cpf, primary_operation_id, candidate_operation_id, score, status, rationale, pair_key, dedupe_key)
      VALUES (?,?,?,?,?,?,?,?,?,?)
      ON CONFLICT (tenant_id, cpf, primary_operation_id, candidate_operation_id) DO UPDATE SET score = EXCLUDED.score, rationale = EXCLUDED.rationale
    `, c.ID, c.TenantID, c.CPF, c.PrimaryOperationID, c.CandidateOperationID, c.Score, c.Status, string(js), c.PairKey, c.DedupeKey)
	return res.RowsAffected > 0, res.Error
}

func (r *DedupRepo) ListCandidates(ctx context.Context, tenantID uuid.UUID, cpf string, status string, minScore, maxScore *float64, tickers []string, page, pageSize int) ([]appops.DedupCandidate, error) {
	qb := r.db.WithContext(ctx).Table("dedup_candidates").Select("id, tenant_id, cpf, primary_operation_id, candidate_operation_id, score, status, pair_key, dedupe_key").Where("tenant_id = ? AND cpf = ?", tenantID, cpf)
	if status != "" {
		qb = qb.Where("status = ?", status)
	}
	var out []appops.DedupCandidate
	if err := qb.Order("score DESC").Limit(pageSize).Offset((page - 1) * pageSize).Scan(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *DedupRepo) ResolveMerge(ctx context.Context, tenantID uuid.UUID, candidateIDs []uuid.UUID, prefer string) error {
    // implementar merge: escolher operação preferida e inativar a outra no ledger
    // Nota: simplificado aqui; detalhamento de supersedes pode ser expandido
    if err := r.db.WithContext(ctx).Exec(`UPDATE dedup_candidates SET status = 'CONFIRMED_MERGE' WHERE tenant_id = ? AND id IN ?`, tenantID, candidateIDs).Error; err != nil { return err }
    return nil
}

func (r *DedupRepo) ResolveOverride(ctx context.Context, tenantID uuid.UUID, candidateIDs []uuid.UUID) error {
    return r.db.WithContext(ctx).Exec(`UPDATE dedup_candidates SET status = 'OVERRIDDEN' WHERE tenant_id = ? AND id IN ?`, tenantID, candidateIDs).Error
}

func (r *DedupRepo) ResolveIgnore(ctx context.Context, tenantID uuid.UUID, candidateIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`UPDATE dedup_candidates SET status = 'IGNORED' WHERE tenant_id = ? AND id IN ?`, tenantID, candidateIDs).Error
}
