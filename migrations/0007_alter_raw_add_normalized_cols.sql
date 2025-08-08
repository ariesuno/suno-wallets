-- Adiciona colunas de status de normalização ao RAW
ALTER TABLE IF EXISTS b3_raw_data_client
  ADD COLUMN IF NOT EXISTS normalized_at timestamptz NULL,
  ADD COLUMN IF NOT EXISTS normalized_count integer NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS ix_b3_raw_normalized_at ON b3_raw_data_client (tenant_id, cpf, data_type, asset_type, normalized_at);

