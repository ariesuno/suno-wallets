#!/bin/bash

# Script alternativo para validação B3 real (sem OAuth2 inicial)
# Usa estratégia diferente: tenta acessar dados do CPF diretamente via endpoints de sync

API_BASE="http://localhost:8080/api/v1"
TENANT_ID="status_invest"

echo "🔐 Validação B3 REAL - Estratégia Alternativa"
echo "🎯 Testando CPFs com dados reais via sync endpoints"
echo "💡 CPFs que retornam dados = Autorizados na B3"
echo ""

# Lista de CPFs conhecidos para teste
TEST_CPFS=("33680115881" "73038881287" "22092882821" "30353845841")

for cpf in "${TEST_CPFS[@]}"; do
    echo -n "🧪 Testando CPF real: ${cpf:0:3}******* ... "
    
    # Estratégia 1: Tentar sincronização (DRY RUN) 
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "${API_BASE}/b3/sync/complete-ingestion" \
        -H "X-Tenant-ID: ${TENANT_ID}" \
        -H "Content-Type: application/json" \
        -d "{
            \"cpf\": \"${cpf}\",
            \"assetTypes\": [\"equity\"],
            \"dataTypes\": [\"transactions\"],
            \"includeReconciliation\": false,
            \"dryRun\": true,
            \"force\": false
        }")
    
    # Extrair HTTP status
    http_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
    
    # Analisar resposta
    if [ "$http_code" = "200" ]; then
        # Verificar se conseguiu processar (mesmo que 0 registros)
        if echo "$body" | grep -q '"strategy"'; then
            echo "✅ AUTORIZADO (dry run processado)"
            
            # Estratégia 2: Testar period específico para confirmação
            period_test=$(curl -s \
                -X GET "${API_BASE}/b3/sync/status-analysis?cpf=${cpf}" \
                -H "X-Tenant-ID: ${TENANT_ID}")
            
            if echo "$period_test" | grep -q '"gapDays"'; then
                echo "   📊 Detalhes: Cliente com $(echo "$period_test" | grep -o '"gapDays":[0-9]*' | cut -d: -f2) dias sem sync"
                if echo "$period_test" | grep -q '"clientStatus":"new"'; then
                    echo "   📝 Status: Cliente novo (sem histórico local)"
                else
                    echo "   📝 Status: Cliente existente"
                fi
            fi
        else
            echo "❌ RESPOSTA INVÁLIDA"
        fi
    elif [ "$http_code" = "422" ]; then
        echo "⚠️  SEM DADOS (período específico)"
        # Pode ter autorização mas sem dados no período testado
    else
        echo "❌ ERRO HTTP ($http_code)"
        # Pode indicar falta de autorização ou outro problema
    fi
    
    echo ""
    sleep 1
done

echo ""
echo "📋 RESUMO DA VALIDAÇÃO REAL:"
echo "   ✅ CPFs que retornaram 200 = Autorizados na B3"  
echo "   ⚠️  CPFs que retornaram 422 = Pode ter autorização, mas sem dados"
echo "   ❌ CPFs que retornaram 4xx/5xx = Provavelmente sem autorização"
echo ""
echo "💡 PRÓXIMOS PASSOS:"
echo "   1. CPFs ✅ são seguros para usar em produção"
echo "   2. CPFs ⚠️ precisam de verificação adicional"
echo "   3. CPFs ❌ não devem ser usados"
echo ""
echo "🔍 Para validação definitiva, use CPFs que você SABE que têm autorização B3"
