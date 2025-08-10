package ops

import (
    "context"
    "github.com/google/uuid"
    "gorm.io/gorm"
    appops "suno-wallets/src/application/ops"
)

// Comentários em pt-BR: repository para USER_MANUAL no ledger

type OperationsRepository struct{ db *gorm.DB }

func NewOperationsRepository(db *gorm.DB) *OperationsRepository { return &OperationsRepository{db: db} }

func (r *OperationsRepository) CreateManual(ctx context.Context, op appops.ManualOperation) (uuid.UUID, error) {
    id := uuid.New()
    err := r.db.WithContext(ctx).Exec(`
      INSERT INTO b3_operations_ledger (id, tenant_id, cpf, ticker, asset_type, operation_date, operation_type, source, quantity, unit_price, currency, is_active, created_by, updated_by)
      VALUES (?,?,?,?, ?, ?, ?, 'USER_MANUAL', ?, ?, COALESCE(?, 'BRL'), true, 'user:api', 'user:api')
    `, id, op.TenantID, op.CPF, op.Ticker, op.AssetType, op.OperationDate.Format("2006-01-02"), op.OperationType, op.Quantity, op.UnitPrice, op.Currency).Error
    return id, err
}

func (r *OperationsRepository) UpdateManual(ctx context.Context, op appops.ManualOperation) error {
    return r.db.WithContext(ctx).Exec(`
      UPDATE b3_operations_ledger
      SET ticker = ?, asset_type = ?, operation_date = ?, operation_type = ?, quantity = ?, unit_price = ?, currency = ?, updated_by = 'user:api', updated_at = CURRENT_TIMESTAMP
      WHERE tenant_id = ? AND id = ? AND source = 'USER_MANUAL' AND is_active = true
    `, op.Ticker, op.AssetType, op.OperationDate.Format("2006-01-02"), op.OperationType, op.Quantity, op.UnitPrice, op.Currency, op.TenantID, op.ID).Error
}

func (r *OperationsRepository) SoftDeleteManual(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
    return r.db.WithContext(ctx).Exec(`
      UPDATE b3_operations_ledger SET is_active = false, updated_by = 'user:api', updated_at = CURRENT_TIMESTAMP
      WHERE tenant_id = ? AND id = ? AND source = 'USER_MANUAL'
    `, tenantID, id).Error
}

func (r *OperationsRepository) ListManual(ctx context.Context, tenantID uuid.UUID, cpf string, filters map[string]interface{}, page, pageSize int) ([]appops.ManualOperation, error) {
    qb := r.db.WithContext(ctx).Table("b3_operations_ledger").Select("id, tenant_id, cpf, ticker, asset_type, operation_date, operation_type, quantity, unit_price, currency").Where("tenant_id = ? AND source = 'USER_MANUAL'", tenantID)
    if cpf != "" { qb = qb.Where("cpf = ?", cpf) }
    var out []appops.ManualOperation
    if err := qb.Order("operation_date DESC").Limit(pageSize).Offset((page-1)*pageSize).Scan(&out).Error; err != nil { return nil, err }
    return out, nil
}

func (r *OperationsRepository) GetPolicyMode(ctx context.Context, tenantID uuid.UUID, cpf string) (string, error) {
    type row struct{ Mode string }
    var rw row
    if err := r.db.WithContext(ctx).Raw(`SELECT COALESCE(mode,'HYBRID') AS mode FROM dedup_policies WHERE tenant_id = ? AND cpf = ? LIMIT 1`, tenantID, cpf).Scan(&rw).Error; err != nil {
        return "HYBRID", nil
    }
    if rw.Mode == "" { return "HYBRID", nil }
    return rw.Mode, nil
}


