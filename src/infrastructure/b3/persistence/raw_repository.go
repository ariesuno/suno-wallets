package persistence

import (
    "context"
    "encoding/json"
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// Comentários em pt-BR: repositório para persistência do RAW da B3

type RawRecord struct {
    ID           uuid.UUID       `gorm:"type:uuid;primaryKey"`
    TenantID     uuid.UUID       `gorm:"type:uuid;not null;index"`
    CPF          string          `gorm:"type:varchar(11);not null;index"`
    DataType     string          `gorm:"type:varchar;not null;index"`
    AssetType    string          `gorm:"type:varchar;not null;index"`
    PeriodStart  time.Time       `gorm:"type:date;not null;index"`
    PeriodEnd    time.Time       `gorm:"type:date;not null;index"`
    Page         int             `gorm:"not null"`
    PayloadJSON  json.RawMessage `gorm:"type:jsonb;not null"`
    PayloadHash  string          `gorm:"type:varchar(64);not null"`
    SourceVer    string          `gorm:"type:varchar;not null"`
    EndpointPath string          `gorm:"type:text;not null"`
    HTTPStatus   int             `gorm:"not null"`
    FetchedAt    time.Time       `gorm:"type:timestamptz;not null;default:now()"`
    RetryCount   int             `gorm:"not null;default:0"`
    RequestID    uuid.UUID       `gorm:"type:uuid;not null"`
}

type FetchedPeriod struct {
    ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
    TenantID      uuid.UUID `gorm:"type:uuid;not null;index"`
    CPF           string    `gorm:"type:varchar(11);not null;index"`
    DataType      string    `gorm:"type:varchar;not null;index"`
    AssetType     string    `gorm:"type:varchar;not null;index"`
    MonthStart    time.Time `gorm:"type:date;not null;index"`
    MonthEnd      time.Time `gorm:"type:date;not null"`
    Pages         int       `gorm:"not null"`
    Completed     bool      `gorm:"not null;default:false"`
    LastFetchedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

type RawRepository interface {
    UpsertRaw(ctx context.Context, rec *RawRecord) error
    MarkMonth(ctx context.Context, p *FetchedPeriod) error
    GetMonth(ctx context.Context, tenantID uuid.UUID, cpf, dataType, assetType string, monthStart time.Time) (*FetchedPeriod, error)
}

type rawRepositoryImpl struct{ db *gorm.DB }

func NewRawRepository(db *gorm.DB) RawRepository { return &rawRepositoryImpl{db: db} }

func (r *rawRepositoryImpl) UpsertRaw(ctx context.Context, rec *RawRecord) error {
    // upsert por chave lógica (tenant, cpf, tipo, asset, período, page, hash)
    return r.db.WithContext(ctx).Exec(`
        INSERT INTO b3_raw_data_client (
            id, tenant_id, cpf, data_type, asset_type, period_start, period_end, page,
            payload_json, payload_hash, source_version, endpoint_path, http_status, fetched_at, retry_count, request_id
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now(), ?, ?)
        ON CONFLICT (tenant_id, cpf, data_type, asset_type, period_start, period_end, page, payload_hash)
        DO UPDATE SET http_status = EXCLUDED.http_status, fetched_at = now(), retry_count = EXCLUDED.retry_count
    `,
        rec.ID, rec.TenantID, rec.CPF, rec.DataType, rec.AssetType, rec.PeriodStart, rec.PeriodEnd, rec.Page,
        rec.PayloadJSON, rec.PayloadHash, rec.SourceVer, rec.EndpointPath, rec.HTTPStatus, rec.RetryCount, rec.RequestID,
    ).Error
}

func (r *rawRepositoryImpl) MarkMonth(ctx context.Context, p *FetchedPeriod) error {
    return r.db.WithContext(ctx).Exec(`
        INSERT INTO b3_fetched_periods (
            id, tenant_id, cpf, data_type, asset_type, month_start, month_end, pages, completed, last_fetched_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, now())
        ON CONFLICT (tenant_id, cpf, data_type, asset_type, month_start)
        DO UPDATE SET pages = EXCLUDED.pages, completed = EXCLUDED.completed, last_fetched_at = now()
    `,
        p.ID, p.TenantID, p.CPF, p.DataType, p.AssetType, p.MonthStart, p.MonthEnd, p.Pages, p.Completed,
    ).Error
}

func (r *rawRepositoryImpl) GetMonth(ctx context.Context, tenantID uuid.UUID, cpf, dataType, assetType string, monthStart time.Time) (*FetchedPeriod, error) {
    var m FetchedPeriod
    err := r.db.WithContext(ctx).Raw(
        `SELECT id, tenant_id, cpf, data_type, asset_type, month_start, month_end, pages, completed, last_fetched_at
         FROM b3_fetched_periods
         WHERE tenant_id = ? AND cpf = ? AND data_type = ? AND asset_type = ? AND month_start = ?`,
        tenantID, cpf, dataType, assetType, monthStart,
    ).Scan(&m).Error
    if err != nil {
        return nil, err
    }
    if m.ID == uuid.Nil {
        return nil, nil
    }
    return &m, nil
}


