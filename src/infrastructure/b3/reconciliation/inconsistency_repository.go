package reconciliation

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
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

// util para hash determinístico (sha256)
func makeHash(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])
}

func getEarliestDate() string {
	if v := os.Getenv("B3_API_EARLIEST_DATE"); v != "" {
		return v
	}
	return "2019-10-01"
}

func makeLockKey(tenantID string, cpf string) int64 {
	b := sha256.Sum256([]byte(tenantID + "|" + cpf))
	// usar primeiros 8 bytes como chave signed 64-bit
	u := binary.BigEndian.Uint64(b[:8])
	return int64(u)
}

// TryAcquireLock tenta adquirir lock por (tenant, cpf). Em Postgres usa advisory lock; em SQLite é no-op (sempre true)
func (r *Repository) TryAcquireLock(ctx context.Context, tenantID string, cpf string, ttlSeconds int) (bool, error) {
	switch r.db.Dialector.Name() {
	case "postgres":
		key := makeLockKey(tenantID, cpf)
		var ok bool
		if err := r.db.WithContext(ctx).Raw("SELECT pg_try_advisory_lock(?)", key).Scan(&ok).Error; err != nil {
			return false, err
		}
		return ok, nil
	default:
		// sqlite e outros: considerar lock local não suportado aqui; retornar true
		return true, nil
	}
}

// ReleaseLock libera advisory lock quando suportado
func (r *Repository) ReleaseLock(ctx context.Context, tenantID string, cpf string) error {
	switch r.db.Dialector.Name() {
	case "postgres":
		key := makeLockKey(tenantID, cpf)
		// ignorar resultado booleano
		return r.db.WithContext(ctx).Exec("SELECT pg_advisory_unlock(?)", key).Error
	default:
		return nil
	}
}

func (r *Repository) ScanOpeningBalanceMissing(ctx context.Context, tenantID string, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]apprecon.Finding, error) {
	// Estratégia: obter primeira posição por ticker e comparar com Σ(buys-sells) do earliest até a first_date
	type posRow struct {
		Ticker string
		Ref    string
		Qty    float64
	}
	var posRows []posRow
	// Busca todas as posições ordenadas para derivar a primeira por ticker em Go (compatível com SQLite)
	if err := r.db.WithContext(ctx).Raw(`
        SELECT ticker, reference_date AS ref, CAST(quantity AS REAL) AS qty
        FROM b3_normalized_positions
        WHERE tenant_id = ? AND cpf = ?
        ORDER BY ticker, reference_date`, tenantID, cpf).Scan(&posRows).Error; err != nil {
		return nil, err
	}
	// Selecionar primeira por ticker
	earliestMap := map[string]posRow{}
	for _, p := range posRows {
		if len(tickers) > 0 {
			ok := false
			for _, t := range tickers {
				if t == p.Ticker {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if _, exists := earliestMap[p.Ticker]; !exists {
			// Filtrar janela se fornecida
			if from != nil && p.Ref < from.Format("2006-01-02") {
				// primeira posição fora do from; continuar procurando uma primeira dentro do recorte
				continue
			}
			if to != nil && p.Ref > to.Format("2006-01-02") {
				continue
			}
			earliestMap[p.Ticker] = p
		}
	}

	// Ordenar tickers estável para resultados determinísticos
	var tickList []string
	for t := range earliestMap {
		tickList = append(tickList, t)
	}
	sort.Strings(tickList)

	earliest := getEarliestDate()
	var out []apprecon.Finding
	for _, tkr := range tickList {
		p := earliestMap[tkr]
		lower := earliest
		if from != nil {
			// para esta regra, limite inferior é o earliest global a menos que o from seja mais recente
			if from.Format("2006-01-02") > lower {
				lower = from.Format("2006-01-02")
			}
		}
		upper := p.Ref
		// Soma líquida de transações no intervalo
		var net float64
		if err := r.db.WithContext(ctx).Raw(`
            SELECT COALESCE(SUM(CASE WHEN UPPER(side) = 'BUY' THEN CAST(quantity AS REAL) ELSE -CAST(quantity AS REAL) END), 0)
            FROM b3_normalized_transactions
            WHERE tenant_id = ? AND cpf = ? AND ticker = ? AND trade_date >= ? AND trade_date <= ?`, tenantID, cpf, tkr, lower, upper).Scan(&net).Error; err != nil {
			return nil, err
		}
		if p.Qty > net { // provável saldo pré-API
			samples := map[string]interface{}{"first_position_date": p.Ref}
			details := map[string]interface{}{"position_qty": p.Qty, "net_tx_until_first_position": net}
			hash := makeHash(tenantID, cpf, tkr, "OPENING_BALANCE_MISSING", p.Ref)
			out = append(out, apprecon.Finding{Ticker: tkr, Type: "OPENING_BALANCE_MISSING", Severity: 2, Samples: samples, Details: details, DedupeHash: hash})
			if maxSamples > 0 && len(out) >= maxSamples {
				break
			}
		}
	}
	return out, nil
}

func (r *Repository) ScanSellWithoutBuy(ctx context.Context, tenantID string, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]apprecon.Finding, error) {
	// Estratégia: identificar tickers com primeira transação SELL ou cumulativo líquido negativo
	type txRow struct {
		Ticker string
		Date   string
		Side   string
		Qty    float64
	}
	var rows []txRow
	// Buscar transações ordenadas, opcionalmente filtrando janela
	qb := r.db.WithContext(ctx).Raw(`
        SELECT ticker, trade_date AS date, UPPER(side) AS side, CAST(quantity AS REAL) AS qty
        FROM b3_normalized_transactions
        WHERE tenant_id = ? AND cpf = ?
        ORDER BY ticker, trade_date`, tenantID, cpf)
	if err := qb.Scan(&rows).Error; err != nil {
		return nil, err
	}
	// Percorrer por ticker
	var out []apprecon.Finding
	i := 0
	for i < len(rows) {
		tkr := rows[i].Ticker
		if len(tickers) > 0 {
			ok := false
			for _, t := range tickers {
				if t == tkr {
					ok = true
					break
				}
			}
			if !ok { // pular bloco deste ticker
				for i < len(rows) && rows[i].Ticker == tkr {
					i++
				}
				continue
			}
		}
		cum := 0.0
		firstSide := ""
		firstDate := ""
		found := false
		for i < len(rows) && rows[i].Ticker == tkr {
			rw := rows[i]
			// filtro de janela
			if from != nil && rw.Date < from.Format("2006-01-02") {
				i++
				continue
			}
			if to != nil && rw.Date > to.Format("2006-01-02") {
				i++
				continue
			}
			if firstSide == "" {
				firstSide = rw.Side
				firstDate = rw.Date
			}
			if rw.Side == "BUY" {
				cum += rw.Qty
			} else {
				cum -= rw.Qty
			}
			if cum < 0 {
				// cumulativo negativo
				samples := map[string]interface{}{"first_tx_date": firstDate, "first_tx_side": firstSide, "min_cum_qty": cum}
				hash := makeHash(tenantID, cpf, tkr, "SELL_WITHOUT_BUY", firstDate)
				out = append(out, apprecon.Finding{Ticker: tkr, Type: "SELL_WITHOUT_BUY", Severity: 3, Samples: samples, Details: nil, DedupeHash: hash})
				found = true
				break
			}
			i++
		}
		if !found && strings.ToUpper(firstSide) == "SELL" {
			samples := map[string]interface{}{"first_tx_date": firstDate, "first_tx_side": firstSide}
			hash := makeHash(tenantID, cpf, tkr, "SELL_WITHOUT_BUY", firstDate)
			out = append(out, apprecon.Finding{Ticker: tkr, Type: "SELL_WITHOUT_BUY", Severity: 3, Samples: samples, Details: nil, DedupeHash: hash})
		}
		if maxSamples > 0 && len(out) >= maxSamples {
			break
		}
		// avançar para próximo ticker caso não tenha avançado até o fim acima
		for i < len(rows) && rows[i].Ticker == tkr {
			i++
		}
	}
	return out, nil
}

func (r *Repository) ScanPositionTxDivergence(ctx context.Context, tenantID string, cpf string, tickers []string, from, to *time.Time, maxSamples int) ([]apprecon.Finding, error) {
	type posRow struct {
		Ticker string
		Ref    string
		Qty    float64
	}
	var rows []posRow
	// Buscar posições (opcionalmente filtrando janela)
	base := `SELECT ticker, reference_date AS ref, CAST(quantity AS REAL) AS qty FROM b3_normalized_positions WHERE tenant_id = ? AND cpf = ?`
	if from != nil {
		base += ` AND reference_date >= '` + from.Format("2006-01-02") + `'`
	}
	if to != nil {
		base += ` AND reference_date <= '` + to.Format("2006-01-02") + `'`
	}
	base += ` ORDER BY ticker, reference_date`
	if err := r.db.WithContext(ctx).Raw(base, tenantID, cpf).Scan(&rows).Error; err != nil {
		return nil, err
	}
	earliest := getEarliestDate()
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
		lower := earliest
		if from != nil && from.Format("2006-01-02") > lower {
			lower = from.Format("2006-01-02")
		}
		upper := rw.Ref
		var net float64
		if err := r.db.WithContext(ctx).Raw(`
            SELECT COALESCE(SUM(CASE WHEN UPPER(side) = 'BUY' THEN CAST(quantity AS REAL) ELSE -CAST(quantity AS REAL) END), 0)
            FROM b3_normalized_transactions
            WHERE tenant_id = ? AND cpf = ? AND ticker = ? AND trade_date >= ? AND trade_date <= ?`, tenantID, cpf, rw.Ticker, lower, upper).Scan(&net).Error; err != nil {
			return nil, err
		}
		if net != rw.Qty {
			samples := map[string]interface{}{"date": rw.Ref, "pos_qty": rw.Qty}
			details := map[string]interface{}{"net_tx_qty": net}
			hash := makeHash(tenantID, cpf, rw.Ticker, "POSITION_TX_DIVERGENCE", rw.Ref)
			out = append(out, apprecon.Finding{Ticker: rw.Ticker, Type: "POSITION_TX_DIVERGENCE", Severity: 2, Samples: samples, Details: details, DedupeHash: hash})
			if maxSamples > 0 && len(out) >= maxSamples {
				break
			}
		}
	}
	return out, nil
}

func (r *Repository) UpsertFindings(ctx context.Context, tenantID string, cpf string, findings []apprecon.Finding) error {
	// upsert idempotente via dedupe
	for _, f := range findings {
		samplesJSON, _ := json.Marshal(f.Samples)
		detailsJSON, _ := json.Marshal(f.Details)

		// Extrair campos normalizados dos JSONs
		var firstTxDate, firstPositionDate *string
		var firstTxSide *string
		var minCumQty, positionQty, netTxQty *float64

		// Extrair de Samples
		if f.Samples != nil {
			if v, ok := f.Samples["first_tx_date"].(string); ok && v != "" {
				firstTxDate = &v
			}
			if v, ok := f.Samples["first_tx_side"].(string); ok && v != "" {
				side := strings.ToUpper(v)
				firstTxSide = &side
			}
			if v, ok := f.Samples["min_cum_qty"].(float64); ok {
				minCumQty = &v
			}
			if v, ok := f.Samples["first_position_date"].(string); ok && v != "" {
				firstPositionDate = &v
			}
		}

		// Extrair de Details
		if f.Details != nil {
			if v, ok := f.Details["position_qty"].(float64); ok {
				positionQty = &v
			}
			if v, ok := f.Details["net_tx_until_first_position"].(float64); ok {
				netTxQty = &v
			}
		}

		// garantir ID em bancos sem default (ex.: SQLite)
		newID := uuid.New().String()
		if err := r.db.WithContext(ctx).Exec(`
      INSERT INTO b3_inconsistencies (
        id, tenant_id, cpf, ticker, type, status, severity, 
        affected_period_start, affected_period_end, 
        sample_dates, details, dedupe_hash, created_by_version, created_by,
        first_transaction_date, first_transaction_side, min_cumulative_quantity,
        first_position_date, position_quantity, net_transactions_quantity
      )
      VALUES (?,?,?,?,?, 'OPEN', ?, NULL, NULL, ?, ?, ?, 'v1.19', 'recon', ?,?,?,?,?,?)
      ON CONFLICT (tenant_id, cpf, ticker, type, dedupe_hash)
      DO UPDATE SET 
        last_detected_at = CURRENT_TIMESTAMP, 
        updated_at = CURRENT_TIMESTAMP,
        first_transaction_date = EXCLUDED.first_transaction_date,
        first_transaction_side = EXCLUDED.first_transaction_side,
        min_cumulative_quantity = EXCLUDED.min_cumulative_quantity,
        first_position_date = EXCLUDED.first_position_date,
        position_quantity = EXCLUDED.position_quantity,
        net_transactions_quantity = EXCLUDED.net_transactions_quantity`,
			newID, tenantID, cpf, f.Ticker, f.Type, f.Severity,
			string(samplesJSON), string(detailsJSON), f.DedupeHash,
			firstTxDate, firstTxSide, minCumQty, firstPositionDate, positionQty, netTxQty,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

// Implementações de leitura (list e get)
func (r *Repository) ListInconsistencies(ctx context.Context, tenantID string, cpf, status, typ, ticker string, from, to *time.Time, page, pageSize int) ([]apprecon.Inconsistency, error) {
	qb := r.db.WithContext(ctx).Table("b3_inconsistencies").Select(`
		id, tenant_id, cpf, ticker, type, status, severity, updated_at,
		first_transaction_date, first_transaction_side, min_cumulative_quantity,
		first_position_date, position_quantity, net_transactions_quantity
	`).Where("tenant_id = ?", tenantID)
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
	type listRow struct {
		ID                      uuid.UUID
		TenantID                string
		CPF                     string
		Ticker                  string
		Type                    string
		Status                  string
		Severity                int
		UpdatedAt               time.Time
		FirstTransactionDate    *time.Time `gorm:"column:first_transaction_date"`
		FirstTransactionSide    *string    `gorm:"column:first_transaction_side"`
		MinCumulativeQuantity   *float64   `gorm:"column:min_cumulative_quantity"`
		FirstPositionDate       *time.Time `gorm:"column:first_position_date"`
		PositionQuantity        *float64   `gorm:"column:position_quantity"`
		NetTransactionsQuantity *float64   `gorm:"column:net_transactions_quantity"`
	}
	var tmp []listRow
	if err := qb.Order("updated_at DESC").Limit(pageSize).Offset(offset).Scan(&tmp).Error; err != nil {
		return nil, err
	}
	out := make([]apprecon.Inconsistency, 0, len(tmp))
	for _, r0 := range tmp {
		out = append(out, apprecon.Inconsistency{
			ID:        r0.ID,
			TenantID:  r0.TenantID,
			CPF:       r0.CPF,
			Ticker:    r0.Ticker,
			Type:      r0.Type,
			Status:    r0.Status,
			Severity:  r0.Severity,
			UpdatedAt: r0.UpdatedAt,
			// Campos normalizados
			FirstTransactionDate:    r0.FirstTransactionDate,
			FirstTransactionSide:    r0.FirstTransactionSide,
			MinCumulativeQuantity:   r0.MinCumulativeQuantity,
			FirstPositionDate:       r0.FirstPositionDate,
			PositionQuantity:        r0.PositionQuantity,
			NetTransactionsQuantity: r0.NetTransactionsQuantity,
		})
	}
	return out, nil
}

func (r *Repository) GetInconsistency(ctx context.Context, tenantID string, id uuid.UUID) (*apprecon.Inconsistency, error) {
	// Carregar inclusive sample_dates e details
	type dbrow struct {
		ID              uuid.UUID
		TenantID        string
		CPF             string
		Ticker          string
		Type            string
		Status          string
		Severity        int
		UpdatedAt       time.Time
		FirstDetectedAt time.Time
		LastDetectedAt  time.Time
		SampleDatesJSON string
		DetailsJSON     string
		// Campos normalizados
		FirstTransactionDate    *time.Time `gorm:"column:first_transaction_date"`
		FirstTransactionSide    *string    `gorm:"column:first_transaction_side"`
		MinCumulativeQuantity   *float64   `gorm:"column:min_cumulative_quantity"`
		FirstPositionDate       *time.Time `gorm:"column:first_position_date"`
		PositionQuantity        *float64   `gorm:"column:position_quantity"`
		NetTransactionsQuantity *float64   `gorm:"column:net_transactions_quantity"`
	}
	var rrow dbrow
	if err := r.db.WithContext(ctx).Raw(`
        SELECT id, tenant_id, cpf, ticker, type, status, severity, updated_at, first_detected_at, last_detected_at,
               COALESCE(CAST(sample_dates AS TEXT), '{}') AS sample_dates_json,
               COALESCE(CAST(details AS TEXT), '{}') AS details_json,
               first_transaction_date, first_transaction_side, min_cumulative_quantity,
               first_position_date, position_quantity, net_transactions_quantity
        FROM b3_inconsistencies WHERE tenant_id = ? AND id = ?`, tenantID, id).Scan(&rrow).Error; err != nil {
		return nil, err
	}
	var samples map[string]interface{}
	var details map[string]interface{}
	_ = json.Unmarshal([]byte(rrow.SampleDatesJSON), &samples)
	_ = json.Unmarshal([]byte(rrow.DetailsJSON), &details)
	out := &apprecon.Inconsistency{
		ID:              rrow.ID,
		TenantID:        rrow.TenantID,
		CPF:             rrow.CPF,
		Ticker:          rrow.Ticker,
		Type:            rrow.Type,
		Status:          rrow.Status,
		Severity:        rrow.Severity,
		UpdatedAt:       rrow.UpdatedAt,
		FirstDetectedAt: rrow.FirstDetectedAt,
		LastDetectedAt:  rrow.LastDetectedAt,
		SampleDates:     samples, // DEPRECATED: mantido para compatibilidade
		Details:         details, // DEPRECATED: mantido para compatibilidade
		// Campos normalizados
		FirstTransactionDate:    rrow.FirstTransactionDate,
		FirstTransactionSide:    rrow.FirstTransactionSide,
		MinCumulativeQuantity:   rrow.MinCumulativeQuantity,
		FirstPositionDate:       rrow.FirstPositionDate,
		PositionQuantity:        rrow.PositionQuantity,
		NetTransactionsQuantity: rrow.NetTransactionsQuantity,
	}
	return out, nil
}
