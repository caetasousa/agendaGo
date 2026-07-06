# Guia de testes

O agendaGo separa os testes em camadas de custo crescente: rápidos (memória, sem Docker) → integração (Postgres real via Testcontainers) → E2E (browser real via Playwright). Veja o "porquê" de cada ferramenta em [tecnologias.md](tecnologias.md#6-testes).

---

## Atalho rápido (raiz do repositório)

```bash
make test        # backend rápido + frontend unitário — sem Docker, sem browser
make test-all     # tudo: + integração (Postgres) + E2E (Playwright)
```

Alvos individuais no `Makefile` da raiz: `test-backend`, `test-backend-integration`, `test-frontend`, `test-e2e`.

---

## Backend (Go)

```bash
cd backend
make test          # rápidos + integração, saída legível (PASS/FAIL por caso)
make test-fast      # só os rápidos (sem Docker)
```

**Testes rápidos** — regras de negócio, usecases e contrato HTTP, em memória:

```bash
go test ./...
```

**Testes de integração** — repositório Postgres real, banco efêmero via [Testcontainers](https://testcontainers.com/), criado/migrado/destruído pelo próprio teste (não precisa do `docker compose` no ar):

```bash
go test -tags=integration ./...
```

> Use `-v` para ver cada caso individualmente e `-count=1` para ignorar o cache.

```
backend/test/
├── domain/       regras de negócio puras
├── usecase/      fluxos de usecase (repositórios em memória)
├── handler/      contrato HTTP via httptest
├── repository/   integração real contra Postgres (build tag `integration`)
├── security/     hasher de senha (Argon2id)
├── token/        geração/hash do token de sessão
└── config/       configuração (ex.: CookieSeguro)
```

---

## Frontend (SvelteKit)

**Testes unitários** — cliente HTTP, login unificado e store de sessão, em Node (sem browser):

```bash
cd frontend
npm run test:unit        # ou: npm test
```

**Testes E2E** — cadastro, login e sessão fim a fim, via [Playwright](https://playwright.dev/). Exigem o app no ar (`docker compose up` na raiz) e o browser instalado uma vez:

```bash
cd frontend
npx playwright install chromium   # só na primeira vez
npm run test:e2e
```

```
frontend/
├── src/lib/api/*.test.ts       cliente HTTP e login unificado
├── src/lib/stores/*.test.ts    store de sessão
└── e2e/                        specs Playwright (cadastro, login, sessão)
```
