#!/bin/bash

# Script para validar CPFs usando o método de sync dry-run que funciona
# Mais confiável que o endpoint /test/connection que está com problemas OAuth2

API_BASE="http://localhost:8080/api/v1"
TENANT_ID="status_invest"

echo "🔐 VALIDAÇÃO B3 - Método Sync Dry-Run (Confiável)"
echo "🎯 Endpoint: ${API_BASE}/b3/sync/complete-ingestion"
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

echo "🚀 Iniciando validação..."
echo ""

for cpf in $CPFS; do
    cpf=$(echo "$cpf" | tr -d ' \n\r') # Limpar espaços e quebras de linha
    if [ -z "$cpf" ]; then continue; fi
    
    total_cpfs=$((total_cpfs + 1))
    echo -n "🧪 Testando CPF #${total_cpfs}: ${cpf:0:3}******* ... "

    # Fazer requisição ao endpoint de sync dry-run
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" --max-time 30 \
        -X POST "${API_BASE}/b3/sync/complete-ingestion" \
        -H "X-Tenant-ID: ${TENANT_ID}" \
        -H "Content-Type: application/json" \
        -d "{
            \"cpf\": \"$cpf\",
            \"assetTypes\": [\"equity\"],
            \"dataTypes\": [\"transactions\"],
            \"includeReconciliation\": false,
            \"dryRun\": true,
            \"force\": false
        }")

    # Extrair código HTTP e corpo da resposta
    http_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')

    if [ "$http_code" = "200" ]; then
        # Verificar se a resposta indica sucesso
        if echo "$body" | grep -q '"success":true'; then
            echo "✅ AUTORIZADO"
            client_status=$(echo "$body" | jq -r '.clientStatus // "N/A"')
            echo "   📊 Status: $client_status"
            validated_cpfs=$((validated_cpfs + 1))
            
            # Atualizar como válido no banco
            docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
                "UPDATE temp_clients SET b3_validated = TRUE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
        else
            echo "⚠️ SUCESSO PARCIAL (HTTP 200 mas com erro)"
            failed_cpfs=$((failed_cpfs + 1))
            docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
                "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
        fi
    elif [ "$http_code" = "422" ]; then
        echo "⚠️ SEM DADOS (422)"
        echo "   📝 Nota: CPF pode ter autorização mas sem dados"
        # Para 422, considerar como "possivelmente válido" mas marcar FALSE para ser conservador
        failed_cpfs=$((failed_cpfs + 1))
        docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
            "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
    elif [ "$http_code" = "500" ]; then
        echo "❌ ERRO SERVIDOR (500)"
        if echo "$body" | grep -q "oauth"; then
            echo "   📝 Erro OAuth2 - sem autorização B3"
        else
            echo "   📝 Erro interno do servidor"
        fi
        failed_cpfs=$((failed_cpfs + 1))
        docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
            "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
    elif [ -z "$http_code" ]; then
        echo "⏰ TIMEOUT (30s)"
        failed_cpfs=$((failed_cpfs + 1))
        docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
            "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
    else
        echo "❌ ERRO HTTP ($http_code)"
        failed_cpfs=$((failed_cpfs + 1))
        docker-compose exec -T postgres psql -U suno_user -d suno_wallets -c \
            "UPDATE temp_clients SET b3_validated = FALSE, updated_at = NOW() WHERE cpf = '$cpf';" > /dev/null
    fi

    # Pequena pausa para não sobrecarregar a API
    sleep 0.2
done

echo ""
echo "📊 RELATÓRIO FINAL DA VALIDAÇÃO B3:"
echo "   📋 Total de CPFs testados: $total_cpfs"
echo "   ✅ CPFs AUTORIZADOS (B3): $validated_cpfs"
echo "   ❌ CPFs SEM AUTORIZAÇÃO: $failed_cpfs"
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
echo "🎯 MÉTODO USADO: Sync Dry-Run (mais confiável que /test/connection)"
echo "✨ Validação B3 concluída!"
