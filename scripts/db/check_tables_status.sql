-- Script para verificar status de todas as tabelas
-- Mostra quantos registros cada tabela possui

SELECT 
    'PRINCIPAIS TABELAS B3' as categoria,
    (SELECT COUNT(*) FROM b3_raw_data_client) as raw_data,
    (SELECT COUNT(*) FROM b3_normalized_transactions) as norm_transactions,
    (SELECT COUNT(*) FROM b3_normalized_positions) as norm_positions,
    (SELECT COUNT(*) FROM b3_sync_state) as sync_state,
    (SELECT COUNT(*) FROM b3_fetched_periods) as fetched_periods

UNION ALL

SELECT 
    'RECONCILIAÇÃO E OPS' as categoria,
    (SELECT COUNT(*) FROM b3_inconsistencies) as inconsistencies,
    (SELECT COUNT(*) FROM b3_operations_ledger) as operations_ledger,
    (SELECT COUNT(*) FROM dedup_candidates) as dedup_candidates,
    (SELECT COUNT(*) FROM dedup_policies) as dedup_policies,
    0 as placeholder

UNION ALL

SELECT 
    'ARCHIVE E AUDITORIA' as categoria,
    (SELECT COUNT(*) FROM b3_raw_data_client_archive) as raw_archive,
    (SELECT COUNT(*) FROM b3_normalized_transactions_archive) as tx_archive,
    (SELECT COUNT(*) FROM b3_normalized_positions_archive) as pos_archive,
    (SELECT COUNT(*) FROM admin_action_audit) as admin_audit,
    (SELECT COUNT(*) FROM client_data_source_policy_audit) as policy_audit

UNION ALL

SELECT 
    'CONFIGURAÇÕES' as categoria,
    (SELECT COUNT(*) FROM client_data_source_policy) as data_policies,
    (SELECT COUNT(*) FROM wallets) as wallets,
    0 as col3,
    0 as col4,
    0 as col5;

-- Resumo total
SELECT 
    '🎯 RESUMO GERAL' as info,
    COUNT(*) as total_tabelas
FROM pg_tables 
WHERE schemaname = 'public' 
AND tablename NOT LIKE 'pg_%';

-- Verificação de integridade das principais entidades para um CPF específico
-- (descomente e ajuste o CPF se necessário)
-- SELECT 
--     '📊 INTEGRIDADE CPF 34551207845' as info,
--     (SELECT COUNT(*) FROM b3_raw_data_client WHERE cpf = '34551207845') as raw_records,
--     (SELECT COUNT(*) FROM b3_normalized_transactions WHERE cpf = '34551207845') as transactions,
--     (SELECT COUNT(*) FROM b3_sync_state WHERE cpf = '34551207845') as sync_records;
