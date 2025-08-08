-- Tabela normalizada de posições (v3)
CREATE TABLE IF NOT EXISTS b3_normalized_positions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
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
  normalized_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_b3_norm_pos
  ON b3_normalized_positions (tenant_id, cpf, asset_type, raw_id, sequence_in_raw, normalized_hash);

CREATE INDEX IF NOT EXISTS ix_b3_norm_pos_date
  ON b3_normalized_positions (tenant_id, cpf, reference_date);

CREATE INDEX IF NOT EXISTS ix_b3_norm_pos_ticker
  ON b3_normalized_positions (tenant_id, cpf, ticker);

