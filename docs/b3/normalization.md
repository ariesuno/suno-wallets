# B3 Normalization (1.10)

Este documento descreve o endpoint de normalização dos payloads RAW (1.9) para tabelas normalizadas de transações (v2) e posições (v3).

## Endpoint

POST /api/v1/b3/normalize/run

Headers:
- X-Tenant-Id: <UUID>
- Content-Type: application/json

Body (JSON):
```
{
  "cpf": "12345678901",
  "dataType": "transactions",
  "assetType": "equity",
  "start": "2023-01-01",
  "end": "2023-01-31",
  "force": false,
  "dryRun": false
}
```

Response (200):
```
{
  "inserted": 10,
  "updated": 0,
  "skipped": 0,
  "errors": 0,
  "rawProcessed": 3
}
```

Notas:
- Linhagem: cada registro normalizado referencia `raw_id` e `sequence_in_raw`.
- Idempotência: upsert por `normalized_hash` e chaves de contexto.
- Sem regras de negócio: apenas flatten/tipagem; datas em YYYY-MM-DD; `ticker` em upper-case; `side` → BUY/SELL/OTHER.
- Segurança/observabilidade: CPF mascarado nos logs; métricas Prometheus (a serem expandidas).


