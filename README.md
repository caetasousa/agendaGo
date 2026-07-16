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
- Mailpit (emails capturados em dev, nada é enviado de verdade): [http://localhost:8025](http://localhost:8025)

### Administrador

O **administrador** (moderação) é semeado no boot a partir de `ADMIN_EMAIL` e `ADMIN_SENHA`
(definidos no `docker-compose.yml`). Em desenvolvimento, as credenciais são:

| Campo | Valor |
|---|---|
| E-mail | `admin@agendago.dev` |
| Senha | `admin12345` |

Ele entra pela mesma tela de login ([http://localhost:5173/login](http://localhost:5173/login))
e cai no painel de moderação (`/admin`), onde bane/reativa prestadores e clientes.
**Troque essas credenciais em produção** editando as variáveis de ambiente.

---

## Rotas disponíveis

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | [`/health`](http://localhost:8080/swagger/index.html#/infra/get_health) | Status do servidor |
| `POST` | [`/providers`](http://localhost:8080/swagger/index.html#/providers/post_providers) | Cadastrar prestador |
| `POST` | [`/clients`](http://localhost:8080/swagger/index.html#/clients/post_clients) | Solicitar cadastro de cliente (envia email de confirmação) |
| `POST` | [`/clients/confirmar-cadastro`](http://localhost:8080/swagger/index.html#/clients/post_clients_confirmar_cadastro) | Confirmar cadastro pelo token do email |
| `GET` | [`/clients/pre-cadastro/{token}`](http://localhost:8080/swagger/index.html#/clients/get_clients_pre_cadastro__token_) | Consultar dados de pré-cadastro (nome/email/telefone) para pré-preencher o cadastro |
| `POST` | [`/clients/pre-cadastro/{token}`](http://localhost:8080/swagger/index.html#/clients/post_clients_pre_cadastro__token_) | Concluir o cadastro a partir do pré-cadastro, sem uma segunda confirmação por email |
| `POST` | [`/auth/provider/login`](http://localhost:8080/swagger/index.html#/auth/post_auth_provider_login) | Login do prestador |
| `POST` | [`/auth/client/login`](http://localhost:8080/swagger/index.html#/auth/post_auth_client_login) | Login do cliente |
| `POST` | [`/auth/admin/login`](http://localhost:8080/swagger/index.html#/auth/post_auth_admin_login) | Login do administrador |
| `POST` | [`/auth/logout`](http://localhost:8080/swagger/index.html#/auth/post_auth_logout) | Encerrar sessão |
| `GET` | [`/auth/me`](http://localhost:8080/swagger/index.html#/auth/get_auth_me) | Usuário autenticado atual |
| `POST` | [`/auth/recuperar-senha`](http://localhost:8080/swagger/index.html#/auth/post_auth_recuperar_senha) | Solicitar recuperação de senha por email |
| `POST` | [`/auth/redefinir-senha`](http://localhost:8080/swagger/index.html#/auth/post_auth_redefinir_senha) | Redefinir a senha com um token de recuperação |
| `PUT` | [`/providers/me/preferencias`](http://localhost:8080/swagger/index.html#/providers/put_providers_me_preferencias) | Atualizar preferências do prestador |
| `GET` | [`/providers/me/agenda`](http://localhost:8080/swagger/index.html#/availability/get_providers_me_agenda) | Consultar agenda resolvida do prestador (por período) |
| `PUT` | [`/providers/me/dias/{data}`](http://localhost:8080/swagger/index.html#/availability/put_providers_me_dias__data_) | Definir um dia (bloqueio ou horários personalizados) |
| `DELETE` | [`/providers/me/dias/{data}`](http://localhost:8080/swagger/index.html#/availability/delete_providers_me_dias__data_) | Remover a definição de um dia (volta ao padrão) |
| `GET` | [`/providers`](http://localhost:8080/swagger/index.html#/providers/get_providers) | Listar prestadores (vitrine) |
| `GET` | [`/providers/{id}`](http://localhost:8080/swagger/index.html#/providers/get_providers__id_) | Buscar prestador (página pública de agendamento) |
| `GET` | [`/providers/{id}/slots`](http://localhost:8080/swagger/index.html#/appointments/get_providers__id__slots) | Consultar horários livres de um prestador (por período) |
| `POST` | [`/agendamentos`](http://localhost:8080/swagger/index.html#/appointments/post_agendamentos) | Solicitar um agendamento (cliente) |
| `POST` | [`/agendamentos/convidado`](http://localhost:8080/swagger/index.html#/appointments/post_agendamentos_convidado) | Solicitar um agendamento sem cadastro (nome/e-mail/telefone) |
| `GET` | [`/agendamentos/cancelar/{token}`](http://localhost:8080/swagger/index.html#/appointments/get_agendamentos_cancelar__token_) | Detalhar um agendamento pelo token de cancelamento (convidado) |
| `POST` | [`/agendamentos/cancelar/{token}`](http://localhost:8080/swagger/index.html#/appointments/post_agendamentos_cancelar__token_) | Cancelar um agendamento pelo token do email (convidado) |
| `GET` | [`/clients/me/agendamentos`](http://localhost:8080/swagger/index.html#/appointments/get_clients_me_agendamentos) | Listar agendamentos do cliente |
| `GET` | [`/providers/me/agendamentos`](http://localhost:8080/swagger/index.html#/appointments/get_providers_me_agendamentos) | Listar agendamentos recebidos pelo prestador |
| `POST` | [`/agendamentos/{id}/confirmar`](http://localhost:8080/swagger/index.html#/appointments/post_agendamentos__id__confirmar) | Confirmar uma solicitação (prestador) |
| `POST` | [`/agendamentos/{id}/recusar`](http://localhost:8080/swagger/index.html#/appointments/post_agendamentos__id__recusar) | Recusar uma solicitação (prestador) |
| `POST` | [`/agendamentos/{id}/cancelar`](http://localhost:8080/swagger/index.html#/appointments/post_agendamentos__id__cancelar) | Cancelar um agendamento (cliente ou prestador) |
| `POST` | [`/agendamentos/{id}/realizado`](http://localhost:8080/swagger/index.html#/appointments/post_agendamentos__id__realizado) | Marcar atendimento como realizado (prestador) |
| `POST` | [`/agendamentos/{id}/nao-compareceu`](http://localhost:8080/swagger/index.html#/appointments/post_agendamentos__id__nao_compareceu) | Registrar não comparecimento (prestador) |
| `GET` | [`/admin/prestadores`](http://localhost:8080/swagger/index.html#/admin/get_admin_prestadores) | Listar prestadores para moderação (admin) |
| `GET` | [`/admin/prestadores/{id}`](http://localhost:8080/swagger/index.html#/admin/get_admin_prestadores__id_) | Detalhar um prestador em leitura: cadastro + agendamentos recebidos (admin) |
| `GET` | [`/admin/clientes`](http://localhost:8080/swagger/index.html#/admin/get_admin_clientes) | Listar clientes para moderação (admin) |
| `GET` | [`/admin/clientes/{id}`](http://localhost:8080/swagger/index.html#/admin/get_admin_clientes__id_) | Detalhar um cliente em leitura: cadastro + agendamentos feitos (admin) |
| `POST` | [`/admin/prestadores/{id}/banir`](http://localhost:8080/swagger/index.html#/admin/post_admin_prestadores__id__banir) | Banir um prestador (admin) |
| `POST` | [`/admin/prestadores/{id}/reativar`](http://localhost:8080/swagger/index.html#/admin/post_admin_prestadores__id__reativar) | Reativar um prestador (admin) |
| `POST` | [`/admin/clientes/{id}/banir`](http://localhost:8080/swagger/index.html#/admin/post_admin_clientes__id__banir) | Banir um cliente (admin) |
| `POST` | [`/admin/clientes/{id}/reativar`](http://localhost:8080/swagger/index.html#/admin/post_admin_clientes__id__reativar) | Reativar um cliente (admin) |
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
- **[docs/producao.md](docs/producao.md)** — deploy: opções de hospedagem grátis/barata, passo a passo (VPS + Caddy), variáveis de ambiente e checklist de go-live

---

## Estrutura do projeto

Monorepo com arquitetura hexagonal no backend (`domain` → `usecase` → `adapter`) e SvelteKit no frontend:

```
agendaGo/
├── backend/    API em Go — cmd/, config/, internal/{domain,usecase,adapter}/, migrations/, test/
├── frontend/   SvelteKit — src/{lib,routes}/, e2e/
└── docs/       tecnologias.md · testes.md · regra-de-negocio.md · producao.md
```
