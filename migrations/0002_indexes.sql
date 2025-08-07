-- Índices recomendados para performance
-- Carteiras
CREATE INDEX IF NOT EXISTS idx_wallets_tenant_owner ON wallets(tenant_id, owner_id);
CREATE INDEX IF NOT EXISTS idx_wallets_tenant_status ON wallets(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_wallets_tenant_type ON wallets(tenant_id, type);
CREATE INDEX IF NOT EXISTS idx_wallets_owner_default ON wallets(owner_id, is_default) WHERE is_default = true;


