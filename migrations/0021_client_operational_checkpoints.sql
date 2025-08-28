-- Backoffice Admin — checkpoints operacionais (opcional) — para métricas persistidas por CPF
-- Esta tabela armazena métricas de performance e timestamps de operações por cliente
-- facilitando a construção do perfil 360 sem necessidade de agregações complexas

CREATE TABLE IF NOT EXISTS client_operational_checkpoints (
  tenant_id UUID NOT NULL,
  cpf VARCHAR(11) NOT NULL,
  
  -- Timestamps das últimas operações
  last_full_fetch_at TIMESTAMPTZ,
  last_incremental_at TIMESTAMPTZ,
  last_recon_scan_at TIMESTAMPTZ,
  last_auto_fix_at TIMESTAMPTZ,
  last_dedup_scan_at TIMESTAMPTZ,
  
  -- Métricas de performance (em segundos)
  avg_full_fetch_seconds NUMERIC(10,2),
  avg_incremental_seconds NUMERIC(10,2),
  avg_recon_scan_seconds NUMERIC(10,2),
  
  -- Contadores de operações executadas
  total_full_fetch_count INTEGER NOT NULL DEFAULT 0,
  total_incremental_count INTEGER NOT NULL DEFAULT 0,
  total_recon_scan_count INTEGER NOT NULL DEFAULT 0,
  total_auto_fix_count INTEGER NOT NULL DEFAULT 0,
  total_dedup_scan_count INTEGER NOT NULL DEFAULT 0,
  
  -- Contadores de sucessos/erros
  full_fetch_success_count INTEGER NOT NULL DEFAULT 0,
  full_fetch_error_count INTEGER NOT NULL DEFAULT 0,
  incremental_success_count INTEGER NOT NULL DEFAULT 0,
  incremental_error_count INTEGER NOT NULL DEFAULT 0,
  
  -- Últimas ações administrativas
  last_admin_action_at TIMESTAMPTZ,
  last_admin_action_type TEXT,
  last_admin_action_status TEXT,
  
  -- Metadados adicionais
  data_source_mode TEXT DEFAULT 'HYBRID',
  policy_last_updated_at TIMESTAMPTZ,
  
  -- Controle de atualização
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  
  PRIMARY KEY (tenant_id, cpf)
);

-- Índice para consultas por última atividade
CREATE INDEX IF NOT EXISTS ix_client_checkpoints_activity
  ON client_operational_checkpoints (tenant_id, last_admin_action_at DESC);

-- Índice para consultas por tipo de ação
CREATE INDEX IF NOT EXISTS ix_client_checkpoints_action_type
  ON client_operational_checkpoints (tenant_id, last_admin_action_type, last_admin_action_at DESC);

-- Função para atualizar automaticamente o updated_at
CREATE OR REPLACE FUNCTION update_client_checkpoints_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger para atualizar automaticamente o updated_at
DROP TRIGGER IF EXISTS tr_client_checkpoints_updated_at ON client_operational_checkpoints;
CREATE TRIGGER tr_client_checkpoints_updated_at
    BEFORE UPDATE ON client_operational_checkpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_client_checkpoints_updated_at();
