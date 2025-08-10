-- Backoffice Admin — auditoria de ações (1.22)

DO $$ BEGIN
  CREATE TYPE admin_action_type AS ENUM (
    'B3_FULL_FETCH','B3_INCREMENTAL_FETCH','RECON_SCAN','AUTO_FIX','DEDUPE_SCAN','DEDUPE_RESOLVE',
    'POLICY_UPDATE','LEDGER_EXPORT','CACHE_INVALIDATE','CLIENT_RESET','CLIENT_ZERO_AND_REFETCH'
  );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE admin_action_status AS ENUM ('REQUESTED','CONFIRMED','RUNNING','SUCCESS','ERROR','SKIPPED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS admin_action_audit (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  action admin_action_type NOT NULL,
  status admin_action_status NOT NULL,
  requested_by text NOT NULL,
  confirmed_by text,
  confirm_token text,
  confirm_deadline timestamptz,
  request_payload jsonb NOT NULL,
  result_payload jsonb,
  error_message text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_admin_action_lookup
  ON admin_action_audit (tenant_id, cpf, action, status, created_at DESC);


