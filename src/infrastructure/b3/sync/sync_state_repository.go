package sync

import (
    "context"
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// Comentários em pt-BR: repositório para estado de sincronismo da B3

type SyncState struct {
    ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
    TenantID      uuid.UUID `gorm:"type:uuid;not null;index"`
    CPF           string    `gorm:"type:varchar(11);not null;index"`
    IsActive      bool      `gorm:"not null;default:true"`
    LastTxSyncAt  *time.Time
    LastPosSyncAt *time.Time
    LastCheckedAt *time.Time
    LastResult    *string
    LastError     *string
    NeedsReprocess bool     `gorm:"not null;default:false"`
    FailureCount  int       `gorm:"not null;default:0"`
    CreatedAt     time.Time `gorm:"not null;default:now()"`
    UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

type Repository interface {
    GetActiveByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]SyncState, error)
    GetByTenantCPF(ctx context.Context, tenantID uuid.UUID, cpf string) (*SyncState, error)
    Upsert(ctx context.Context, st *SyncState) error
    MarkResult(ctx context.Context, id uuid.UUID, success bool, needsReprocess bool, lastResult string, lastError *string, txSyncAt, posSyncAt *time.Time) error
}

type repositoryImpl struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repositoryImpl{db: db} }

func (r *repositoryImpl) GetActiveByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]SyncState, error) {
    var out []SyncState
    q := r.db.WithContext(ctx).Table("b3_sync_state").Where("tenant_id = ? AND is_active = true", tenantID).Order("cpf")
    if limit > 0 { q = q.Limit(limit) }
    if err := q.Scan(&out).Error; err != nil { return nil, err }
    return out, nil
}

func (r *repositoryImpl) GetByTenantCPF(ctx context.Context, tenantID uuid.UUID, cpf string) (*SyncState, error) {
    var st SyncState
    if err := r.db.WithContext(ctx).Raw(`SELECT * FROM b3_sync_state WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Scan(&st).Error; err != nil { return nil, err }
    if st.ID == uuid.Nil { return nil, nil }
    return &st, nil
}

func (r *repositoryImpl) Upsert(ctx context.Context, st *SyncState) error {
    return r.db.WithContext(ctx).Exec(`
        INSERT INTO b3_sync_state (id, tenant_id, cpf, is_active, last_tx_sync_at, last_pos_sync_at, last_checked_at, last_result, last_error, needs_reprocess, failure_count, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now())
        ON CONFLICT (tenant_id, cpf)
        DO UPDATE SET is_active = EXCLUDED.is_active
    `, st.ID, st.TenantID, st.CPF, st.IsActive, st.LastTxSyncAt, st.LastPosSyncAt, st.LastCheckedAt, st.LastResult, st.LastError, st.NeedsReprocess, st.FailureCount).Error
}

func (r *repositoryImpl) MarkResult(ctx context.Context, id uuid.UUID, success bool, needsReprocess bool, lastResult string, lastError *string, txSyncAt, posSyncAt *time.Time) error {
    set := "last_checked_at = now(), last_result = ?, needs_reprocess = ?, updated_at = now()"
    args := []any{lastResult, needsReprocess}
    if lastError != nil { set += ", last_error = ?"; args = append(args, *lastError) }
    if success {
        if txSyncAt != nil { set += ", last_tx_sync_at = ?"; args = append(args, *txSyncAt) }
        if posSyncAt != nil { set += ", last_pos_sync_at = ?"; args = append(args, *posSyncAt) }
        set += ", failure_count = 0"
    } else {
        set += ", failure_count = failure_count + 1"
    }
    args = append(args, id)
    return r.db.WithContext(ctx).Exec("UPDATE b3_sync_state SET "+set+" WHERE id = ?", args...).Error
}


