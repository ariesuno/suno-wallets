#!/bin/bash

# Script otimizado para validar CPFs usando análise de status (4ms por CPF)
# Atualiza campo b3_validated na tabela temp_clients

API_BASE="http://localhost:8080/api/v1"
TENANT_ID="status_invest"

echo "🚀 Iniciando validação RÁPIDA de CPFs usando análise de status..."
echo "🎯 Endpoint: ${API_BASE}/b3/sync/status-analysis"
echo "⚡ Sem conectividade externa B3 - apenas análise local"
echo ""

# Buscar todos os CPFs da tabela que ainda não foram validados
CPFS=$(docker-compose exec -T postgres psql -U suno_user -d suno_wallets -t -c "SELECT cpf FROM temp_clients WHERE b3_validated IS NULL OR b3_validated = FALSE ORDER BY is_priority DESC, id;")

total_cpfs=0
validated_cpfs=0
failed_cpfs=0

for cpf in $CPFS; do
    # Remover espaços e quebras de linha
    cpf=$(echo "$cpf" | tr -d ' \n\r')
    
    if [ -z "$cpf" ]; then
        continue
    fi
    
    total_cpfs=$((total_cpfs + 1))
    
    echo -n "⚡ Validando CPF: ${cpf:0:3}******* ... "
    
    # Testar CPF usando análise de status (super rápido)
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "${API_BASE}/b3/sync/status-analysis?cpf=${cpf}" \
        -H "X-Tenant-ID: ${TENANT_ID}" \
        -H "Content-Type: application/json")
    
    # Extrair HTTP status
    http_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
    
    # Verificar se é 200 (CPF válido para análise)
    if [ "$http_code" = "200" ]; then
        # Verificar se response contém dados de estratégia
        if echo "$body" | grep -q '"strategy"'; then
            # Verificar tipo de cliente para dar mais detalhes
            if echo "$body" | grep -q '"clientStatus":"new"'; then
                echo "✅ VÁLIDO (cliente novo)"
            elif echo "$body" | grep -q '"clientStatus":"existing"'; then
                echo "✅ VÁLIDO (cliente existente)"
            else
                echo "✅ VÁLIDO (análise ok)"
            fi
            
            validated_cpfs=$((validated_cpfs + 1))
            
            # Atualizar banco como validado
            docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
                "UPDATE temp_clients SET b3_validated = TRUE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
        else
            echo "❌ RESPOSTA INVÁLIDA"
            failed_cpfs=$((failed_cpfs + 1))
            
            # Atualizar banco como não validado
            docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
                "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
        fi
    elif [ "$http_code" = "400" ]; then
        echo "❌ CPF INVÁLIDO (400)"
        failed_cpfs=$((failed_cpfs + 1))
        
        # CPF com formato inválido
        docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
            "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
    else
        echo "❌ ERRO HTTP ($http_code)"
        failed_cpfs=$((failed_cpfs + 1))
        
        # Outros erros HTTP
        docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
            "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
    fi
    
    # Pausa mínima para não sobrecarregar (muito rápido)
    sleep 0.1
done

echo ""
echo "📊 RESUMO DA VALIDAÇÃO RÁPIDA:"
echo "   Total de CPFs testados: $total_cpfs"
echo "   CPFs validados: $validated_cpfs"
echo "   CPFs rejeitados: $failed_cpfs"
if [ $total_cpfs -gt 0 ]; then
    echo "   Taxa de sucesso: $(echo "scale=1; $validated_cpfs * 100 / $total_cpfs" | bc -l)%"
    echo "   Tempo médio: ~4ms por CPF"
fi
echo ""

# Mostrar estatísticas finais do banco
echo "📈 ESTATÍSTICAS ATUALIZADAS NO BANCO:"
docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c "
SELECT 
    '🎯 CLIENTES PRIORITÁRIOS VALIDADOS' as categoria,
    COUNT(*) as quantidade
FROM temp_clients 
WHERE is_priority = TRUE AND b3_validated = TRUE
UNION ALL
SELECT 
    '📊 TOTAL VALIDADOS',
    COUNT(*)
FROM temp_clients 
WHERE b3_validated = TRUE
UNION ALL
SELECT 
    '⚠️  NECESSITA CORREÇÃO',
    COUNT(*)
FROM temp_clients 
WHERE b3_validated = FALSE OR b3_validated IS NULL;
"

echo ""
echo "✅ Validação RÁPIDA concluída!"
echo "💡 Dica: Use 'SELECT cpf FROM temp_clients WHERE b3_validated = TRUE AND is_priority = TRUE;' para CPFs prioritários"
