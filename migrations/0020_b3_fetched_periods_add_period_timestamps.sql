-- Adicionar campos period_start/period_end com timezone para suporte ao mês corrente
-- e cálculos precisos com timezone America/Sao_Paulo

ALTER TABLE b3_fetched_periods 
ADD COLUMN period_start TIMESTAMPTZ,
ADD COLUMN period_end TIMESTAMPTZ;

-- Para registros existentes, derivar period_start/period_end a partir de month_start/month_end
-- usando timezone America/Sao_Paulo
UPDATE b3_fetched_periods 
SET 
    period_start = timezone('America/Sao_Paulo', month_start::timestamp),
    period_end = timezone('America/Sao_Paulo', (month_end::timestamp + interval '23 hours 59 minutes 59.999999 seconds'))
WHERE period_start IS NULL;

-- Criar coluna computada period_month para compatibilidade e performance
ALTER TABLE b3_fetched_periods 
ADD COLUMN period_month DATE GENERATED ALWAYS AS (date_trunc('month', period_start AT TIME ZONE 'America/Sao_Paulo')::date) STORED;

-- Índices para performance nas novas colunas
CREATE INDEX IF NOT EXISTS ix_b3_fetched_periods_period_start 
    ON b3_fetched_periods (tenant_id, cpf, data_type, asset_type, period_start);

CREATE INDEX IF NOT EXISTS ix_b3_fetched_periods_period_range 
    ON b3_fetched_periods (period_start, period_end);

CREATE UNIQUE INDEX IF NOT EXISTS ux_b3_fetched_periods_unique_period 
    ON b3_fetched_periods (tenant_id, cpf, data_type, asset_type, period_month);

-- Comentário explicativo
COMMENT ON COLUMN b3_fetched_periods.period_start IS 'Início preciso do período com timezone America/Sao_Paulo';
COMMENT ON COLUMN b3_fetched_periods.period_end IS 'Fim preciso do período com timezone America/Sao_Paulo - para mês corrente usa D-1 23:59:59.999999';
COMMENT ON COLUMN b3_fetched_periods.period_month IS 'Mês do período (YYYY-MM-01) gerado automaticamente a partir de period_start';
