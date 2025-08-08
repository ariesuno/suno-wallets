-- Tabela de controle de períodos já buscados
CREATE TABLE IF NOT EXISTS b3_fetched_periods (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  data_type varchar NOT NULL,
  asset_type varchar NOT NULL,
  month_start date NOT NULL,
  month_end date NOT NULL,
  pages integer NOT NULL,
  completed boolean NOT NULL DEFAULT false,
  last_fetched_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_b3_fetched_periods_unique
  ON b3_fetched_periods (tenant_id, cpf, data_type, asset_type, month_start);

CREATE INDEX IF NOT EXISTS ix_b3_fetched_periods_lookup
  ON b3_fetched_periods (tenant_id, cpf, data_type, asset_type, month_start);

