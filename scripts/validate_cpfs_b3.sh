#!/bin/bash

# Script para validar CPFs usando análise de status (sem conexão B3 externa)
# Atualiza campo b3_validated na tabela temp_clients

API_BASE="http://localhost:8080/api/v1"
TENANT_ID="status_invest"

echo "🔍 Iniciando validação de CPFs usando análise de status..."
echo "🎯 Endpoint: ${API_BASE}/b3/sync/status-analysis"
echo ""

# Buscar todos os CPFs da tabela
CPFS=$(docker-compose exec -T postgres psql -U suno_user -d suno_wallets -t -c "SELECT cpf FROM temp_clients ORDER BY id;")

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
    
    echo -n "🧪 Analisando CPF: ${cpf:0:3}******* ... "
    
    # Testar CPF usando análise de status (sem conexão externa B3)
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
            echo "✅ VÁLIDO (análise concluída)"
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
    
    # Pequena pausa para não sobrecarregar API
    sleep 0.2
done

echo ""
echo "📊 RESUMO DA VALIDAÇÃO:"
echo "   Total de CPFs: $total_cpfs"
echo "   CPFs validados: $validated_cpfs"
echo "   CPFs rejeitados: $failed_cpfs"
echo "   Taxa de sucesso: $(echo "scale=1; $validated_cpfs * 100 / $total_cpfs" | bc -l)%"
echo ""

# Mostrar estatísticas finais do banco
echo "📈 ESTATÍSTICAS ATUALIZADAS NO BANCO:"
docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c "
SELECT 
    COUNT(*) as total_cpfs,
    SUM(CASE WHEN b3_validated = TRUE THEN 1 ELSE 0 END) as validados_b3,
    SUM(CASE WHEN b3_validated = FALSE THEN 1 ELSE 0 END) as rejeitados_b3,
    SUM(CASE WHEN b3_validated IS NULL THEN 1 ELSE 0 END) as nao_testados,
    SUM(CASE WHEN is_priority AND b3_validated = TRUE THEN 1 ELSE 0 END) as prioritarios_validados,
    ROUND(SUM(CASE WHEN b3_validated = TRUE THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 1) as taxa_validacao
FROM temp_clients;
"

echo ""
echo "✅ Validação concluída! Use apenas os CPFs com b3_validated = TRUE para testes."
