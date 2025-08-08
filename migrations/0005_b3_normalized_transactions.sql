-- Tabela normalizada de transações (v2)
CREATE TABLE IF NOT EXISTS b3_normalized_transactions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
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
  normalized_at timestamptz NOT NULL DEFAULT now()
);

-- Índices/constraints
CREATE UNIQUE INDEX IF NOT EXISTS ux_b3_norm_tx
  ON b3_normalized_transactions (tenant_id, cpf, asset_type, raw_id, sequence_in_raw, normalized_hash);

CREATE INDEX IF NOT EXISTS ix_b3_norm_tx_date
  ON b3_normalized_transactions (tenant_id, cpf, trade_date);

CREATE INDEX IF NOT EXISTS ix_b3_norm_tx_ticker
  ON b3_normalized_transactions (tenant_id, cpf, ticker);

