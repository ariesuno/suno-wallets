-- Ledger unificado de operações (1.18)
DO $$ BEGIN
  CREATE TYPE operation_source AS ENUM ('B3_RAW','SYSTEM_SYNTHETIC','USER_MANUAL');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE operation_type AS ENUM ('BUY','SELL','OPENING_BALANCE','ADJUSTMENT','FRACTION_ADJUSTMENT','CASH_ADJUSTMENT');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE price_confidence AS ENUM ('MARKET_CLOSE','MIRRORED_SELL','DERIVED_POSITION','UNKNOWN');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS b3_operations_ledger (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  ticker text NOT NULL,
  asset_type text NOT NULL DEFAULT 'EQUITY',
  operation_date date NOT NULL,
  operation_type operation_type NOT NULL,
  source operation_source NOT NULL,
  quantity numeric(28,10) NOT NULL,
  unit_price numeric(28,10),
  currency text NOT NULL DEFAULT 'BRL',

  reason_code text,
  price_confidence price_confidence NOT NULL DEFAULT 'UNKNOWN',
  generated_by_inconsistency_id uuid,
  supersedes_operation_id uuid,
  superseded_by_operation_id uuid,

  is_active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  created_by text,
  updated_at timestamptz NOT NULL DEFAULT now(),
  updated_by text
);

CREATE INDEX IF NOT EXISTS ix_ops_ledger_lookup
  ON b3_operations_ledger (tenant_id, cpf, ticker, operation_date);

CREATE UNIQUE INDEX IF NOT EXISTS uq_ops_ledger_system_dedupe
ON b3_operations_ledger (tenant_id, cpf, ticker, operation_type, source, operation_date, reason_code, generated_by_inconsistency_id);


