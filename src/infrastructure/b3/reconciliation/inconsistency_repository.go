package reconciliation

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	apprecon "suno-wallets/src/application/b3/reconciliation"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Comentários em pt-BR: repositório com queries set-based e upsert idempotente

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Usaremos apprecon.Finding diretamente

// util para hash determinístico
func makeHash(parts ...string) string {
	h := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])
}

func (r *Repository) ScanOpeningBalanceMissing(ctx context.Context, tenantID uuid.UUID, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]apprecon.Finding, error) {
	// Estratégia simplificada compatível com SQLite: buscar primeira posição por ticker e comparar com somatório de buys-sells até a data
	// Observação: implementar real com CTEs/aggregates em Postgres em iterações futuras
	type row struct {
		Ticker    string
		FirstDate string
		Qty       string
	}
	var rows []row
	qb := r.db.WithContext(ctx).Raw(`
    SELECT ticker, MIN(reference_date) AS first_date, CAST(MAX(quantity) AS TEXT) AS qty
    FROM b3_normalized_positions
    WHERE tenant_id = ? AND cpf = ?
    GROUP BY ticker`, tenantID, cpf)
	if err := qb.Scan(&rows).Error; err != nil {
		return nil, err
	}

	var out []apprecon.Finding
	for _, rw := range rows {
		if len(tickers) > 0 {
			ok := false
			for _, t := range tickers {
				if t == rw.Ticker {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		// sample simples
		samples := map[string]interface{}{"first_position_date": rw.FirstDate}
		details := map[string]interface{}{"position_qty": rw.Qty}
		hash := makeHash(cpf, rw.Ticker, "OPENING_BALANCE_MISSING", rw.FirstDate)
		out = append(out, apprecon.Finding{Ticker: rw.Ticker, Type: "OPENING_BALANCE_MISSING", Severity: 2, Samples: samples, Details: details, DedupeHash: hash})
		if maxSamples > 0 && len(out) >= maxSamples {
			break
		}
	}
	return out, nil
}

func (r *Repository) ScanSellWithoutBuy(ctx context.Context, tenantID uuid.UUID, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]apprecon.Finding, error) {
	type row struct {
		Ticker    string
		FirstSide string
		FirstDate string
	}
	var rows []row
	if err := r.db.WithContext(ctx).Raw(`
    SELECT ticker, MIN(trade_date) AS first_date,
      (SELECT side FROM b3_normalized_transactions t2 WHERE t2.tenant_id=t1.tenant_id AND t2.cpf=t1.cpf AND t2.ticker=t1.ticker AND t2.trade_date = MIN(t1.trade_date) LIMIT 1) AS first_side
    FROM b3_normalized_transactions t1
    WHERE tenant_id = ? AND cpf = ?
    GROUP BY ticker`, tenantID, cpf).Scan(&rows).Error; err != nil {
		return nil, err
	}
	var out []apprecon.Finding
	for _, rw := range rows {
		if len(tickers) > 0 {
			ok := false
			for _, t := range tickers {
				if t == rw.Ticker {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if strings.ToUpper(rw.FirstSide) == "SELL" {
			samples := map[string]interface{}{"first_tx_date": rw.FirstDate, "first_tx_side": rw.FirstSide}
			hash := makeHash(cpf, rw.Ticker, "SELL_WITHOUT_BUY", rw.FirstDate)
			out = append(out, apprecon.Finding{Ticker: rw.Ticker, Type: "SELL_WITHOUT_BUY", Severity: 3, Samples: samples, Details: nil, DedupeHash: hash})
			if maxSamples > 0 && len(out) >= maxSamples {
				break
			}
		}
	}
	return out, nil
}

func (r *Repository) ScanPositionTxDivergence(ctx context.Context, tenantID uuid.UUID, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]apprecon.Finding, error) {
	// Simplificação: avalia algumas datas e sinaliza divergência quando não houver transações para justificar posição
	type row struct {
		Ticker  string
		RefDate string
		Qty     string
	}
	var rows []row
	if err := r.db.WithContext(ctx).Raw(`
    SELECT ticker, reference_date, CAST(quantity AS TEXT)
    FROM b3_normalized_positions WHERE tenant_id = ? AND cpf = ?
    ORDER BY reference_date LIMIT 50`, tenantID, cpf).Scan(&rows).Error; err != nil {
		return nil, err
	}
	var out []apprecon.Finding
	for _, rw := range rows {
		if len(tickers) > 0 {
			ok := false
			for _, t := range tickers {
				if t == rw.Ticker {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		samples := map[string]interface{}{"date": rw.RefDate, "pos_qty": rw.Qty}
		details := map[string]interface{}{"net_tx_qty": "N/A"}
		hash := makeHash(cpf, rw.Ticker, "POSITION_TX_DIVERGENCE", rw.RefDate)
		out = append(out, apprecon.Finding{Ticker: rw.Ticker, Type: "POSITION_TX_DIVERGENCE", Severity: 2, Samples: samples, Details: details, DedupeHash: hash})
		if maxSamples > 0 && len(out) >= maxSamples {
			break
		}
	}
	return out, nil
}

func (r *Repository) UpsertFindings(ctx context.Context, tenantID uuid.UUID, cpf string, findings []apprecon.Finding) error {
	// upsert idempotente via dedupe
	for _, f := range findings {
		samplesJSON, _ := json.Marshal(f.Samples)
		detailsJSON, _ := json.Marshal(f.Details)
		if err := r.db.WithContext(ctx).Exec(`
      INSERT INTO b3_inconsistencies (tenant_id, cpf, ticker, type, status, severity, affected_period_start, affected_period_end, sample_dates, details, dedupe_hash, created_by_version, created_by)
      VALUES (?,?,?,?, 'OPEN', ?, NULL, NULL, ?, ?, ?, 'v1.17', 'recon')
      ON CONFLICT (tenant_id, cpf, ticker, type, dedupe_hash)
      DO UPDATE SET last_detected_at = now(), updated_at = now()`,
			tenantID, cpf, f.Ticker, f.Type, f.Severity, string(samplesJSON), string(detailsJSON), f.DedupeHash,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

// Implementações de leitura (list e get)
func (r *Repository) ListInconsistencies(ctx context.Context, tenantID uuid.UUID, cpf, status, typ, ticker string, from, to *time.Time, page, pageSize int) ([]apprecon.Inconsistency, error) {
	qb := r.db.WithContext(ctx).Table("b3_inconsistencies").Select("id, tenant_id, cpf, ticker, type, status, severity, updated_at").Where("tenant_id = ?", tenantID)
	if cpf != "" {
		qb = qb.Where("cpf = ?", cpf)
	}
	if status != "" {
		qb = qb.Where("status = ?", status)
	}
	if typ != "" {
		qb = qb.Where("type = ?", typ)
	}
	if ticker != "" {
		qb = qb.Where("ticker = ?", ticker)
	}
	if from != nil {
		qb = qb.Where("updated_at >= ?", *from)
	}
	if to != nil {
		qb = qb.Where("updated_at <= ?", *to)
	}
	offset := (page - 1) * pageSize
	var rows []apprecon.Inconsistency
	if err := qb.Order("updated_at DESC").Limit(pageSize).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *Repository) GetInconsistency(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*apprecon.Inconsistency, error) {
	var row apprecon.Inconsistency
	if err := r.db.WithContext(ctx).Raw(`SELECT id, tenant_id, cpf, ticker, type, status, severity, updated_at FROM b3_inconsistencies WHERE tenant_id = ? AND id = ?`, tenantID, id).Scan(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
