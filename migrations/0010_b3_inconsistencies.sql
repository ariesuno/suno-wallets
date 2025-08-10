-- Inconsistencies table and types (PostgreSQL)
DO $$ BEGIN
  CREATE TYPE inconsistency_type AS ENUM (
    'OPENING_BALANCE_MISSING',
    'SELL_WITHOUT_BUY',
    'POSITION_TX_DIVERGENCE',
    'OTHER'
  );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE inconsistency_status AS ENUM ('OPEN','RESOLVED','OVERRIDDEN');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS b3_inconsistencies (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  ticker text NOT NULL,
  type inconsistency_type NOT NULL,
  status inconsistency_status NOT NULL DEFAULT 'OPEN',
  severity smallint NOT NULL DEFAULT 2,
  affected_period_start date,
  affected_period_end date,
  first_detected_at timestamptz NOT NULL DEFAULT now(),
  last_detected_at  timestamptz NOT NULL DEFAULT now(),
  sample_dates jsonb,
  details jsonb,
  dedupe_hash text NOT NULL,
  created_by_version text,
  created_by text,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_b3_inconsistencies_dedupe
  ON b3_inconsistencies(tenant_id, cpf, ticker, type, dedupe_hash);

CREATE INDEX IF NOT EXISTS ix_b3_inconsistencies_lookup
  ON b3_inconsistencies(tenant_id, cpf, status, type);


