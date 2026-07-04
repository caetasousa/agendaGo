# agendaGo

> API de agendamento entre clientes e prestadores de serviço, construída em Go com arquitetura hexagonal.

---

## Sumário

- [Requisitos](#requisitos)
- [Executando o projeto](#executando-o-projeto)
- [Rotas disponíveis](#rotas-disponíveis)
- [Testes](#testes)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Regras de negócio](#regras-de-negócio)

---

## Requisitos

- [Docker](https://docs.docker.com/get-docker/) e Docker Compose

---

## Executando o projeto

```bash
docker compose up
```

O comando acima sobe backend e frontend juntos, na seguinte sequência:

1. **Postgres** sobe e aguarda ficar saudável (healthcheck) antes de liberar o próximo passo.
2. **Flyway** aplica as migrations (`backend/migrations`) contra o Postgres e encerra.
3. **API** gera a documentação Swagger (`swag init`) e sobe com hot reload via Air — alterações em arquivos `.go` reiniciam a API automaticamente. Disponível em `http://localhost:8080`.
4. **Web** instala as dependências e sobe o frontend (SvelteKit) com hot reload via Vite. Roda em paralelo aos passos 1-3, sem depender dessa cadeia. Disponível em `http://localhost:5173`.

> A pasta `docs/` (documentação Swagger) é gerada automaticamente pelo container a cada
> `docker compose up` e não é versionada — não é necessário rodar `swag init` manualmente.

### Encerrando

```bash
docker compose down        # mantém os dados do banco
docker compose down -v     # apaga os dados do banco junto
```

---

## Rotas disponíveis

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/health` | Status do servidor |
| `POST` | `/providers` | Cadastrar prestador |
| `GET` | [`/swagger/index.html`](http://localhost:8080/swagger/index.html) | Documentação interativa |

---

## Testes

A forma recomendada de rodar tudo (rápidos + integração) com saída legível:

```bash
cd backend
make test
```

`make test` lista cada caso de teste com `PASS`/`FAIL` e falha se algum quebrar. Exige Docker rodando (sobe um Postgres efêmero para os testes de integração).

Os testes se dividem em dois grupos, que também podem ser rodados diretamente:

**Testes rápidos** — regras de negócio, usecases e contrato HTTP. Rodam em memória, exigem apenas Go instalado (sem Docker):

```bash
go test ./...        # ou: make test-fast
```

**Testes de integração** — exercitam o repositório Postgres (SQL real) contra um banco efêmero criado sob demanda via [Testcontainers](https://testcontainers.com/). Exigem Docker rodando e a build tag `integration`:

```bash
go test -tags=integration ./...
```

O container do Postgres é criado, migrado e destruído automaticamente pelo próprio teste — não é necessário subir o `docker compose`.

> Use `-v` para ver cada caso de teste individualmente e `-count=1` para ignorar o cache.

```
test/
├── domain/       testes das regras de negócio
├── usecase/      testes dos fluxos de usecase
├── handler/      testes do contrato HTTP
└── repository/   testes de integração do repositório Postgres (Testcontainers)
```

---

## Estrutura do projeto

Monorepo com backend (Go) e frontend (SvelteKit) em pastas separadas:

```
agendaGo/
├── backend/              API em Go (arquitetura hexagonal)
│   ├── cmd/api/          entrypoint do servidor HTTP
│   ├── config/           configuração do servidor chi (porta, timeouts)
│   ├── internal/
│   │   ├── adapter/
│   │   │   ├── http/
│   │   │   │   ├── dto/      request e response HTTP
│   │   │   │   ├── handler/  handlers das rotas
│   │   │   │   └── middleware/
│   │   │   └── repository/   implementações de repositório (memória, postgres)
│   │   ├── domain/
│   │   │   ├── appointment/  agendamento e máquina de estados
│   │   │   ├── availability/ disponibilidade do prestador
│   │   │   ├── client/       cliente
│   │   │   ├── provider/     prestador de serviço
│   │   │   └── slot/         slots de horário (calculados sob demanda)
│   │   └── usecase/
│   │       ├── appointment/  solicitar, confirmar, recusar, cancelar, expirar
│   │       ├── availability/ definir disponibilidade e exceções
│   │       ├── client/       cadastro e busca de cliente
│   │       └── provider/     cadastro e configuração do prestador
│   ├── migrations/       arquivos SQL versionados pelo Flyway (V1__, V2__...)
│   └── test/
│       ├── domain/
│       ├── handler/
│       ├── repository/   integração do repositório Postgres (build tag `integration`)
│       └── usecase/
└── frontend/             cliente web em SvelteKit + TypeScript
    ├── src/
    └── static/
```

---

## Regras de negócio

Consulte [regra-de-negocio.md](regra-de-negocio.md) para entender o modelo completo:
disponibilidade do prestador, geração de slots, ciclo de vida do agendamento e decisões de arquitetura.
