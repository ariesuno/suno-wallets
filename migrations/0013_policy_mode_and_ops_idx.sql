-- Add policy mode and index for ops ledger source/active (1.19)
DO $$ BEGIN
  ALTER TABLE dedup_policies ADD COLUMN IF NOT EXISTS mode TEXT NOT NULL DEFAULT 'HYBRID';
EXCEPTION WHEN undefined_table THEN NULL; END $$;

CREATE INDEX IF NOT EXISTS ix_ops_ledger_source_active
  ON b3_operations_ledger (tenant_id, cpf, source, is_active);


