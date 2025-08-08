-- Tabela de persistência do RAW da B3
CREATE TABLE IF NOT EXISTS b3_raw_data_client (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  data_type varchar NOT NULL,
  asset_type varchar NOT NULL,
  period_start date NOT NULL,
  period_end date NOT NULL,
  page integer NOT NULL,
  payload_json jsonb NOT NULL,
  payload_hash varchar(64) NOT NULL,
  source_version varchar NOT NULL,
  endpoint_path text NOT NULL,
  http_status integer NOT NULL,
  fetched_at timestamptz NOT NULL DEFAULT now(),
  retry_count integer NOT NULL DEFAULT 0,
  request_id uuid NOT NULL
);

-- Índices e constraints
CREATE UNIQUE INDEX IF NOT EXISTS ux_b3_raw_unique
  ON b3_raw_data_client (tenant_id, cpf, data_type, asset_type, period_start, period_end, page, payload_hash);

CREATE INDEX IF NOT EXISTS ix_b3_raw_tenant_period
  ON b3_raw_data_client (tenant_id, cpf, period_start, period_end);

