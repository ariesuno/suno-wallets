### B3 RAW Persistence — Ingestão Histórica (1.9)

Endpoint

POST /api/v1/b3/fetch/historical

Headers
- X-Tenant-Id: <TENANT_ID>
- Content-Type: application/json

Body (JSON)
{
  "cpf": "12345678901",
  "dataType": "transactions|positions",
  "assetType": "equity",
  "start": "YYYY-MM-DD",
  "end": "YYYY-MM-DD",
  "fetchAllPages": true,
  "force": false,
  "dryRun": false
}

Exemplo cURL

```bash
curl -X POST "http://localhost:8080/api/v1/b3/fetch/historical" \
  -H "X-Tenant-Id: 550e8400-e29b-41d4-a716-446655440001" \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678901",
    "dataType": "positions",
    "assetType": "equity",
    "start": "2024-01-01",
    "end": "2024-03-31",
    "fetchAllPages": true,
    "force": false,
    "dryRun": true
  }'
```

Resposta (sumário)
{
  "saved": 10,
  "skipped": 2,
  "errors": 0,
  "monthsProcessed": 3,
  "pagesProcessed": 10,
  "dryRun": true,
  "force": false,
  "tenantId": "550e8400-e29b-41d4-a716-446655440001"
}

Validações
- cpf: 11 dígitos numéricos
- dataType: "transactions" | "positions"
- assetType: default "equity"
- start/end: formato YYYY-MM-DD
- X-Tenant-Id obrigatório (UUID)

Métricas
- b3_raw_saved_total, b3_raw_skipped_total, b3_raw_errors_total
- b3_raw_pages_processed_total, b3_raw_months_completed_total

Observações
- dryRun=true não grava no banco; apenas simula
- force=true reprocessa mesmo períodos completos
- RAW não é excluído automaticamente (LGPD)

### B3 RAW Persistence — Ingestão Histórica (1.9)

Endpoint:

POST /api/v1/b3/fetch/historical

Headers:
- X-Tenant-Id: <TENANT_ID>
- Content-Type: application/json

Body (JSON):
{
  "cpf": "12345678901",
  "dataType": "transactions|positions",
  "assetType": "equity",
  "start": "YYYY-MM-DD",
  "end": "YYYY-MM-DD",
  "fetchAllPages": true,
  "force": false,
  "dryRun": false
}

Exemplo cURL:

```bash
curl -X POST "http://localhost:8080/api/v1/b3/fetch/historical" \
  -H "X-Tenant-Id: 550e8400-e29b-41d4-a716-446655440001" \
  -H "Content-Type: application/json" \
  -d {
