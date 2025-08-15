-- Script para TRUNCATE de todas as tabelas EXCETO temp_clients
-- Mantém apenas os CPFs de teste na tabela temp_clients
-- ⚠️  ATENÇÃO: Este script apaga QUASE TODOS OS DADOS do banco!

BEGIN;

-- Desabilitar verificações de foreign key temporariamente
SET session_replication_role = replica;

-- ===========================================
-- 1. TABELAS DE AUDITORIA E LOGS (sem dependências)
-- ===========================================
TRUNCATE TABLE admin_action_audit RESTART IDENTITY CASCADE;
TRUNCATE TABLE client_data_source_policy_audit RESTART IDENTITY CASCADE;

-- ===========================================
-- 2. TABELAS DERIVADAS/ARCHIVE (dependem das principais)
-- ===========================================
TRUNCATE TABLE b3_raw_data_client_archive RESTART IDENTITY CASCADE;
TRUNCATE TABLE b3_normalized_transactions_archive RESTART IDENTITY CASCADE;
TRUNCATE TABLE b3_normalized_positions_archive RESTART IDENTITY CASCADE;

-- ===========================================
-- 3. TABELAS DE RECONCILIAÇÃO E OPERAÇÕES
-- ===========================================
TRUNCATE TABLE b3_inconsistencies RESTART IDENTITY CASCADE;
TRUNCATE TABLE b3_operations_ledger RESTART IDENTITY CASCADE;
TRUNCATE TABLE dedup_candidates RESTART IDENTITY CASCADE;

-- ===========================================
-- 4. TABELAS NORMALIZADAS (dependem de raw)
-- ===========================================
TRUNCATE TABLE b3_normalized_transactions RESTART IDENTITY CASCADE;
TRUNCATE TABLE b3_normalized_positions RESTART IDENTITY CASCADE;

-- ===========================================
-- 5. TABELAS DE CONTROLE E ESTADO
-- ===========================================
TRUNCATE TABLE b3_fetched_periods RESTART IDENTITY CASCADE;
TRUNCATE TABLE b3_sync_state RESTART IDENTITY CASCADE;

-- ===========================================
-- 6. TABELAS BASE/RAW (base da pirâmide)
-- ===========================================
TRUNCATE TABLE b3_raw_data_client RESTART IDENTITY CASCADE;

-- ===========================================
-- 7. TABELAS DE CONFIGURAÇÃO E POLÍTICAS
-- ===========================================
TRUNCATE TABLE client_data_source_policy RESTART IDENTITY CASCADE;
TRUNCATE TABLE dedup_policies RESTART IDENTITY CASCADE;

-- ===========================================
-- 8. TABELAS DE ENTIDADES PRINCIPAIS
-- ===========================================
TRUNCATE TABLE wallets RESTART IDENTITY CASCADE;

-- ===========================================
-- PRESERVANDO temp_clients - NÃO TRUNCAR!
-- ===========================================
-- ✅ temp_clients será mantida intacta com os CPFs de teste

-- Reabilitar verificações de foreign key
SET session_replication_role = DEFAULT;

COMMIT;

-- ===========================================
-- VERIFICAÇÃO PÓS-TRUNCATE
-- ===========================================
DO $$
DECLARE
    table_name text;
    row_count integer;
    total_tables integer := 0;
    empty_tables integer := 0;
    temp_clients_count integer;
BEGIN
    RAISE NOTICE '=== VERIFICAÇÃO PÓS-TRUNCATE (PRESERVANDO temp_clients) ===';
    
    -- Verificar temp_clients separadamente
    SELECT COUNT(*) INTO temp_clients_count FROM temp_clients;
    RAISE NOTICE '🔒 temp_clients PRESERVADA: % CPFs mantidos', temp_clients_count;
    RAISE NOTICE '';
    
    FOR table_name IN 
        SELECT tablename 
        FROM pg_tables 
        WHERE schemaname = 'public' 
        AND tablename NOT LIKE 'pg_%'
        AND tablename != 'temp_clients'  -- Excluir temp_clients da verificação
        ORDER BY tablename
    LOOP
        EXECUTE format('SELECT COUNT(*) FROM %I', table_name) INTO row_count;
        total_tables := total_tables + 1;
        
        IF row_count = 0 THEN
            empty_tables := empty_tables + 1;
            RAISE NOTICE '✅ % está vazia (% registros)', table_name, row_count;
        ELSE
            RAISE NOTICE '⚠️  % ainda tem dados (% registros)', table_name, row_count;
        END IF;
    END LOOP;
    
    RAISE NOTICE '';
    RAISE NOTICE '📊 RESUMO: %/% tabelas foram esvaziadas (temp_clients preservada)', empty_tables, total_tables;
    RAISE NOTICE '🔒 temp_clients mantida com % CPFs para teste', temp_clients_count;
    
    IF empty_tables = total_tables THEN
        RAISE NOTICE '🎉 SUCESSO: Banco zerado, temp_clients preservada!';
    ELSE
        RAISE NOTICE '⚠️  ATENÇÃO: % tabelas ainda contêm dados', (total_tables - empty_tables);
    END IF;
END $$;

-- Mostrar CPFs que serão processados
SELECT 
    '🎯 CPFs PARA PROCESSAMENTO ASYNC:' as info,
    cpf,
    CASE WHEN is_priority THEN '⭐ Priority' ELSE '📝 Normal' END as priority,
    CASE WHEN is_paying THEN '💰 Paying' ELSE '🆓 Free' END as status
FROM temp_clients 
ORDER BY is_priority DESC, cpf;
