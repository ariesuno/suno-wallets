### B3 Transactions v2 — Preview (equity)

Endpoint:

GET /api/v1/b3/fetch/transactions/preview?cpf=XXXXXXXXXXX&start=YYYY-MM-DD&end=YYYY-MM-DD&assetType=equity&page=1&fetchAllPages=false

Exemplo cURL (substitua variáveis reais; tokens são gerenciados pelo cliente):

```bash
curl -s \
  -H "X-Tenant-Id: <TENANT_ID>" \
  "http://localhost:8080/api/v1/b3/fetch/transactions/preview?cpf=12345678901&start=2024-01-01&end=2024-01-31&assetType=equity&page=1&fetchAllPages=false"
```

Notas:
- Não há persistência neste endpoint.
- CPF é mascarado nos logs.
- Quando `fetchAllPages=true`, o serviço pagina até o fim.


