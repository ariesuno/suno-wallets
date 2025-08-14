#!/bin/sh

# Script para configurar rotas específicas para B3 dentro do container
# Resolve problemas de conectividade com domains B3

echo "🔧 Configurando rotas B3..."

# Verificar se é root
if [ "$(id -u)" != "0" ]; then
    echo "❌ Script precisa ser executado como root"
    exit 1
fi

# IPs conhecidos para B3
B3_IP="191.233.25.233"
MS_IP="20.190.173.1"

# Configurar rota específica para B3
echo "📡 Configurando rota para investidor.b3.com.br..."
ip route add $B3_IP via $(ip route show default | awk '/default/ {print $3}') 2>/dev/null || true

# Configurar rota específica para Microsoft
echo "📡 Configurando rota para login.microsoftonline.com..."  
ip route add $MS_IP via $(ip route show default | awk '/default/ {print $3}') 2>/dev/null || true

# Atualizar hosts file
echo "📝 Atualizando /etc/hosts..."
echo "$B3_IP investidor.b3.com.br" >> /etc/hosts
echo "$MS_IP login.microsoftonline.com" >> /etc/hosts

# Verificar conectividade
echo "🧪 Testando conectividade..."
if nc -z -w5 $B3_IP 2443; then
    echo "✅ B3 port 2443 acessível"
else
    echo "❌ B3 port 2443 inacessível"
fi

if nc -z -w5 $MS_IP 443; then
    echo "✅ Microsoft port 443 acessível"
else
    echo "❌ Microsoft port 443 inacessível"
fi

echo "🎯 Configuração de rotas B3 concluída"
