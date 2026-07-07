# agendaGo

> Agendamento entre clientes e prestadores de serviço — API em Go (arquitetura hexagonal) + frontend SvelteKit. Projeto de estudo.

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)

---

## Stack

| Camada | Tecnologia |
|---|---|
| Backend | Go 1.26 · [chi](https://github.com/go-chi/chi) · [pgx](https://github.com/jackc/pgx) · [Argon2id](https://github.com/alexedwards/argon2id) |
| Banco | PostgreSQL 16 · [Flyway](https://flywaydb.org/) (migrations) |
| Frontend | [Svelte 5](https://svelte.dev) · SvelteKit · TypeScript · Tailwind CSS 4 |
| Testes | Go testing · [Testcontainers](https://testcontainers.com/) · [Vitest](https://vitest.dev/) · [Playwright](https://playwright.dev/) |

Guia completo, com o "porquê" de cada escolha e fontes para estudo: **[docs/tecnologias.md](docs/tecnologias.md)**.

---

## Executando o projeto

Requisito: [Docker](https://docs.docker.com/get-docker/) e Docker Compose.

```bash
docker compose up
```

Sobe Postgres → Flyway (migrations) → API (`:8080`, hot reload via Air) → frontend (`:5173`, hot reload via Vite). A documentação Swagger é gerada automaticamente.

```bash
docker compose down        # mantém os dados do banco
docker compose down -v     # apaga os dados do banco junto
```

- App: [http://localhost:5173](http://localhost:5173)
- API: [http://localhost:8080](http://localhost:8080)
- Swagger: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

---

## Rotas disponíveis

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | [`/health`](http://localhost:8080/swagger/index.html#/infra/get_health) | Status do servidor |
| `POST` | [`/providers`](http://localhost:8080/swagger/index.html#/providers/post_providers) | Cadastrar prestador |
| `POST` | [`/clients`](http://localhost:8080/swagger/index.html#/clients/post_clients) | Cadastrar cliente (com conta) |
| `POST` | [`/auth/provider/login`](http://localhost:8080/swagger/index.html#/auth/post_auth_provider_login) | Login do prestador |
| `POST` | [`/auth/client/login`](http://localhost:8080/swagger/index.html#/auth/post_auth_client_login) | Login do cliente |
| `POST` | [`/auth/logout`](http://localhost:8080/swagger/index.html#/auth/post_auth_logout) | Encerrar sessão |
| `GET` | [`/auth/me`](http://localhost:8080/swagger/index.html#/auth/get_auth_me) | Usuário autenticado atual |
| `PUT` | [`/providers/me/preferencias`](http://localhost:8080/swagger/index.html#/providers/put_providers_me_preferencias) | Atualizar preferências do prestador |
| `GET` | [`/providers/me/disponibilidade`](http://localhost:8080/swagger/index.html#/availability/get_providers_me_disponibilidade) | Consultar grade semanal do prestador |
| `PUT` | [`/providers/me/disponibilidade`](http://localhost:8080/swagger/index.html#/availability/put_providers_me_disponibilidade) | Definir grade semanal do prestador |
| `GET` | [`/providers/me/excecoes`](http://localhost:8080/swagger/index.html#/availability/get_providers_me_excecoes) | Listar exceções de data do prestador |
| `POST` | [`/providers/me/excecoes`](http://localhost:8080/swagger/index.html#/availability/post_providers_me_excecoes) | Criar exceção de data (bloqueio ou extra) |
| `DELETE` | [`/providers/me/excecoes/{id}`](http://localhost:8080/swagger/index.html#/availability/delete_providers_me_excecoes__id_) | Remover exceção de data |
| `GET` | [`/swagger/index.html`](http://localhost:8080/swagger/index.html) | Documentação interativa |

---

## Testes

```bash
make test        # rápidos (backend + frontend), sem Docker/browser
make test-all     # tudo: + integração (Postgres) + E2E (Playwright)
```

Guia completo (build tags, Testcontainers, Playwright): **[docs/testes.md](docs/testes.md)**.

---

## Documentação

- **[docs/tecnologias.md](docs/tecnologias.md)** — guia de estudo: o que é cada tecnologia, por que está aqui, fontes oficiais
- **[docs/testes.md](docs/testes.md)** — como rodar cada camada de teste
- **[docs/regra-de-negocio.md](docs/regra-de-negocio.md)** — modelo de negócio: disponibilidade, slots, ciclo de vida do agendamento

---

## Estrutura do projeto

Monorepo com arquitetura hexagonal no backend (`domain` → `usecase` → `adapter`) e SvelteKit no frontend:

```
agendaGo/
├── backend/    API em Go — cmd/, config/, internal/{domain,usecase,adapter}/, migrations/, test/
├── frontend/   SvelteKit — src/{lib,routes}/, e2e/
└── docs/       tecnologias.md · testes.md · regra-de-negocio.md
```
