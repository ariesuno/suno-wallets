#!/bin/bash

# Script para validar TODOS os CPFs da tabela temp_clients usando conexão B3 REAL
# Usa o endpoint /b3/test/connection que agora está funcionando
# Atualiza campo b3_validated conforme resultado

API_BASE="http://localhost:8080/api/v1"
TENANT_ID="status_invest"
YEAR_MONTH="2025-08"  # Mês atual para teste

echo "🔐 VALIDAÇÃO B3 REAL - TODOS OS CPFs"
echo "🎯 Endpoint: ${API_BASE}/b3/test/connection"
echo "📅 Período de teste: ${YEAR_MONTH}"
echo "🔄 Atualizando campo b3_validated automaticamente"
echo ""

# Buscar todos os CPFs da tabela temp_clients
echo "📋 Buscando CPFs da tabela temp_clients..."
CPFS=$(docker-compose exec -T postgres psql -U suno_user -d suno_wallets -t -c "SELECT cpf FROM temp_clients ORDER BY id;")

if [ -z "$CPFS" ]; then
    echo "❌ Nenhum CPF encontrado na tabela temp_clients"
    exit 1
fi

total_cpfs=0
validated_cpfs=0
failed_cpfs=0
timeout_cpfs=0

echo "🚀 Iniciando validação..."
echo ""

for cpf in $CPFS; do
    cpf=$(echo "$cpf" | tr -d ' \n\r') # Limpar espaços e quebras de linha
    if [ -z "$cpf" ]; then continue; fi
    
    total_cpfs=$((total_cpfs + 1))
    echo -n "🧪 Testando CPF #${total_cpfs}: ${cpf:0:3}******* ... "

    # Fazer requisição ao endpoint de teste B3
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" --max-time 45 \
        -X GET "${API_BASE}/b3/test/connection?cpf=${cpf}&yearMonth=${YEAR_MONTH}" \
        -H "X-Tenant-ID: ${TENANT_ID}" \
        -H "Content-Type: application/json")

    # Extrair código HTTP e corpo da resposta
    http_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')

    if [ "$http_code" = "200" ]; then
        # Verificar se a resposta indica sucesso
        if echo "$body" | grep -q '"success":true' && echo "$body" | grep -q '"connectionStatus":"OK"'; then
            echo "✅ AUTORIZADO"
            echo "   📊 Detalhes: $(echo "$body" | jq -r '.report.summary.dataQuality // "N/A"')"
            echo "   ⏱️  Tempo auth: $(echo "$body" | jq -r '.report.connectionTest.responseTime // "N/A"')"
            validated_cpfs=$((validated_cpfs + 1))
            
            # Atualizar como válido no banco
            docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
                "UPDATE temp_clients SET b3_validated = TRUE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
        else
            echo "⚠️ RESPOSTA INVÁLIDA (HTTP 200 mas sem sucesso)"
            failed_cpfs=$((failed_cpfs + 1))
            
            # Atualizar como inválido no banco
            docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
                "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
        fi
    elif [ "$http_code" = "422" ]; then
        echo "⚠️ SEM DADOS (422)"
        echo "   📝 Nota: CPF pode ter autorização mas sem dados no período"
        # Para 422, considerar como "possivelmente válido" mas marcar FALSE para ser conservador
        failed_cpfs=$((failed_cpfs + 1))
        docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
            "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
    elif [ "$http_code" = "500" ]; then
        echo "❌ ERRO SERVIDOR (500)"
        echo "   📝 Nota: Provavelmente sem autorização B3 ou erro OAuth2"
        failed_cpfs=$((failed_cpfs + 1))
        docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
            "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
    elif [ -z "$http_code" ]; then
        echo "⏰ TIMEOUT (45s)"
        timeout_cpfs=$((timeout_cpfs + 1))
        docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
            "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
    else
        echo "❌ ERRO HTTP ($http_code)"
        failed_cpfs=$((failed_cpfs + 1))
        docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
            "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
    fi

    # Pequena pausa para não sobrecarregar a API
    sleep 0.3
done

echo ""
echo "📊 RELATÓRIO FINAL DA VALIDAÇÃO B3:"
echo "   📋 Total de CPFs testados: $total_cpfs"
echo "   ✅ CPFs AUTORIZADOS (B3): $validated_cpfs"
echo "   ❌ CPFs SEM AUTORIZAÇÃO: $failed_cpfs"
echo "   ⏰ CPFs com TIMEOUT: $timeout_cpfs"
echo ""

# Calcular percentuais
if [ $total_cpfs -gt 0 ]; then
    success_rate=$(( (validated_cpfs * 100) / total_cpfs ))
    echo "📈 Taxa de SUCESSO: ${success_rate}%"
fi

echo ""
echo "💾 RESUMO NO BANCO DE DADOS:"
docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
    "SELECT 
        COUNT(*) as total_cpfs,
        SUM(CASE WHEN b3_validated = TRUE THEN 1 ELSE 0 END) as autorizados,
        SUM(CASE WHEN b3_validated = FALSE THEN 1 ELSE 0 END) as nao_autorizados,
        SUM(CASE WHEN b3_validated IS NULL THEN 1 ELSE 0 END) as nao_testados
     FROM temp_clients;"

echo ""
echo "🎯 PRÓXIMOS PASSOS:"
echo "   ✅ CPFs com b3_validated = TRUE → Seguros para usar em produção"
echo "   ❌ CPFs com b3_validated = FALSE → Não devem ser usados"
echo "   🔍 Use: SELECT * FROM temp_clients WHERE b3_validated = TRUE; para ver válidos"
echo ""
echo "✨ Validação B3 REAL concluída!"
