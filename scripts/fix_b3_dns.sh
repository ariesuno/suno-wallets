#!/bin/bash

# Script para corrigir problemas DNS B3 de forma definitiva
# Adiciona entradas DNS diretas no /etc/hosts do container

echo "🔧 Corrigindo DNS B3..."

# IPs atuais (obtidos via nslookup do host)
MS_IPS="20.190.173.129 20.190.173.145 20.190.173.128 20.190.173.144 20.190.173.146 20.190.173.65 20.190.173.130 20.190.173.66"
B3_IP="191.233.25.233"

for ip in $MS_IPS; do
    echo "📝 Adicionando $ip login.microsoftonline.com"
    docker-compose exec app sh -c "echo '$ip login.microsoftonline.com' >> /etc/hosts" 2>/dev/null
done

echo "📝 Adicionando $B3_IP investidor.b3.com.br"
docker-compose exec app sh -c "echo '$B3_IP investidor.b3.com.br' >> /etc/hosts" 2>/dev/null

echo "✅ DNS B3 configurado diretamente no /etc/hosts"
echo "🧪 Testando conectividade..."

# Testar resolução
docker-compose exec app sh -c "nslookup login.microsoftonline.com" || true
echo "📋 Mostrando entradas /etc/hosts relevantes:"
docker-compose exec app sh -c "grep -E '(microsoftonline|b3.com)' /etc/hosts" || true

