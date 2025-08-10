-- Índices auxiliares para Timeline & Summary (1.20)

-- Para ordenação e filtros da timeline por tenant/cpf/ticker/data
CREATE INDEX IF NOT EXISTS ix_ops_ledger_timeline
  ON b3_operations_ledger (tenant_id, cpf, ticker, operation_date DESC, id DESC);

-- Para filtros por fonte e ativo
CREATE INDEX IF NOT EXISTS ix_ops_ledger_source_active
  ON b3_operations_ledger (tenant_id, cpf, source, is_active);

-- Para agregações de inconsistências por status e tipo
CREATE INDEX IF NOT EXISTS ix_b3_inconsistencies_status_type
  ON b3_inconsistencies (tenant_id, cpf, status, type);


