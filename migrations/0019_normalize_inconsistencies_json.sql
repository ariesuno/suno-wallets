-- Normalizar campos JSON mais utilizados da tabela b3_inconsistencies
-- Adicionar colunas específicas para dados frequentemente acessados

-- Adicionar novas colunas normalizadas
ALTER TABLE b3_inconsistencies 
ADD COLUMN IF NOT EXISTS first_transaction_date DATE,
ADD COLUMN IF NOT EXISTS first_transaction_side VARCHAR(4) CHECK (first_transaction_side IN ('BUY', 'SELL')),
ADD COLUMN IF NOT EXISTS min_cumulative_quantity DECIMAL(15,6),
ADD COLUMN IF NOT EXISTS first_position_date DATE,
ADD COLUMN IF NOT EXISTS position_quantity DECIMAL(15,6),
ADD COLUMN IF NOT EXISTS net_transactions_quantity DECIMAL(15,6);

-- Criar índices para melhorar performance das consultas
CREATE INDEX IF NOT EXISTS ix_b3_inconsistencies_first_tx_date 
  ON b3_inconsistencies(tenant_id, cpf, first_transaction_date);

CREATE INDEX IF NOT EXISTS ix_b3_inconsistencies_first_position_date 
  ON b3_inconsistencies(tenant_id, cpf, first_position_date);

-- Atualizar registros existentes extraindo dados do JSON
UPDATE b3_inconsistencies 
SET 
  first_transaction_date = CASE 
    WHEN sample_dates->>'first_tx_date' IS NOT NULL 
    THEN (sample_dates->>'first_tx_date')::DATE 
    ELSE NULL 
  END,
  first_transaction_side = CASE 
    WHEN sample_dates->>'first_tx_side' IS NOT NULL 
    THEN UPPER(sample_dates->>'first_tx_side')
    ELSE NULL 
  END,
  min_cumulative_quantity = CASE 
    WHEN sample_dates->>'min_cum_qty' IS NOT NULL 
    THEN (sample_dates->>'min_cum_qty')::DECIMAL(15,6)
    ELSE NULL 
  END,
  first_position_date = CASE 
    WHEN sample_dates->>'first_position_date' IS NOT NULL 
    THEN (sample_dates->>'first_position_date')::DATE 
    ELSE NULL 
  END,
  position_quantity = CASE 
    WHEN details->>'position_qty' IS NOT NULL 
    THEN (details->>'position_qty')::DECIMAL(15,6)
    ELSE NULL 
  END,
  net_transactions_quantity = CASE 
    WHEN details->>'net_tx_until_first_position' IS NOT NULL 
    THEN (details->>'net_tx_until_first_position')::DECIMAL(15,6)
    ELSE NULL 
  END
WHERE 
  (sample_dates IS NOT NULL AND sample_dates != 'null'::jsonb) 
  OR 
  (details IS NOT NULL AND details != 'null'::jsonb);

-- Comentários para documentação
COMMENT ON COLUMN b3_inconsistencies.first_transaction_date IS 'Data da primeira transação extraída de sample_dates.first_tx_date';
COMMENT ON COLUMN b3_inconsistencies.first_transaction_side IS 'Lado da primeira transação (BUY/SELL) extraído de sample_dates.first_tx_side';
COMMENT ON COLUMN b3_inconsistencies.min_cumulative_quantity IS 'Quantidade cumulativa mínima extraída de sample_dates.min_cum_qty';
COMMENT ON COLUMN b3_inconsistencies.first_position_date IS 'Data da primeira posição extraída de sample_dates.first_position_date';
COMMENT ON COLUMN b3_inconsistencies.position_quantity IS 'Quantidade da posição extraída de details.position_qty';
COMMENT ON COLUMN b3_inconsistencies.net_transactions_quantity IS 'Transações líquidas extraídas de details.net_tx_until_first_position';
