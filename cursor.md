# 📐 Diretrizes de Desenvolvimento — `suno-wallets` (versão revisada)

Estas diretrizes são **obrigatórias** em todos os prompts, arquivos e PRs do projeto. Em caso de conflito entre o prompt e este guia, **este guia prevalece**.

## 🌍 Idioma, Nomenclatura e Estilo
- **Código SEMPRE em inglês** (arquivos, packages, tipos, funções, variáveis, endpoints).
- **Comentários SEMPRE em português-BR**, claros e objetivos, explicando _o porquê_ além do _o quê_.
- Convenções (Go):
  - **packages**: minúsculo, sem underscore (ex.: `b3`, `transactions`, `observability`).
  - **arquivos**: `snake_case.go` (ex.: `b3_official_client.go`).
  - **tipos/structs**: `PascalCase` (ex.: `B3OfficialClient`).
  - **métodos/funções/vars**: `camelCase`.
  - **testes**: `*_test.go`.
- Tamanho de arquivo: **máx. ~300 linhas**. Quebre em arquivos coesos quando ultrapassar sempre que possível.
- Nomear com semântica do domínio. Evitar genéricos como `util.go`, `service.go`.

## 🧱 Estrutura (DDD)
```
/src
  /api                 # HTTP (controllers, routes, middlewares, validators específicos de request)
  /application         # Casos de uso e orquestração de serviços (sem dependências de infra concreta)
  /domain              # Entidades, value objects, enums, interfaces (contratos) — sem infra
  /infrastructure      # Repositórios, B3 client, DB/Redis, background jobs, observabilidade, config
  /shared              # Helpers, errors, constants, context/tenant, logging, validation comum
  /tests               # Unit e Integration (espelhar estrutura por contexto)
/docs                  # Swagger, Postman, Bruno, guias por módulo, cURL de exemplo
/prompts               # Prompts usados (com .gitignore)
/prompt_results        # Respostas do Cursor (com .gitignore)
```

Restrições de fronteiras (muito importantes):
- `Domain` **nunca** importa `Infrastructure`.
- `Application` depende de **interfaces do `Domain`**, **nunca** de implementações concretas.
- `Api` chama **Application** e valida input/output.
- `Infrastructure` provê implementações **através** de interfaces do `Domain`.

## 🏷️ Multi-tenant (obrigatório)
- Header padrão: **`X-Tenant-Id`**. Criar middleware em `/Api` que valida e injeta no **context**.
- Campo **`tenantId`** obrigatório em entidades/tabulação e **filtro** de todas as queries.
- Logs e métricas **sempre** com `tenantId`.
- Testes devem cobrir _escopo de tenant_ (um tenant não enxerga dados de outro).

## 🔐 Segurança e Governança de Dados
- Validar **input** em todo endpoint (tipos, formatos, ranges).
- **Nunca** logar segredos/tokens. **Mascarar CPF** e PII.
- **RAW data da B3 nunca é excluído automaticamente**. Exclusão **apenas manual** (LGPD/inatividade), com log de auditoria.
- Ao consultar B3, **usar o RAW local se já existir** (evitar rate limit e custo).
- Permissões administrativas: endpoints internos (debug, reprocessamento, purge) exigem perfil `admin` do tenant.

## 📊 Observabilidade (sempre)
- **Prometheus**: expor métricas de latência (histogram), contadores de sucesso/erro por endpoint/status, retries, volume por tipo de ativo.
- **Logs estruturados (JSON)**: `timestamp`, `level`, `tenantId`, `endpoint`, `cpfMasked`, `httpStatus`, `durationMs`, `retries`, `traceId`.
- **Loki** (ou equivalente) centralizando logs; **Grafana** com dashboards:
  - Visão geral (APIs, jobs, erros, latência p95/p99).
  - Integração B3 (sucesso/erro por endpoint, 429/5xx, tempo por página).
- Endpoint `/metrics` habilitado para scrape.
- Sempre adicionar observabilidade **no mesmo PR** de qualquer funcionalidade executável.

## 🧪 Testes
- Pastas:
  - `/src/tests/unit/...`
  - `/src/tests/integration/...`
- Framework: **testify**.
- Convenções:
  - Prefixar testes de B3 com **`b3_...`** (facilita consulta).
  - Testes unitários para validações, formatação, paginação, retry, etc.
  - Testes de integração para conexões reais (quando possível via envs). Na ausência de dados reais, **parar e pedir instruções** — **nada de dados fake sem autorização explícita**.
- Cobertura mínima: **definir meta por PR** (sugestão: 70%+).

## 📄 Documentação Viva (obrigatória por prompt)
- **Swagger**: atualizar sempre (tags por domínio: `B3 Data`, `Users`, `Health`, `Debug`, etc.).
- **Postman/Bruno**: collections em `/docs`, importáveis, com exemplos reais (sanitizados).
- **Docs**: criar arquivos por módulo em `/docs/<modulo>/...` com exemplos **cURL** e fluxos.
- Sempre salvar o **prompt** em `/prompts` e o **resultado** em `/prompt_results` (ambos ignorados pelo Git).

## 🚦 Regras para Prompts (checklist que o Cursor deve seguir SEMPRE)
1. Respeitar **estrutura DDD** e fronteiras.
2. Nomes **em inglês**, comentários **pt-BR**.
3. **Não criar** arquivos fora das pastas definidas.
4. Gerar e organizar **testes** nas pastas corretas (sem duplicar).
5. Atualizar **Swagger** e **collections** (Postman/Bruno) quando houver endpoints.
6. Adicionar **métricas** e **logs** (com CPF mascarado e `tenantId`).
7. **Não usar dados fake** sem pedir autorização. Se faltar dado, **parar** e perguntar.
8. Se criar arquivo temporário, **deletar** ao final do prompt.
9. Se um arquivo grande emergir (>300 linhas), **quebrar** em módulos.
10. Registrar **README/Docs** quando necessário (instalação local, variáveis de ambiente, exemplos).

## 🧩 Integração B3 — padrões obrigatórios (quando aplicável)
- Uso **obrigatório** do `B3OfficialClient` (mTLS `.p12`, OAuth2 Client Credentials/PKCE).
- Headers **sempre**: `Authorization: Bearer <token>` e `X-Investor-CPF: <cpf>`.
- **Validações**:
  - CPF com **11 dígitos** (apenas números).
  - Datas `YYYY-MM-DD`.
  - `page >= 1` (padrão inicia em 1).
- **Resiliência**: retry com **backoff exponencial + jitter** para `429` e `5xx`.
- **Preview endpoints** **não** persistem (RAW persiste apenas no bloco específico).
- **RAW**: antes de consultar a B3, checar se o período já existe; se existir, **reusar**.

## ⚙️ CI/CD (recomendado para cada PR)
- Executar: `go vet`, `golangci-lint`, `go test ./...` (com relatório de cobertura).
- Validar Swagger (`swagger validate`) e sincronizar collections.
- Bloquear merge se:
  - Arquivos/sumários fora de pasta.
  - Testes quebrados ou sem cobertura mínima.
  - Falta de atualização do Swagger/collections quando há endpoints.

## 🔒 Variáveis de Ambiente (exemplos)
- `B3_URL_DATA`, `B3_CLIENT_ID`, `B3_CLIENT_SECRET`
- `B3_CERT_P12_PATH`, `B3_CERT_PASSPHRASE`
- `DB_URL`, `REDIS_URL`
- `LOG_LEVEL`

## 🧹 Antipadrões PROIBIDOS
- Duplicar testes/código sem remoção do obsoleto.
- Colocar arquivos na raiz do repo ou fora da árvore definida.
- Usar dados fake sem avisar.
- Logar tokens/CPFs/segredos em texto puro.
- Acoplamento de `Application` com `Infrastructure` concreta.

## ✅ Entrega Mínima por Prompt Estruturante
- Código + testes (unit/integration) nas pastas corretas.
- Observabilidade (métricas/logs) adicionada.
- Swagger e collections atualizadas (quando houver endpoints).
- Prompt e resultado salvos (`/prompts`, `/prompt_results`).
- Documentação curta em `/docs/<modulo>/...` quando fizer sentido.

> **Dica:** Antes de escrever novo arquivo, verifique se um já existe com responsabilidade semelhante. **Atualize** em vez de duplicar.
