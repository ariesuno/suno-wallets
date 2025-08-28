# Guia de Operações - Backoffice Admin

Este guia explica como utilizar o **Backoffice Admin** e o que observar no **Grafana/Prometheus** para monitoramento operacional.

## 📊 Métricas Prometheus

### Métricas de Ações Administrativas

| Métrica | Tipo | Labels | Descrição |
|---------|------|--------|-----------|
| `admin_actions_total` | Counter | `action`, `status` | Total de ações administrativas executadas |
| `admin_actions_duration_seconds` | Histogram | `action` | Duração das execuções de ações administrativas |
| `admin_exports_total` | Counter | `format` | Total de exports executados por formato |
| `admin_profile_requests_total` | Counter | `result` | Total de requisições de profile do cliente |

### Status de Ações

- `REQUESTED` - Ação solicitada, aguardando confirmação
- `CONFIRMED` - Ação confirmada, executando
- `RUNNING` - Em execução
- `SUCCESS` - Executada com sucesso
- `ERROR` - Erro na execução
- `SKIPPED` - Pulada por alguma condição

## 🔍 Monitoramento por Ação

### B3 Full Fetch (`B3_FULL_FETCH`)
**Observar no Grafana:**
- **Duração**: `admin_actions_duration_seconds{action="B3_FULL_FETCH"}` 
  - ✅ Normal: 30s-5min (depende do histórico)
  - ⚠️ Alerta: > 10min
- **Taxa de sucesso**: `rate(admin_actions_total{action="B3_FULL_FETCH",status="SUCCESS"}[5m])`
- **Erros**: `increase(admin_actions_total{action="B3_FULL_FETCH",status="ERROR"}[1h])`

### B3 Incremental (`B3_INCREMENTAL_FETCH`)
**Observar no Grafana:**
- **Duração**: `admin_actions_duration_seconds{action="B3_INCREMENTAL_FETCH"}` 
  - ✅ Normal: 5s-2min
  - ⚠️ Alerta: > 5min
- **Frequência**: Deve ser usado para atualizações diárias

### Reconciliation Scan (`RECON_SCAN`)
**Observar no Grafana:**
- **Duração**: `admin_actions_duration_seconds{action="RECON_SCAN"}`
- **Inconsistências encontradas**: Verificar logs estruturados
- **Taxa de detecção**: Monitorar padrões anômalos

### Auto Fix (`AUTO_FIX`)
**Observar no Grafana:**
- **Sucesso vs Error**: `admin_actions_total{action="AUTO_FIX"}`
- **Impacto**: Verificar se `rate(admin_actions_total{action="AUTO_FIX",status="SUCCESS"}[1h])` está estável

### Policy Update (`POLICY_UPDATE`)
**Observar no Grafana:**
- **Frequência**: Baixa frequência é normal
- **Erros**: `increase(admin_actions_total{action="POLICY_UPDATE",status="ERROR"}[24h])`

### Ações Perigosas (`CLIENT_RESET`, `CLIENT_ZERO_AND_REFETCH`)
**Observar no Grafana:**
- **Alertas críticos**: Qualquer execução dessas ações
- **Duração**: Podem ser longas (5-30min)
- **Frequência**: Deve ser muito baixa

## 📈 Dashboards Recomendados

### Dashboard Principal - Admin Actions
```promql
# Total de ações por status (últimas 24h)
increase(admin_actions_total[24h])

# Duração média por tipo de ação
rate(admin_actions_duration_seconds_sum[5m]) / rate(admin_actions_duration_seconds_count[5m])

# Top 5 ações mais executadas
topk(5, increase(admin_actions_total[1h]))

# Taxa de erro por ação
rate(admin_actions_total{status="ERROR"}[5m]) / rate(admin_actions_total[5m])
```

### Dashboard Exports
```promql
# Exports por formato
admin_exports_total

# Requisições de profile
increase(admin_profile_requests_total[1h])
```

## 🚨 Alertas Recomendados

### Críticos
```yaml
# Alta taxa de erro em ações críticas
- alert: AdminActionHighErrorRate
  expr: rate(admin_actions_total{status="ERROR"}[5m]) / rate(admin_actions_total[5m]) > 0.1
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "Alta taxa de erro em ações administrativas: {{ $value | humanizePercentage }}"

# Execução de ação perigosa
- alert: DangerousActionExecuted  
  expr: increase(admin_actions_total{action=~"CLIENT_RESET|CLIENT_ZERO_AND_REFETCH"}[5m]) > 0
  for: 0s
  labels:
    severity: critical
  annotations:
    summary: "Ação perigosa executada: {{ $labels.action }}"
```

### Warnings
```yaml
# Duração anômala de B3 Full Fetch
- alert: B3FullFetchSlow
  expr: histogram_quantile(0.95, admin_actions_duration_seconds{action="B3_FULL_FETCH"}) > 600
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "B3 Full Fetch está lento: {{ $value }}s no P95"

# Muitos exports simultâneos
- alert: HighExportVolume
  expr: rate(admin_exports_total[5m]) > 10
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "Volume alto de exports: {{ $value }} exports/s"
```

## 📋 Checklist Operacional

### Durante Execução de B3 Full Fetch
- [ ] Verificar duração esperada baseada no histórico
- [ ] Monitorar uso de CPU/memória do banco
- [ ] Verificar logs de erro para falhas de conexão B3
- [ ] Confirmar que não há outras ingestões simultâneas

### Durante Auto Fix
- [ ] Verificar tipos de inconsistências sendo corrigidas
- [ ] Monitorar impacto no ledger normalizado
- [ ] Confirmar que correções são idempotentes
- [ ] Validar resultado via endpoint de profile

### Durante Client Reset (CRÍTICO)
- [ ] **Confirmar dupla aprovação obtida**
- [ ] Fazer backup manual antes da execução
- [ ] Monitorar logs em tempo real
- [ ] Validar estado após execução
- [ ] Documentar razão da execução

## 🔧 Troubleshooting

### Ação Travada em "RUNNING"
1. Verificar se há locks de BD ativos
2. Checar logs de erro da aplicação
3. Verificar conectividade com B3 (se aplicável)
4. Considerar restart da ação se > timeout esperado

### Taxa de Erro Alta
1. Verificar conectividade com serviços externos
2. Analisar logs estruturados para padrões
3. Verificar capacidade do banco de dados
4. Confirmar que não há problemas de permissão

### Performance Degradada
1. Verificar índices do banco estão sendo utilizados
2. Monitorar uso de locks por (tenantId, cpf)
3. Analisar se há concorrência excessiva
4. Verificar se export streaming está funcionando

## 📞 Escalação

### Nível 1 - Suporte
- Pode executar: consultas, dry-run, recon-scan
- Não pode: auto-fix, policy-set, ações destrutivas

### Nível 2 - Admin  
- Pode executar: todas as ações exceto destrutivas
- Precisa aprovação para: client-reset, zero-and-refetch

### Nível 3 - Operações Críticas
- Requer: dupla aprovação + documentação
- Backup obrigatório antes de execução
- Monitoramento em tempo real obrigatório
