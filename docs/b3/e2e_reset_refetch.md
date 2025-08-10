Rota admin para reset seguro dos dados de um CPF e reingestão histórica completa com normalização.

- Rota: `POST /api/v1/b3/admin/reset-and-refetch`
- Headers: `X-Tenant-ID` (obrigatório), `X-Admin-Secret` (se `ADMIN_SECRET` definido)

Exemplo:

```bash
curl -X POST "$BASE_URL/api/v1/b3/admin/reset-and-refetch" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: $TENANT" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Admin-Secret: $ADMIN_SECRET" \
  -d '{
        "cpf":"00000000000",
        "assetTypes":["equity"],
        "dataTypes":["transactions","positions"],
        "force": true,
        "dryRun": true,
        "mode": "archive",
        "confirm": "RESET_AND_REFETCH"
      }'
```

Resposta inclui `raw`, `normalized`, `startedAt`, `finishedAt`, `durationMs`, `mode`, `force`, `dryRun`.


