package reports

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	repsvc "suno-wallets/src/application/b3/reports"
)

// Comentários em pt-BR: repositório de relatórios com SQLs otimizadas (PostgreSQL)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) RawDateRange(ctx context.Context, tenantID uuid.UUID, cpf string) (from, to *time.Time, err error) {
	// Compatível com SQLite (tests) e Postgres: escanear como string e parsear
	var fNS, tNS sql.NullString
	if err = r.db.WithContext(ctx).Raw(`
        SELECT MIN(period_start) AS from_date, MAX(period_end) AS to_date
        FROM b3_raw_data_client WHERE tenant_id = ? AND cpf = ?
    `, tenantID, cpf).Row().Scan(&fNS, &tNS); err != nil {
		return nil, nil, err
	}
	parse := func(ns sql.NullString) *time.Time {
		if !ns.Valid {
			return nil
		}
		if t, e := time.Parse("2006-01-02", ns.String); e == nil {
			return &t
		}
		return nil
	}
	return parse(fNS), parse(tNS), nil
}

func (r *Repository) Summary(ctx context.Context, tenantID uuid.UUID, cpf string, from, to time.Time) (*repsvc.SummaryOut, error) {
	// monthsWithTransactions: computa em Go para compatibilidade entre dialetos
	var dates []string
	if err := r.db.WithContext(ctx).Raw(`
        SELECT trade_date FROM b3_normalized_transactions
        WHERE tenant_id = ? AND cpf = ? AND trade_date BETWEEN ? AND ?`, tenantID, cpf, from, to).Scan(&dates).Error; err != nil {
		return nil, err
	}
	monthsSet := map[string]struct{}{}
	for _, s := range dates {
		// suporta formatos YYYY-MM-DD
		if len(s) >= 7 {
			monthsSet[s[:7]] = struct{}{}
		}
	}
	months := len(monthsSet)

	// total transactions & distinct tickers
	var tot, tickers int
	if err := r.db.WithContext(ctx).Raw(`
        SELECT COUNT(*) AS total_tx, COUNT(DISTINCT ticker) AS tickers_distinct
        FROM b3_normalized_transactions
        WHERE tenant_id = ? AND cpf = ? AND trade_date BETWEEN ? AND ?`, tenantID, cpf, from, to).Row().Scan(&tot, &tickers); err != nil {
		return nil, err
	}

	// positions count
	var pos int
	if err := r.db.WithContext(ctx).Raw(`
        SELECT COUNT(*) FROM b3_normalized_positions
        WHERE tenant_id = ? AND cpf = ? AND reference_date BETWEEN ? AND ?`, tenantID, cpf, from, to).Scan(&pos).Error; err != nil {
		return nil, err
	}

	// gross BRL sum (if exists) como texto
	var gross sql.NullString
	if err := r.db.WithContext(ctx).Raw(`
        SELECT COALESCE(SUM(position_value), 0) AS gross_brl_sum
        FROM b3_normalized_positions
        WHERE tenant_id = ? AND cpf = ? AND reference_date BETWEEN ? AND ? AND currency = 'BRL'`, tenantID, cpf, from, to).Row().Scan(&gross); err != nil {
		return nil, err
	}
	grossStr := "0.00"
	if gross.Valid {
		if f, e := strconv.ParseFloat(gross.String, 64); e == nil {
			grossStr = fmt.Sprintf("%.2f", f)
		} else {
			grossStr = gross.String
		}
	}

	return &repsvc.SummaryOut{
		MonthsWithTransactions: months,
		TotalTransactions:      tot,
		TickersCount:           tickers,
		PositionsCount:         pos,
		GrossValueBRLSum:       grossStr,
	}, nil
}

func (r *Repository) Tickers(ctx context.Context, tenantID uuid.UUID, cpf string, from, to time.Time, limit, offset int) ([]repsvc.TickerRow, error) {
	type row struct {
		Ticker        string
		QuantityTotal string
	}
	var rows []row
	if err := r.db.WithContext(ctx).Raw(`
        SELECT ticker, SUM(quantity) AS quantity_total
        FROM b3_normalized_positions
        WHERE tenant_id = ? AND cpf = ? AND reference_date BETWEEN ? AND ?
        GROUP BY ticker ORDER BY ticker
        LIMIT ? OFFSET ?`, tenantID, cpf, from, to, limit, offset).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]repsvc.TickerRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, repsvc.TickerRow{Ticker: r.Ticker, QuantityTotal: r.QuantityTotal})
	}
	return out, nil
}
