-- Refatora tenant_id de UUID para VARCHAR(50) em todas as tabelas
-- Para facilitar uso com nomes: nai, status_invest, fiis, funds_explorer, orcana

-- Função para mapear UUIDs existentes para nomes de tenant
CREATE OR REPLACE FUNCTION map_uuid_to_tenant_name(uuid_val UUID) RETURNS VARCHAR(50) AS $$
BEGIN
  CASE uuid_val::text
    WHEN '00000000-0000-0000-0000-000000000001' THEN RETURN 'status_invest';
    WHEN '00000000-0000-0000-0000-000000000002' THEN RETURN 'nai';
    WHEN '00000000-0000-0000-0000-000000000003' THEN RETURN 'fiis';
    WHEN '00000000-0000-0000-0000-000000000004' THEN RETURN 'funds_explorer';
    WHEN '00000000-0000-0000-0000-000000000005' THEN RETURN 'orcana';
    ELSE RETURN 'status_invest'; -- Default fallback
  END CASE;
END;
$$ LANGUAGE plpgsql;

-- B3 Raw Data
ALTER TABLE b3_raw_data_client 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE b3_raw_data_client 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE b3_raw_data_client 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE b3_raw_data_client 
  RENAME COLUMN tenant_name TO tenant_id;

-- B3 Fetched Periods
ALTER TABLE b3_fetched_periods 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE b3_fetched_periods 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE b3_fetched_periods 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE b3_fetched_periods 
  RENAME COLUMN tenant_name TO tenant_id;

-- B3 Normalized Transactions
ALTER TABLE b3_normalized_transactions 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE b3_normalized_transactions 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE b3_normalized_transactions 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE b3_normalized_transactions 
  RENAME COLUMN tenant_name TO tenant_id;

-- B3 Normalized Positions
ALTER TABLE b3_normalized_positions 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE b3_normalized_positions 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE b3_normalized_positions 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE b3_normalized_positions 
  RENAME COLUMN tenant_name TO tenant_id;

-- B3 Inconsistencies
ALTER TABLE b3_inconsistencies 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE b3_inconsistencies 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE b3_inconsistencies 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE b3_inconsistencies 
  RENAME COLUMN tenant_name TO tenant_id;

-- B3 Operations Ledger
ALTER TABLE b3_operations_ledger 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE b3_operations_ledger 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE b3_operations_ledger 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE b3_operations_ledger 
  RENAME COLUMN tenant_name TO tenant_id;

-- B3 Sync State
ALTER TABLE b3_sync_state 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE b3_sync_state 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE b3_sync_state 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE b3_sync_state 
  RENAME COLUMN tenant_name TO tenant_id;

-- Dedup Candidates
ALTER TABLE dedup_candidates 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE dedup_candidates 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE dedup_candidates 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE dedup_candidates 
  RENAME COLUMN tenant_name TO tenant_id;

-- Dedup Policies
ALTER TABLE dedup_policies 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE dedup_policies 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE dedup_policies 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE dedup_policies 
  RENAME COLUMN tenant_name TO tenant_id;

-- Client Data Source Policy
ALTER TABLE client_data_source_policy 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE client_data_source_policy 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE client_data_source_policy 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE client_data_source_policy 
  RENAME COLUMN tenant_name TO tenant_id;

-- Client Data Source Policy Audit
ALTER TABLE client_data_source_policy_audit 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE client_data_source_policy_audit 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE client_data_source_policy_audit 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE client_data_source_policy_audit 
  RENAME COLUMN tenant_name TO tenant_id;

-- Admin Action Audit
ALTER TABLE admin_action_audit 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE admin_action_audit 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE admin_action_audit 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE admin_action_audit 
  RENAME COLUMN tenant_name TO tenant_id;

-- Archive Tables
ALTER TABLE b3_raw_data_client_archive 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE b3_raw_data_client_archive 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE b3_raw_data_client_archive 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE b3_raw_data_client_archive 
  RENAME COLUMN tenant_name TO tenant_id;

ALTER TABLE b3_normalized_transactions_archive 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE b3_normalized_transactions_archive 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE b3_normalized_transactions_archive 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE b3_normalized_transactions_archive 
  RENAME COLUMN tenant_name TO tenant_id;

ALTER TABLE b3_normalized_positions_archive 
  ADD COLUMN tenant_name VARCHAR(50);
UPDATE b3_normalized_positions_archive 
  SET tenant_name = map_uuid_to_tenant_name(tenant_id);
ALTER TABLE b3_normalized_positions_archive 
  ALTER COLUMN tenant_name SET NOT NULL,
  DROP COLUMN tenant_id;
ALTER TABLE b3_normalized_positions_archive 
  RENAME COLUMN tenant_name TO tenant_id;

-- Domain Tables (se existirem)
DO $$ 
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'wallets') THEN
    ALTER TABLE wallets ADD COLUMN tenant_name VARCHAR(50);
    UPDATE wallets SET tenant_name = map_uuid_to_tenant_name(tenant_id);
    ALTER TABLE wallets ALTER COLUMN tenant_name SET NOT NULL;
    ALTER TABLE wallets DROP COLUMN tenant_id;
    ALTER TABLE wallets RENAME COLUMN tenant_name TO tenant_id;
  END IF;
END $$;

DO $$ 
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
    ALTER TABLE users ADD COLUMN tenant_name VARCHAR(50);
    UPDATE users SET tenant_name = map_uuid_to_tenant_name(tenant_id);
    ALTER TABLE users ALTER COLUMN tenant_name SET NOT NULL;
    ALTER TABLE users DROP COLUMN tenant_id;
    ALTER TABLE users RENAME COLUMN tenant_name TO tenant_id;
  END IF;
END $$;

DO $$ 
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'assets') THEN
    ALTER TABLE assets ADD COLUMN tenant_name VARCHAR(50);
    UPDATE assets SET tenant_name = map_uuid_to_tenant_name(tenant_id);
    ALTER TABLE assets ALTER COLUMN tenant_name SET NOT NULL;
    ALTER TABLE assets DROP COLUMN tenant_id;
    ALTER TABLE assets RENAME COLUMN tenant_name TO tenant_id;
  END IF;
END $$;

-- Recriar índices que usavam tenant_id
DROP INDEX IF EXISTS ux_b3_raw_unique;
CREATE UNIQUE INDEX ux_b3_raw_unique 
  ON b3_raw_data_client (tenant_id, cpf, data_type, asset_type, period_start, period_end, page, payload_hash);

DROP INDEX IF EXISTS ix_b3_raw_tenant_period;
CREATE INDEX ix_b3_raw_tenant_period 
  ON b3_raw_data_client (tenant_id, cpf, period_start, period_end);

DROP INDEX IF EXISTS ux_b3_fetched_periods_unique;
CREATE UNIQUE INDEX ux_b3_fetched_periods_unique 
  ON b3_fetched_periods (tenant_id, cpf, data_type, asset_type, month_start);

DROP INDEX IF EXISTS ux_b3_norm_tx;
CREATE UNIQUE INDEX ux_b3_norm_tx 
  ON b3_normalized_transactions (tenant_id, cpf, asset_type, raw_id, sequence_in_raw, normalized_hash);

DROP INDEX IF EXISTS ux_b3_norm_pos;
CREATE UNIQUE INDEX ux_b3_norm_pos 
  ON b3_normalized_positions (tenant_id, cpf, asset_type, raw_id, sequence_in_raw, normalized_hash);

DROP INDEX IF EXISTS uq_b3_inconsistencies_dedupe;
CREATE UNIQUE INDEX uq_b3_inconsistencies_dedupe 
  ON b3_inconsistencies(tenant_id, cpf, ticker, type, dedupe_hash);

DROP INDEX IF EXISTS uq_ops_ledger_system_dedupe;
CREATE UNIQUE INDEX uq_ops_ledger_system_dedupe 
  ON b3_operations_ledger (tenant_id, cpf, ticker, operation_type, source, operation_date, reason_code, generated_by_inconsistency_id);

-- Limpar função temporária
DROP FUNCTION map_uuid_to_tenant_name(UUID);
