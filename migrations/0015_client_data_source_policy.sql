-- Client Data Source Policy (1.21)

DO $$ BEGIN
  CREATE TYPE client_policy_mode AS ENUM ('B3_ONLY','MANUAL_ONLY','HYBRID');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS client_data_source_policy (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  mode client_policy_mode NOT NULL,
  reason text,
  effective_from timestamptz DEFAULT now(),
  effective_to timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  created_by text,
  updated_at timestamptz NOT NULL DEFAULT now(),
  updated_by text,
  UNIQUE (tenant_id, cpf)
);

CREATE TABLE IF NOT EXISTS client_data_source_policy_audit (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  old_mode client_policy_mode,
  new_mode client_policy_mode NOT NULL,
  changed_by text,
  reason text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_client_policy_lookup ON client_data_source_policy (tenant_id, cpf);
CREATE INDEX IF NOT EXISTS ix_client_policy_audit ON client_data_source_policy_audit (tenant_id, cpf, created_at DESC);


