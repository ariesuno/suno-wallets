-- Estado de sincronismo incremental da B3 por cliente
CREATE TABLE IF NOT EXISTS b3_sync_state (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  cpf varchar(11) NOT NULL,
  is_active boolean NOT NULL DEFAULT true,
  last_tx_sync_at timestamptz NULL,
  last_pos_sync_at timestamptz NULL,
  last_checked_at timestamptz NULL,
  last_result varchar NULL,
  last_error text NULL,
  needs_reprocess boolean NOT NULL DEFAULT false,
  failure_count integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_b3_sync_state_tenant_cpf ON b3_sync_state (tenant_id, cpf);
CREATE INDEX IF NOT EXISTS ix_b3_sync_state_active ON b3_sync_state (tenant_id, is_active);
CREATE INDEX IF NOT EXISTS ix_b3_sync_state_reprocess ON b3_sync_state (tenant_id, needs_reprocess);

-- Trigger simples para updated_at
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_b3_sync_state_updated_at ON b3_sync_state;
CREATE TRIGGER trg_b3_sync_state_updated_at BEFORE UPDATE ON b3_sync_state
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

