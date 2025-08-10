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

type SysOpsRepository struct { db *gorm.DB }

func NewSysOpsRepository(db *gorm.DB) *SysOpsRepository { return &SysOpsRepository{db: db} }

func (r *SysOpsRepository) TryAcquireLock(ctx context.Context, tenantID uuid.UUID, cpf string, ttlSeconds int) (bool, error) {
    return (&Repository{db: r.db}).TryAcquireLock(ctx, tenantID, cpf, ttlSeconds)
}
func (r *SysOpsRepository) ReleaseLock(ctx context.Context, tenantID uuid.UUID, cpf string) error {
    return (&Repository{db: r.db}).ReleaseLock(ctx, tenantID, cpf)
}

func (r *SysOpsRepository) ListOpenInconsistencies(ctx context.Context, tenantID uuid.UUID, cpf string, types []string, tickers []string) ([]apprecon.Inconsistency, error) {
    qb := r.db.WithContext(ctx).Table("b3_inconsistencies").Select("id, tenant_id, cpf, ticker, type, status, severity, updated_at").Where("tenant_id = ? AND cpf = ? AND status = 'OPEN'", tenantID, cpf)
    if len(types) > 0 {
        qb = qb.Where("type IN ?", types)
    }
    if len(tickers) > 0 {
        qb = qb.Where("ticker IN ?", tickers)
    }
    var rows []apprecon.Inconsistency
    if err := qb.Scan(&rows).Error; err != nil { return nil, err }
    return rows, nil
}

func (r *SysOpsRepository) UpsertSystemOperation(ctx context.Context, tenantID uuid.UUID, op apprecon.OperationPreview) (bool, error) {
    // Chave natural de idempotência
    natural := strings.Join([]string{tenantID.String(), op.Ticker, op.Operation, op.Date, op.ReasonCode, op.InconsID.String()}, "|")
    h := sha256.Sum256([]byte(natural))
    _ = hex.EncodeToString(h[:])
    // Upsert baseado no unique index
    res := r.db.WithContext(ctx).Exec(`
      INSERT INTO b3_operations_ledger (tenant_id, cpf, ticker, asset_type, operation_date, operation_type, source, quantity, unit_price, currency, reason_code, price_confidence, generated_by_inconsistency_id, is_active, created_by, updated_by)
      VALUES (?,?,?,?, ?, ?, 'SYSTEM_SYNTHETIC', 0, NULL, 'BRL', ?, ?, ?, true, 'system:auto-fix', 'system:auto-fix')
      ON CONFLICT (tenant_id, cpf, ticker, operation_type, source, operation_date, reason_code, generated_by_inconsistency_id)
      DO NOTHING`, tenantID, op.InconsID.String(), op.Ticker, "EQUITY", op.Date, op.Operation, op.ReasonCode, op.Confidence, op.InconsID)
    if res.Error != nil { return false, res.Error }
    return res.RowsAffected > 0, nil
}

func (r *SysOpsRepository) ResolveInconsistency(ctx context.Context, tenantID uuid.UUID, inconsID uuid.UUID, generatedIDs []uuid.UUID) error {
    // Simples: marcar RESOLVED; anexar detalhes pode ficar para iteração seguinte
    return r.db.WithContext(ctx).Exec(`UPDATE b3_inconsistencies SET status = 'RESOLVED', updated_at = CURRENT_TIMESTAMP WHERE tenant_id = ? AND id = ?`, tenantID, inconsID).Error
}


