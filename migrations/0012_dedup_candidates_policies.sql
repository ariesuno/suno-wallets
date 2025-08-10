-- Dedup candidates e policies (1.19)
DO $$ BEGIN
  CREATE TYPE dedup_status AS ENUM ('OPEN','AUTO_MERGED','OVERRIDDEN','IGNORED','CONFIRMED_MERGE');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS dedup_candidates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  primary_operation_id uuid NOT NULL,
  candidate_operation_id uuid NOT NULL,
  score numeric(6,3) NOT NULL,
  status dedup_status NOT NULL DEFAULT 'OPEN',
  rationale jsonb NOT NULL,
  pair_key text NOT NULL,
  dedupe_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  created_by text,
  resolved_at timestamptz,
  resolved_by text
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_dedup_pair
  ON dedup_candidates(tenant_id, cpf, primary_operation_id, candidate_operation_id);

CREATE INDEX IF NOT EXISTS ix_dedup_lookup
  ON dedup_candidates(tenant_id, cpf, status, score);

CREATE TABLE IF NOT EXISTS dedup_policies (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  prefer_source text NOT NULL DEFAULT 'B3_RAW',
  auto_merge_threshold numeric(6,3) NOT NULL DEFAULT 0.92,
  alert_threshold numeric(6,3) NOT NULL DEFAULT 0.70,
  date_tolerance_days smallint NOT NULL DEFAULT 2,
  quantity_tolerance_ratio numeric(6,4) NOT NULL DEFAULT 0.005,
  gross_tolerance_ratio numeric(6,4) NOT NULL DEFAULT 0.005,
  updated_at timestamptz NOT NULL DEFAULT now(),
  updated_by text,
  UNIQUE (tenant_id, cpf)
);


