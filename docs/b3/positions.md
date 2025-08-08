### B3 Positions v3 — Preview (equity)

Endpoint:

GET /api/v1/b3/fetch/positions/preview?cpf=XXXXXXXXXXX&start=YYYY-MM-DD&end=YYYY-MM-DD&assetType=equity&page=1&fetchAllPages=false

Exemplo cURL:

```bash
curl -s \
  -H "X-Tenant-Id: <TENANT_ID>" \
  "http://localhost:8080/api/v1/b3/fetch/positions/preview?cpf=12345678901&start=2024-01-01&end=2024-01-31&assetType=equity&page=1&fetchAllPages=false"
```

Notas:
- Sem persistência; apenas preview.
- CPF mascarado nos logs.
- `fetchAllPages=true` itera páginas até o fim.


