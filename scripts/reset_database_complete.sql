-- Script completo para reset do banco de dados
-- Inclui liberação de locks e limpeza de todas as tabelas

-- 1. Liberar todos os advisory locks ativos
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN 
        SELECT DISTINCT pid 
        FROM pg_locks 
        WHERE locktype = 'advisory' 
        AND granted = true
    LOOP
        PERFORM pg_terminate_backend(r.pid);
    END LOOP;
END $$;

-- 2. Aguardar um momento para os locks serem liberados
SELECT pg_sleep(1);

-- 3. Truncar todas as tabelas na ordem correta (respeitando foreign keys)
TRUNCATE TABLE b3_inconsistencies CASCADE;
TRUNCATE TABLE b3_operations_ledger CASCADE;
TRUNCATE TABLE b3_sync_state CASCADE;
TRUNCATE TABLE b3_normalized_positions CASCADE;
TRUNCATE TABLE b3_normalized_transactions CASCADE;
TRUNCATE TABLE b3_normalized_positions_archive CASCADE;
TRUNCATE TABLE b3_normalized_transactions_archive CASCADE;
TRUNCATE TABLE b3_raw_data_client_archive CASCADE;
TRUNCATE TABLE b3_fetched_periods CASCADE;
TRUNCATE TABLE b3_raw_data_client CASCADE;
TRUNCATE TABLE dedup_candidates CASCADE;
TRUNCATE TABLE admin_action_audit CASCADE;
TRUNCATE TABLE client_data_source_policy_audit CASCADE;

-- 4. Reiniciar sequences se necessário
-- (As tabelas usam UUID, então não há sequences para reiniciar)

-- 5. Verificar se limpeza foi bem-sucedida
SELECT 
    'b3_raw_data_client' as tabela, COUNT(*) as registros FROM b3_raw_data_client
UNION ALL
SELECT 
    'b3_normalized_transactions' as tabela, COUNT(*) as registros FROM b3_normalized_transactions
UNION ALL
SELECT 
    'b3_normalized_positions' as tabela, COUNT(*) as registros FROM b3_normalized_positions
UNION ALL
SELECT 
    'b3_sync_state' as tabela, COUNT(*) as registros FROM b3_sync_state
UNION ALL
SELECT 
    'b3_fetched_periods' as tabela, COUNT(*) as registros FROM b3_fetched_periods;

