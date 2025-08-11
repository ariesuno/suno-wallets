package reconciliation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	apprecon "suno-wallets/src/application/b3/reconciliation"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Comentários em pt-BR: Repository para auto-fix (ledger + inconsistências)

type SysOpsRepository struct{ db *gorm.DB }

func NewSysOpsRepository(db *gorm.DB) *SysOpsRepository { return &SysOpsRepository{db: db} }

func (r *SysOpsRepository) TryAcquireLock(ctx context.Context, tenantID string, cpf string, ttlSeconds int) (bool, error) {
	return (&Repository{db: r.db}).TryAcquireLock(ctx, tenantID, cpf, ttlSeconds)
}
func (r *SysOpsRepository) ReleaseLock(ctx context.Context, tenantID string, cpf string) error {
	return (&Repository{db: r.db}).ReleaseLock(ctx, tenantID, cpf)
}

func (r *SysOpsRepository) ListOpenInconsistencies(ctx context.Context, tenantID string, cpf string, types []string, tickers []string) ([]apprecon.Inconsistency, error) {
	qb := r.db.WithContext(ctx).Table("b3_inconsistencies").Select("id, tenant_id, cpf, ticker, type, status, severity, updated_at").Where("tenant_id = ? AND cpf = ? AND status = 'OPEN'", tenantID, cpf)
	if len(types) > 0 {
		qb = qb.Where("type IN ?", types)
	}
	if len(tickers) > 0 {
		qb = qb.Where("ticker IN ?", tickers)
	}
	var rows []apprecon.Inconsistency
	if err := qb.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *SysOpsRepository) GetInconsistencyByID(ctx context.Context, tenantID string, id uuid.UUID) (*apprecon.Inconsistency, error) {
	return (&Repository{db: r.db}).GetInconsistency(ctx, tenantID, id)
}

func (r *SysOpsRepository) UpsertSystemOperation(ctx context.Context, tenantID string, op apprecon.OperationPreview) (bool, error) {
	// Chave natural de idempotência
	natural := strings.Join([]string{tenantID, op.Ticker, op.Operation, op.Date, op.ReasonCode, op.InconsID.String()}, "|")
	h := sha256.Sum256([]byte(natural))
	_ = hex.EncodeToString(h[:])
	// Upsert baseado no unique index
	res := r.db.WithContext(ctx).Exec(`
      INSERT INTO b3_operations_ledger (tenant_id, cpf, ticker, asset_type, operation_date, operation_type, source, quantity, unit_price, currency, reason_code, price_confidence, generated_by_inconsistency_id, is_active, created_by, updated_by)
      VALUES (?,?,?,?, ?, ?, 'SYSTEM_SYNTHETIC', ?, ?, 'BRL', ?, ?, ?, true, 'system:auto-fix', 'system:auto-fix')
      ON CONFLICT (tenant_id, cpf, ticker, operation_type, source, operation_date, reason_code, generated_by_inconsistency_id)
      DO NOTHING`, tenantID, op.CPF, op.Ticker, "EQUITY", op.Date, op.Operation, op.Quantity, op.UnitPrice, op.ReasonCode, op.Confidence, op.InconsID)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *SysOpsRepository) ResolveInconsistency(ctx context.Context, tenantID string, inconsID uuid.UUID, generatedIDs []uuid.UUID) error {
	// Simples: marcar RESOLVED; anexar detalhes pode ficar para iteração seguinte
	return r.db.WithContext(ctx).Exec(`UPDATE b3_inconsistencies SET status = 'RESOLVED', updated_at = CURRENT_TIMESTAMP WHERE tenant_id = ? AND id = ?`, tenantID, inconsID).Error
}

// GetFirstSellTxDetails retorna data/quantidade/preço da primeira venda
func (r *SysOpsRepository) GetFirstSellTxDetails(ctx context.Context, tenantID string, cpf string, ticker string) (string, float64, float64, bool, error) {
	type row struct {
		D string
		Q float64
		P float64
	}
	var rw row
	err := r.db.WithContext(ctx).Raw(`
      SELECT trade_date AS d, CAST(quantity AS REAL) AS q, CAST(price AS REAL) AS p
      FROM b3_normalized_transactions
      WHERE tenant_id = ? AND cpf = ? AND ticker = ? AND UPPER(side) = 'SELL'
      ORDER BY trade_date LIMIT 1`, tenantID, cpf, ticker).Scan(&rw).Error
	if err != nil {
		return "", 0, 0, false, err
	}
	if rw.D == "" {
		return "", 0, 0, false, nil
	}
	return rw.D, rw.Q, rw.P, true, nil
}

// DerivePriceFromPosition tenta derivar preço unitário a partir do valor da posição (value/qty) na data
func (r *SysOpsRepository) DerivePriceFromPosition(ctx context.Context, tenantID string, cpf, ticker, date string) (float64, bool, error) {
	type row struct {
		Q float64
		V float64
	}
	var rw row
	err := r.db.WithContext(ctx).Raw(`
      SELECT CAST(quantity AS REAL) AS q, CAST(position_value AS REAL) AS v
      FROM b3_normalized_positions
      WHERE tenant_id = ? AND cpf = ? AND ticker = ? AND reference_date = ?
      LIMIT 1`, tenantID, cpf, ticker, date).Scan(&rw).Error
	if err != nil {
		return 0, false, err
	}
	if rw.Q <= 0 || rw.V <= 0 {
		return 0, false, nil
	}
	return rw.V / rw.Q, true, nil
}
