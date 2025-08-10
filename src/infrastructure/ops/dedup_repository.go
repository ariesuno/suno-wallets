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
	type row struct {
		ID            string
		Ticker        string
		OperationDate string
		OperationType string
		Quantity      float64
		UnitPrice     *float64
	}
	qb := r.db.WithContext(ctx).Table("b3_operations_ledger").Select("id, ticker, operation_date, operation_type, quantity, unit_price").Where("tenant_id = ? AND cpf = ? AND source = ? AND is_active = true", tenantID, cpf, source)
	var rows []row
	if err := qb.Scan(&rows).Error; err != nil {
		return nil, err
	}
	var out []appops.LedgerOp
	for _, rw := range rows {
		d, _ := time.Parse("2006-01-02", rw.OperationDate)
		out = append(out, appops.LedgerOp{ID: uuid.MustParse(rw.ID), Ticker: rw.Ticker, OperationDate: d, OperationType: rw.OperationType, Quantity: rw.Quantity, UnitPrice: rw.UnitPrice})
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
	qb := r.db.WithContext(ctx).Table("dedup_candidates").Select("id, tenant_id, cpf, primary_operation_id, candidate_operation_id, score, status, pair_key, dedupe_key, rationale").Where("tenant_id = ? AND cpf = ?", tenantID, cpf)
	if status != "" {
		qb = qb.Where("status = ?", status)
	}
	type row struct {
		ID                   string
		TenantID             string
		CPF                  string
		PrimaryOperationID   string
		CandidateOperationID string
		Score                float64
		Status               string
		PairKey              string
		DedupeKey            string
		Rationale            string
	}
	var rows []row
	if err := qb.Order("score DESC").Limit(pageSize).Offset((page - 1) * pageSize).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]appops.DedupCandidate, 0, len(rows))
	for _, rw := range rows {
		var rationale map[string]interface{}
		_ = json.Unmarshal([]byte(rw.Rationale), &rationale)
		out = append(out, appops.DedupCandidate{
			ID: uuid.MustParse(rw.ID), TenantID: uuid.MustParse(rw.TenantID), CPF: rw.CPF,
			PrimaryOperationID: uuid.MustParse(rw.PrimaryOperationID), CandidateOperationID: uuid.MustParse(rw.CandidateOperationID),
			Score: rw.Score, Status: rw.Status, PairKey: rw.PairKey, DedupeKey: rw.DedupeKey, Rationale: rationale,
		})
	}
	return out, nil
}

func (r *DedupRepo) ResolveMerge(ctx context.Context, tenantID uuid.UUID, candidateIDs []uuid.UUID, prefer string) error {
	tx := r.db.WithContext(ctx).Begin()
	defer func() { _ = tx.Rollback() }()
	type pair struct {
		ID        uuid.UUID
		Primary   uuid.UUID
		Candidate uuid.UUID
	}
	var pairs []pair
	if err := tx.Raw(`SELECT id, primary_operation_id, candidate_operation_id FROM dedup_candidates WHERE tenant_id = ? AND id IN ?`, tenantID, candidateIDs).Scan(&pairs).Error; err != nil {
		return err
	}
	type opRow struct {
		ID     uuid.UUID
		Source string
	}
	for _, p := range pairs {
		var ops []opRow
		if err := tx.Raw(`SELECT id, source FROM b3_operations_ledger WHERE tenant_id = ? AND id IN (?,?)`, tenantID, p.Primary, p.Candidate).Scan(&ops).Error; err != nil {
			return err
		}
		var preferred, suppressed uuid.UUID
		// default
		preferred = p.Primary
		suppressed = p.Candidate
		if len(ops) == 2 {
			// escolher conforme prefer
			if prefer == "USER_MANUAL" {
				// achar manual
				if ops[0].Source == "USER_MANUAL" {
					preferred, suppressed = ops[0].ID, ops[1].ID
				} else if ops[1].Source == "USER_MANUAL" {
					preferred, suppressed = ops[1].ID, ops[0].ID
				}
			} else if prefer == "B3_RAW" {
				if ops[0].Source == "B3_RAW" {
					preferred, suppressed = ops[0].ID, ops[1].ID
				} else if ops[1].Source == "B3_RAW" {
					preferred, suppressed = ops[1].ID, ops[0].ID
				}
			}
		}
		if err := tx.Exec(`UPDATE b3_operations_ledger SET is_active = false, superseded_by_operation_id = ?, updated_by = 'dedup:merge', updated_at = CURRENT_TIMESTAMP WHERE tenant_id = ? AND id = ?`, preferred, tenantID, suppressed).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE dedup_candidates SET status = 'CONFIRMED_MERGE', resolved_at = CURRENT_TIMESTAMP, resolved_by = 'dedup:merge' WHERE tenant_id = ? AND id = ?`, tenantID, p.ID).Error; err != nil {
			return err
		}
	}
	return tx.Commit().Error
}

func (r *DedupRepo) ResolveOverride(ctx context.Context, tenantID uuid.UUID, candidateIDs []uuid.UUID) error {
	tx := r.db.WithContext(ctx).Begin()
	defer func() { _ = tx.Rollback() }()
	type pair struct {
		ID        uuid.UUID
		Primary   uuid.UUID
		Candidate uuid.UUID
	}
	var pairs []pair
	if err := tx.Raw(`SELECT id, primary_operation_id, candidate_operation_id FROM dedup_candidates WHERE tenant_id = ? AND id IN ?`, tenantID, candidateIDs).Scan(&pairs).Error; err != nil {
		return err
	}
	type opRow struct {
		ID     uuid.UUID
		Source string
	}
	for _, p := range pairs {
		var ops []opRow
		if err := tx.Raw(`SELECT id, source FROM b3_operations_ledger WHERE tenant_id = ? AND id IN (?,?)`, tenantID, p.Primary, p.Candidate).Scan(&ops).Error; err != nil {
			return err
		}
		var preferred, suppressed uuid.UUID
		// preferência do OVERRIDE: USER_MANUAL prevalece quando disponível
		if len(ops) == 2 {
			if ops[0].Source == "USER_MANUAL" {
				preferred, suppressed = ops[0].ID, ops[1].ID
			} else if ops[1].Source == "USER_MANUAL" {
				preferred, suppressed = ops[1].ID, ops[0].ID
			} else {
				preferred, suppressed = p.Candidate, p.Primary
			}
		} else {
			preferred, suppressed = p.Candidate, p.Primary
		}
		if err := tx.Exec(`UPDATE b3_operations_ledger SET is_active = false, superseded_by_operation_id = ?, updated_by = 'dedup:override', updated_at = CURRENT_TIMESTAMP WHERE tenant_id = ? AND id = ?`, preferred, tenantID, suppressed).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE dedup_candidates SET status = 'OVERRIDDEN', resolved_at = CURRENT_TIMESTAMP, resolved_by = 'dedup:override' WHERE tenant_id = ? AND id = ?`, tenantID, p.ID).Error; err != nil {
			return err
		}
	}
	return tx.Commit().Error
}

func (r *DedupRepo) ResolveIgnore(ctx context.Context, tenantID uuid.UUID, candidateIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`UPDATE dedup_candidates SET status = 'IGNORED' WHERE tenant_id = ? AND id IN ?`, tenantID, candidateIDs).Error
}
