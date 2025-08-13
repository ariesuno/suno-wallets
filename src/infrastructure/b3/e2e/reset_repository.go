package e2e

import (
	"context"
	"fmt"

	appe2e "suno-wallets/src/application/b3/e2e"

	"gorm.io/gorm"
)

// Comentários em pt-BR: Repositório de reset seguro (archive/hard-delete) + locks simples por (tenant, cpf)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// TryAcquireLock usa advisory locks do Postgres por (tenant, cpf)
func (r *Repository) TryAcquireLock(ctx context.Context, tenantID string, cpf string) (bool, error) {
	var ok bool
	// chave: hash de tenant e cpf em bigint (usa pg's advisory lock por two-int)
	// Atenção: para simplicidade, usamos uma única chave derivada via hashtext
	err := r.db.WithContext(ctx).Raw(`SELECT pg_try_advisory_lock(hashtext(?))`, tenantID+":"+cpf).Scan(&ok).Error
	return ok, err
}

func (r *Repository) ReleaseLock(ctx context.Context, tenantID string, cpf string) error {
	return r.db.WithContext(ctx).Exec(`SELECT pg_advisory_unlock(hashtext(?))`, tenantID+":"+cpf).Error
}

// Reset move para _archive (ou deleta) dados vinculados ao CPF e limpa períodos/sync_state
func (r *Repository) Reset(ctx context.Context, tenantID string, cpf, mode, archivedBy string) (*appe2e.ResetResult, error) {
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if tx.Error != nil {
			tx.Rollback()
		}
	}()

	res := &appe2e.ResetResult{}

	// Garantir tabelas de archive existam (não cria aqui; migrations devem criar)
	if mode != "archive" && mode != "hard-delete" {
		return nil, fmt.Errorf("invalid mode: %s", mode)
	}

	// RAW
	if mode == "archive" {
		if err := tx.Exec(`INSERT INTO b3_raw_data_client_archive
            (id, cpf, data_type, asset_type, period_start, period_end, page, payload_json, payload_hash, 
             source_version, endpoint_path, http_status, fetched_at, retry_count, request_id, 
             normalized_at, normalized_count, tenant_id, archived_at, archived_by)
            SELECT id, cpf, data_type, asset_type, period_start, period_end, page, payload_json, payload_hash,
                   source_version, endpoint_path, http_status, fetched_at, retry_count, request_id,
                   normalized_at, normalized_count, tenant_id, now(), ?
            FROM b3_raw_data_client WHERE tenant_id = ? AND cpf = ?`, archivedBy, tenantID, cpf).Error; err != nil {
			return nil, err
		}
		res.RawMoved = int(tx.RowsAffected)
		if err := tx.Exec(`DELETE FROM b3_raw_data_client WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Error; err != nil {
			return nil, err
		}
	} else {
		if err := tx.Exec(`DELETE FROM b3_raw_data_client WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Error; err != nil {
			return nil, err
		}
		res.RawDeleted = int(tx.RowsAffected)
	}

	// Normalized transactions
	if mode == "archive" {
		if err := tx.Exec(`INSERT INTO b3_normalized_transactions_archive
            (id, cpf, asset_type, source_version, raw_id, sequence_in_raw, trade_id, broker_code, 
             trade_date, settlement_date, ticker, isin, side, quantity, price, gross_value, 
             currency, market_name, participant_name, participant_document, asset_trading_code,
             expiration_date, option_exercise_value, original_trade_price, original_adjustment_value,
             trade_datetime, normalized_hash, normalized_at, tenant_id, archived_at, archived_by)
            SELECT id, cpf, asset_type, source_version, raw_id, sequence_in_raw, trade_id, broker_code,
                   trade_date, settlement_date, ticker, isin, side, quantity, price, gross_value,
                   currency, market_name, participant_name, participant_document, asset_trading_code,
                   expiration_date, option_exercise_value, original_trade_price, original_adjustment_value,
                   trade_datetime, normalized_hash, normalized_at, tenant_id, now(), ?
            FROM b3_normalized_transactions WHERE tenant_id = ? AND cpf = ?`, archivedBy, tenantID, cpf).Error; err != nil {
			return nil, err
		}
		res.NormTxMoved = int(tx.RowsAffected)
		if err := tx.Exec(`DELETE FROM b3_normalized_transactions WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Error; err != nil {
			return nil, err
		}
	} else {
		if err := tx.Exec(`DELETE FROM b3_normalized_transactions WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Error; err != nil {
			return nil, err
		}
		res.NormTxDeleted = int(tx.RowsAffected)
	}

	// Normalized positions
	if mode == "archive" {
		if err := tx.Exec(`INSERT INTO b3_normalized_positions_archive
            (id, cpf, asset_type, source_version, raw_id, sequence_in_raw, reference_date, ticker, 
             isin, quantity, avg_price, position_value, currency, extra_json, normalized_hash, 
             normalized_at, tenant_id, archived_at, archived_by)
            SELECT id, cpf, asset_type, source_version, raw_id, sequence_in_raw, reference_date, ticker,
                   isin, quantity, avg_price, position_value, currency, extra_json, normalized_hash,
                   normalized_at, tenant_id, now(), ?
            FROM b3_normalized_positions WHERE tenant_id = ? AND cpf = ?`, archivedBy, tenantID, cpf).Error; err != nil {
			return nil, err
		}
		res.NormPosMoved = int(tx.RowsAffected)
		if err := tx.Exec(`DELETE FROM b3_normalized_positions WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Error; err != nil {
			return nil, err
		}
	} else {
		if err := tx.Exec(`DELETE FROM b3_normalized_positions WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Error; err != nil {
			return nil, err
		}
		res.NormPosDeleted = int(tx.RowsAffected)
	}

	// Fetched periods
	if err := tx.Exec(`DELETE FROM b3_fetched_periods WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Error; err != nil {
		return nil, err
	}
	res.PeriodsCleared = int(tx.RowsAffected)

	// Sync state reset
	if err := tx.Exec(`UPDATE b3_sync_state SET last_tx_sync_at = NULL, last_pos_sync_at = NULL, last_checked_at = NULL,
        last_result = NULL, last_error = NULL, needs_reprocess = false, failure_count = 0 WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Error; err != nil {
		return nil, err
	}
	res.SyncStateReset = true

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return res, nil
}
