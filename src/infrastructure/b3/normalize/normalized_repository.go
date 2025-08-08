package normalize

import (
	"context"
	"encoding/json"
	"suno-wallets/src/infrastructure/b3/persistence"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Comentários em pt-BR: persistência de registros normalizados

type NormalizedTransaction struct {
	ID             uuid.UUID       `gorm:"type:uuid;primaryKey"`
	TenantID       uuid.UUID       `gorm:"type:uuid;not null"`
	CPF            string          `gorm:"type:varchar(11);not null"`
	AssetType      string          `gorm:"type:varchar;not null"`
	SourceVersion  string          `gorm:"type:varchar;not null"`
	RawID          uuid.UUID       `gorm:"type:uuid;not null"`
	SequenceInRaw  int             `gorm:"not null"`
	TradeID        *string         `gorm:"type:text"`
	BrokerCode     *string         `gorm:"type:text"`
	TradeDate      time.Time       `gorm:"type:date;not null"`
	SettlementDate *time.Time      `gorm:"type:date"`
	Ticker         string          `gorm:"type:text;not null"`
	ISIN           *string         `gorm:"type:text"`
	Side           string          `gorm:"type:varchar;not null"`
	Quantity       string          `gorm:"type:numeric"`
	Price          string          `gorm:"type:numeric"`
	GrossValue     *string         `gorm:"type:numeric"`
	Currency       *string         `gorm:"type:varchar(8)"`
	ExtraJSON      json.RawMessage `gorm:"type:jsonb"`
	NormalizedHash string          `gorm:"type:varchar(64);not null"`
	NormalizedAt   time.Time       `gorm:"type:timestamptz;not null"`
}

type NormalizedPosition struct {
	ID             uuid.UUID       `gorm:"type:uuid;primaryKey"`
	TenantID       uuid.UUID       `gorm:"type:uuid;not null"`
	CPF            string          `gorm:"type:varchar(11);not null"`
	AssetType      string          `gorm:"type:varchar;not null"`
	SourceVersion  string          `gorm:"type:varchar;not null"`
	RawID          uuid.UUID       `gorm:"type:uuid;not null"`
	SequenceInRaw  int             `gorm:"not null"`
	ReferenceDate  time.Time       `gorm:"type:date;not null"`
	Ticker         string          `gorm:"type:text;not null"`
	ISIN           *string         `gorm:"type:text"`
	Quantity       string          `gorm:"type:numeric"`
	AvgPrice       *string         `gorm:"type:numeric"`
	PositionValue  *string         `gorm:"type:numeric"`
	Currency       *string         `gorm:"type:varchar(8)"`
	ExtraJSON      json.RawMessage `gorm:"type:jsonb"`
	NormalizedHash string          `gorm:"type:varchar(64);not null"`
	NormalizedAt   time.Time       `gorm:"type:timestamptz;not null"`
}

type NormalizedRepository interface {
	UpsertTransactions(ctx context.Context, items []NormalizedTransaction) error
	UpsertPositions(ctx context.Context, items []NormalizedPosition) error
	SelectPendingRaw(ctx context.Context, tenantID uuid.UUID, cpf, dataType, assetType string, start, end time.Time, force bool) ([]persistence.RawRecord, error)
	MarkRawNormalized(ctx context.Context, rawID uuid.UUID, count int) error
}

type normalizedRepositoryImpl struct{ db *gorm.DB }

func NewNormalizedRepository(db *gorm.DB) NormalizedRepository {
	return &normalizedRepositoryImpl{db: db}
}

func (r *normalizedRepositoryImpl) UpsertTransactions(ctx context.Context, items []NormalizedTransaction) error {
	for _, it := range items {
		if err := r.db.WithContext(ctx).Exec(`
            INSERT INTO b3_normalized_transactions (
                id, tenant_id, cpf, asset_type, source_version, raw_id, sequence_in_raw, trade_id, broker_code,
                trade_date, settlement_date, ticker, isin, side, quantity, price, gross_value, currency, extra_json,
                normalized_hash, normalized_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT (tenant_id, cpf, asset_type, raw_id, sequence_in_raw, normalized_hash)
            DO UPDATE SET normalized_at = EXCLUDED.normalized_at
        `,
			it.ID, it.TenantID, it.CPF, it.AssetType, it.SourceVersion, it.RawID, it.SequenceInRaw, it.TradeID, it.BrokerCode,
			it.TradeDate, it.SettlementDate, it.Ticker, it.ISIN, it.Side, it.Quantity, it.Price, it.GrossValue, it.Currency, it.ExtraJSON,
			it.NormalizedHash, time.Now(),
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *normalizedRepositoryImpl) UpsertPositions(ctx context.Context, items []NormalizedPosition) error {
	for _, it := range items {
		if err := r.db.WithContext(ctx).Exec(`
            INSERT INTO b3_normalized_positions (
                id, tenant_id, cpf, asset_type, source_version, raw_id, sequence_in_raw, reference_date, ticker, isin,
                quantity, avg_price, position_value, currency, extra_json, normalized_hash, normalized_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT (tenant_id, cpf, asset_type, raw_id, sequence_in_raw, normalized_hash)
            DO UPDATE SET normalized_at = EXCLUDED.normalized_at
        `,
			it.ID, it.TenantID, it.CPF, it.AssetType, it.SourceVersion, it.RawID, it.SequenceInRaw, it.ReferenceDate, it.Ticker, it.ISIN,
			it.Quantity, it.AvgPrice, it.PositionValue, it.Currency, it.ExtraJSON, it.NormalizedHash, time.Now(),
		).Error; err != nil {
			return err
		}
	}
	return nil
}

// Seleciona RAW pendente (sem normalized_at) dentro do período, ou tudo se force=true
func (r *normalizedRepositoryImpl) SelectPendingRaw(ctx context.Context, tenantID uuid.UUID, cpf, dataType, assetType string, start, end time.Time, force bool) ([]persistence.RawRecord, error) {
	var rows []persistence.RawRecord
	qb := r.db.WithContext(ctx).Table("b3_raw_data_client").
		Where("tenant_id = ? AND cpf = ? AND data_type = ? AND asset_type = ? AND period_start >= ? AND period_end <= ?", tenantID, cpf, dataType, assetType, start, end)
	if !force {
		qb = qb.Where("(normalized_at IS NULL OR normalized_count = 0)")
	}
	if err := qb.Order("period_start, page").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Marca o RAW como normalizado e atualiza contagem
func (r *normalizedRepositoryImpl) MarkRawNormalized(ctx context.Context, rawID uuid.UUID, count int) error {
	return r.db.WithContext(ctx).Exec(`
        UPDATE b3_raw_data_client SET normalized_at = now(), normalized_count = COALESCE(normalized_count,0) + ?
        WHERE id = ?
    `, count, rawID).Error
}
