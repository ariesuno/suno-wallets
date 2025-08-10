B3 Reports (Somente leitura)

Fornece visibilidade rápida sobre cobertura temporal (RAW), sumário agregado e consolidação por ticker. Sem regras de negócio (sem PM/P&L/impostos).

Endpoints

- GET `/api/v1/b3/client/raw-date-range?cpf=00000000000`
- GET `/api/v1/b3/client/summary?cpf=00000000000&from=YYYY-MM-DD&to=YYYY-MM-DD`
- GET `/api/v1/b3/client/tickers?cpf=00000000000&from=YYYY-MM-DD&to=YYYY-MM-DD`

Headers obrigatórios: `X-Tenant-Id` e autenticação (se aplicada pelo gateway).

Exemplos cURL

```bash
# RAW date range
curl "http://localhost:8080/api/v1/b3/client/raw-date-range?cpf=00000000000" \
  -H "X-Tenant-Id: $TENANT"

# Summary
curl "http://localhost:8080/api/v1/b3/client/summary?cpf=00000000000&from=2024-01-01&to=2024-12-31" \
  -H "X-Tenant-Id: $TENANT"

# Tickers
curl "http://localhost:8080/api/v1/b3/client/tickers?cpf=00000000000&from=2024-01-01&to=2024-12-31" \
  -H "X-Tenant-Id: $TENANT"
```

Observabilidade

- Logs estruturados: `tenantId`, `cpfMasked`, `from`, `to`.
- Métricas Prometheus: `b3_reports_requests_total`, `b3_reports_duration_seconds`, `b3_reports_errors_total`.

Notas

- Consultas sempre filtram por `tenant_id` e `cpf`.
- `grossValueBRLSum` retorna soma apenas se existir no payload; sem PM/P&L.
- Posições usadas como base para consolidação por ticker.

