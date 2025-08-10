-- Tabelas de arquivo para reset seguro (archive)
CREATE TABLE IF NOT EXISTS b3_raw_data_client_archive (
  id uuid NOT NULL,
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
  fetched_at timestamptz,
  retry_count integer,
  request_id uuid,
  normalized_at timestamptz,
  normalized_count integer,
  archived_at timestamptz NOT NULL,
  archived_by text NOT NULL
);

CREATE TABLE IF NOT EXISTS b3_normalized_transactions_archive (
  id uuid NOT NULL,
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  asset_type varchar NOT NULL,
  source_version varchar NOT NULL,
  raw_id uuid NOT NULL,
  sequence_in_raw integer NOT NULL,
  trade_id text NULL,
  broker_code text NULL,
  trade_date date NOT NULL,
  settlement_date date NULL,
  ticker text NOT NULL,
  isin text NULL,
  side varchar NOT NULL,
  quantity numeric(28,10) NOT NULL,
  price numeric(28,10) NOT NULL,
  gross_value numeric(28,10) NULL,
  currency varchar(8) NULL,
  extra_json jsonb NULL,
  normalized_hash varchar(64) NOT NULL,
  normalized_at timestamptz NOT NULL,
  archived_at timestamptz NOT NULL,
  archived_by text NOT NULL
);

CREATE TABLE IF NOT EXISTS b3_normalized_positions_archive (
  id uuid NOT NULL,
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  asset_type varchar NOT NULL,
  source_version varchar NOT NULL,
  raw_id uuid NOT NULL,
  sequence_in_raw integer NOT NULL,
  reference_date date NOT NULL,
  ticker text NOT NULL,
  isin text NULL,
  quantity numeric(28,10) NOT NULL,
  avg_price numeric(28,10) NULL,
  position_value numeric(28,10) NULL,
  currency varchar(8) NULL,
  extra_json jsonb NULL,
  normalized_hash varchar(64) NOT NULL,
  normalized_at timestamptz NOT NULL,
  archived_at timestamptz NOT NULL,
  archived_by text NOT NULL
);


