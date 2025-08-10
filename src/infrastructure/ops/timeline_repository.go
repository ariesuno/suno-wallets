package ops

import (
	"context"
	"strings"
	appops "suno-wallets/src/application/ops"
	"time"

	"gorm.io/gorm"
)

// Comentários em pt-BR: repositório de leitura da timeline com keyset pagination

type TimelineRepo struct{ db *gorm.DB }

func NewTimelineRepo(db *gorm.DB) *TimelineRepo { return &TimelineRepo{db: db} }

func (r *TimelineRepo) ListTimeline(ctx context.Context, tenantID string, f appops.TimelineFilters) ([]appops.TimelineItem, *string, error) {
	qb := r.db.WithContext(ctx).Table("b3_operations_ledger").Select(`
        id, cpf, ticker as original_ticker, ticker as canonical_ticker, asset_type, operation_date, operation_type,
        source, quantity, unit_price, currency, reason_code, price_confidence,
        generated_by_inconsistency_id, supersedes_operation_id, superseded_by_operation_id,
        is_active, created_at, updated_at
    `).Where("tenant_id = ? AND is_active = true", tenantID)
	if f.CPF != "" {
		qb = qb.Where("cpf = ?", f.CPF)
	}
	if len(f.Tickers) > 0 {
		qb = qb.Where("ticker IN ?", f.Tickers)
	}
	if len(f.Sources) > 0 {
		qb = qb.Where("source IN ?", f.Sources)
	}
	if len(f.AssetTypes) > 0 {
		qb = qb.Where("asset_type IN ?", f.AssetTypes)
	}
	if f.From != nil {
		qb = qb.Where("operation_date >= ?", f.From.Format("2006-01-02"))
	}
	if f.To != nil {
		qb = qb.Where("operation_date <= ?", f.To.Format("2006-01-02"))
	}
	// keyset
	if f.Cursor != nil && *f.Cursor != "" {
		d, id, _ := appops.DecodeCursor(*f.Cursor)
		qb = qb.Where("(operation_date < ? OR (operation_date = ? AND id < ?))", d.Format("2006-01-02"), d.Format("2006-01-02"), id)
	}
	qb = qb.Order("operation_date DESC, id DESC").Limit(f.PageSize)

	type row struct {
		ID                         string
		OriginalTicker             string
		CanonicalTicker            string
		AssetType                  string
		OperationDate              string
		OperationType              string
		Source                     string
		Quantity                   float64
		UnitPrice                  *float64
		Currency                   string
		ReasonCode                 *string
		PriceConfidence            string
		GeneratedByInconsistencyID *string
		SupersedesOperationID      *string
		SupersededByOperationID    *string
		IsActive                   bool
		CreatedAt                  string
		UpdatedAt                  string
	}
	var rows []row
	if err := qb.Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	items := make([]appops.TimelineItem, 0, len(rows))
	for _, rw := range rows {
		d, _ := time.Parse("2006-01-02", rw.OperationDate)
		it := appops.TimelineItem{
			ID:              rw.ID,
			CanonicalTicker: rw.CanonicalTicker,
			OriginalTicker:  rw.OriginalTicker,
			AssetType:       rw.AssetType,
			OperationDate:   d,
			OperationType:   rw.OperationType,
			Source:          rw.Source,
			Quantity:        rw.Quantity,
			UnitPrice:       rw.UnitPrice,
			Currency:        rw.Currency,
			ReasonCode:      rw.ReasonCode,
			PriceConfidence: rw.PriceConfidence,
		}
		if rw.GeneratedByInconsistencyID != nil {
			it.Links.InconsistencyID = rw.GeneratedByInconsistencyID
		}
		if rw.SupersedesOperationID != nil {
			it.Links.SupersedesOperationID = rw.SupersedesOperationID
		}
		if rw.SupersededByOperationID != nil {
			it.Links.SupersededByOperationID = rw.SupersededByOperationID
		}
		if rw.CreatedAt != "" {
			if dt, err := time.Parse("2006-01-02", rw.CreatedAt); err == nil {
				it.Meta.CreatedAt = dt
			}
		}
		if rw.UpdatedAt != "" {
			if dt, err := time.Parse("2006-01-02", rw.UpdatedAt); err == nil {
				it.Meta.UpdatedAt = dt
			}
		}
		items = append(items, it)
	}
	var next *string
	if len(items) == f.PageSize {
		last := items[len(items)-1]
		c := appops.EncodeCursor(last.OperationDate, last.ID)
		next = &c
	}
	// limpeza básica (canonicalize: placeholder — sem coluna canonical, devolve original)
	if !f.Canonicalize {
		for i := range items {
			items[i].CanonicalTicker = items[i].OriginalTicker
		}
	}
	_ = strings.Builder{}
	return items, next, nil
}
