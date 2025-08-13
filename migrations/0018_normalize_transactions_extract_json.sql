-- Migration: Extract JSON fields to dedicated columns in normalized_transactions
-- Remove extra_json field and add dedicated columns for all B3 transaction data

BEGIN;

-- Create new table with proper structure (no extra_json)
CREATE TABLE b3_normalized_transactions_new (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    cpf character varying(11) NOT NULL,
    asset_type character varying NOT NULL,
    source_version character varying NOT NULL,
    raw_id uuid NOT NULL,
    sequence_in_raw integer NOT NULL,
    
    -- Existing transaction fields
    trade_id text,
    broker_code text,
    trade_date date NOT NULL,
    settlement_date date,
    ticker text NOT NULL,
    isin text,
    side character varying NOT NULL,
    quantity numeric(28,10) NOT NULL,
    price numeric(28,10) NOT NULL,
    gross_value numeric(28,10),
    currency character varying(8),
    
    -- New fields extracted from extra_json
    market_name text,                        -- from marketName
    participant_name text,                   -- from participantName  
    participant_document text,               -- from participantDocumentNumber
    asset_trading_code text,                 -- from assetTradingObjectCode
    expiration_date date,                    -- from expirationDate
    option_exercise_value numeric(28,10),    -- from optionExerciseValue
    original_trade_price numeric(28,10),     -- from originalTradePriceValue
    original_adjustment_value numeric(28,10), -- from originalTotalAdjustmentValue
    trade_datetime timestamp with time zone, -- from tradeDateTime
    
    -- Metadata fields
    normalized_hash character varying(64) NOT NULL,
    normalized_at timestamp with time zone NOT NULL DEFAULT now(),
    tenant_id character varying(50) NOT NULL
);

-- Migrate data from old table to new table
INSERT INTO b3_normalized_transactions_new (
    id, cpf, asset_type, source_version, raw_id, sequence_in_raw,
    trade_id, broker_code, trade_date, settlement_date, ticker, isin, 
    side, quantity, price, gross_value, currency,
    market_name, participant_name, participant_document, asset_trading_code,
    expiration_date, option_exercise_value, original_trade_price, 
    original_adjustment_value, trade_datetime,
    normalized_hash, normalized_at, tenant_id
)
SELECT 
    id, cpf, asset_type, source_version, raw_id, sequence_in_raw,
    trade_id, broker_code, trade_date, settlement_date, ticker, isin,
    side, quantity, price, gross_value, currency,
    
    -- Extract JSON fields
    extra_json->>'marketName',
    extra_json->>'participantName',
    extra_json->>'participantDocumentNumber',
    extra_json->>'assetTradingObjectCode',
    CASE 
        WHEN extra_json->>'expirationDate' = '9999-12-31' THEN NULL
        ELSE (extra_json->>'expirationDate')::date
    END,
    COALESCE((extra_json->>'optionExerciseValue')::numeric, 0),
    (extra_json->>'originalTradePriceValue')::numeric,
    COALESCE((extra_json->>'originalTotalAdjustmentValue')::numeric, 0),
    (extra_json->>'tradeDateTime')::timestamp with time zone,
    
    normalized_hash, normalized_at, tenant_id
FROM b3_normalized_transactions;

-- Drop old table and rename new one
DROP TABLE b3_normalized_transactions;
ALTER TABLE b3_normalized_transactions_new RENAME TO b3_normalized_transactions;

-- Recreate indexes
CREATE UNIQUE INDEX ux_b3_norm_tx ON b3_normalized_transactions (
    tenant_id, cpf, asset_type, raw_id, sequence_in_raw, normalized_hash
);

-- Create additional useful indexes
CREATE INDEX ix_b3_norm_tx_cpf_date ON b3_normalized_transactions (tenant_id, cpf, trade_date);
CREATE INDEX ix_b3_norm_tx_ticker ON b3_normalized_transactions (ticker);
CREATE INDEX ix_b3_norm_tx_participant ON b3_normalized_transactions (participant_name);
CREATE INDEX ix_b3_norm_tx_market ON b3_normalized_transactions (market_name);

-- Update archive table structure as well
DROP TABLE IF EXISTS b3_normalized_transactions_archive;
CREATE TABLE b3_normalized_transactions_archive (
    LIKE b3_normalized_transactions INCLUDING ALL,
    archived_at timestamp with time zone NOT NULL,
    archived_by text NOT NULL
);

COMMIT;
