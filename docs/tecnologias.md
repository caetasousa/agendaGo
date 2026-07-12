# Tecnologias do agendaGo — guia de estudo

Este documento existe para quem quer entender **por que** cada peça do stack foi escolhida e **onde estudá-la a fundo**. Não é uma lista de dependências — é um roteiro de aprendizado organizado por camada, com ponteiros para o código real do projeto e fontes primárias (documentação oficial, RFCs, artigos de referência) em vez de tutoriais de terceiros.

---

## Visão geral

| Camada | Tecnologia | Papel no projeto | Versão |
|---|---|---|---|
| Linguagem (backend) | [Go](https://go.dev) | API HTTP, domínio, persistência | 1.26 |
| Arquitetura | Hexagonal (Ports & Adapters) | Organização do backend em `domain/usecase/adapter` | — |
| Roteamento HTTP | [chi](https://github.com/go-chi/chi) | Router e middlewares da API | v5.3 |
| Banco de dados | [PostgreSQL](https://www.postgresql.org/) | Persistência relacional | 16 (alpine) |
| Driver Postgres | [pgx](https://github.com/jackc/pgx) | Acesso ao banco a partir do Go | v5.10 |
| Migrations | [Flyway](https://flywaydb.org/) | Versionamento de schema | 10 |
| Hash de senha | [Argon2id](https://github.com/alexedwards/argon2id) | Armazenamento seguro de credenciais | v1.0 |
| Validação | [go-playground/validator](https://github.com/go-playground/validator) | Validação de DTOs de entrada | v10.30 |
| Documentação de API | [Swaggo](https://github.com/swaggo/swag) | OpenAPI/Swagger gerado a partir de comentários | v1.16 |
| Testes de integração | [Testcontainers](https://testcontainers.com/) | Postgres real e efêmero em cada teste | v0.43 |
| Envio de email | [go-mail](https://github.com/wneessen/go-mail) | Cliente SMTP para recuperação de senha e notificações | v0.8 |
| SMTP em desenvolvimento | [Mailpit](https://mailpit.axllent.org/) | Captura os emails localmente, sem enviar de verdade | — |
| Provedor SMTP (produção) | [Brevo](https://www.brevo.com/) | Envio real de email, plano gratuito (300/dia) | — |
| Framework (frontend) | [Svelte 5](https://svelte.dev) + [SvelteKit](https://kit.svelte.dev) | UI reativa com runes, roteamento file-based | 5.56 / 2.63 |
| Linguagem (frontend) | [TypeScript](https://www.typescriptlang.org/) | Tipagem estática no cliente | 6.0 |
| Build tool | [Vite](https://vitejs.dev/) | Dev server e bundler | 8.0 |
| Estilos | [Tailwind CSS](https://tailwindcss.com/) | Utility-first CSS | 4.3 |
| Testes unitários (frontend) | [Vitest](https://vitest.dev/) | Testes do cliente HTTP e da store de sessão | 4.1 |
| Testes E2E | [Playwright](https://playwright.dev/) | Fluxos de cadastro/login/sessão no browser real | 1.61 |
| Orquestração local | [Docker Compose](https://docs.docker.com/compose/) | Sobe banco + migrations + API + web juntos | — |

---

## 1. Linguagem e arquitetura do backend

### Go

Go é a linguagem escolhida pela simplicidade da sintaxe, tooling embutido (`go test`, `go fmt`, `go vet`) e concorrência nativa via goroutines — mesmo que o agendaGo ainda não explore concorrência pesada, é o tipo de decisão que compensa à medida que o projeto cresce (ex.: processar múltiplos agendamentos, notificações assíncronas).

**Para estudar:**
- [A Tour of Go](https://go.dev/tour/) — interativo, cobre a sintaxe do zero
- [Effective Go](https://go.dev/doc/effective_go) — como escrever Go idiomático (nomenclatura, erros, interfaces)
- [Go by Example](https://gobyexample.com/) — referência rápida por tópico

### Arquitetura Hexagonal (Ports & Adapters)

O backend é organizado em três camadas que só se enxergam por interfaces:

```
internal/domain/{provider,client,session}/   → regras de negócio puras, sem I/O
internal/usecase/{provider,client,auth}/     → orquestração; define as interfaces (ports)
                                                que os adapters implementam
internal/adapter/{http,repository,security}/ → HTTP, Postgres, Argon2id — os detalhes
```

O ganho prático: o domínio (`internal/domain/provider/provider.go`) não sabe que existe HTTP ou Postgres. Trocar o banco por outro, ou adicionar uma segunda forma de expor a API, não exige tocar em uma linha de regra de negócio. Repare como `internal/usecase/provider/repositorio.go` declara a interface `repositorioCadastrar` do lado de quem consome — não do lado do Postgres — o que é a marca registrada de Ports & Adapters (a interface pertence a quem precisa dela, não a quem a implementa).

**Para estudar:**
- [Hexagonal Architecture — Alistair Cockburn](https://alistair.cockburn.us/hexagonal-architecture/) (artigo original, quem cunhou o termo)
- [Ports & Adapters Pattern — Netflix TechBlog](https://netflixtechblog.com/ready-for-changes-with-hexagonal-architecture-b315ec967749) (aplicação prática em produção)

---

## 2. Backend HTTP

### chi

Router HTTP minimalista, compatível com a stdlib (`net/http`) — não reinventa `http.Handler`, só adiciona roteamento por padrão de URL, middlewares encadeáveis e grupos de rotas. Usado em `config/server.go` e no wiring de `cmd/api/main.go`, onde rotas autenticadas ficam agrupadas sob o middleware `Autenticar`.

**Para estudar:**
- [chi — README oficial](https://github.com/go-chi/chi#chi) (exemplos de roteamento e middleware)
- [net/http — pacote padrão do Go](https://pkg.go.dev/net/http) (a base sobre a qual o chi é construído)

### go-playground/validator

Validação declarativa via struct tags — em vez de escrever `if` para cada campo, o DTO descreve suas próprias regras (`validate:"required,email"`). Ver `internal/adapter/http/dto/provider.go` e `dto/client.go`.

**Para estudar:**
- [validator — documentação oficial](https://pkg.go.dev/github.com/go-playground/validator/v10) (lista completa de tags)

### Swaggo

Gera a especificação OpenAPI a partir de comentários no código (`@Summary`, `@Router` etc., visíveis em `internal/adapter/http/handler/provider.go`). A documentação nunca fica desatualizada em relação ao handler, porque é gerada dele.

**Para estudar:**
- [Swaggo — README oficial](https://github.com/swaggo/swag) (sintaxe das anotações)
- [OpenAPI Specification](https://swagger.io/specification/) (o formato por trás do Swagger)

---

## 3. Segurança e autenticação

Esta é a parte mais rica para estudo — o agendaGo implementa autenticação seguindo recomendações da OWASP, não um tutorial genérico.

### Hash de senha: Argon2id

Senhas nunca são armazenadas em texto puro (ver a migration `V2__renomeia_coluna_senha_para_senha_hash.sql` e `internal/adapter/security/argon2id.go`). Argon2id é o algoritmo **recomendado atualmente** pela OWASP para hash de senha — venceu a Password Hashing Competition (2015) justamente por ser resistente a ataques com hardware especializado (GPU/ASIC), já que seu custo é dominado por acesso à memória, não só processamento.

Os parâmetros usados (19 MiB, 2 iterações, salt de 16 bytes) seguem exatamente a recomendação mínima da OWASP para 2024+.

**Para estudar:**
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) (a referência prática nº 1)
- [RFC 9106 — Argon2 Memory-Hard Function](https://www.rfc-editor.org/rfc/rfc9106.html) (a especificação formal do algoritmo)

### Sessões server-side + cookie HttpOnly

O login (`internal/usecase/auth/login_provider.go`) gera um token opaco de 256 bits (`internal/pkg/token/token.go`), guarda apenas o **hash SHA-256** dele no banco (tabela `sessions`), e entrega o token puro só no cookie. Essa escolha — sessão em vez de JWT — foi deliberada: revogar uma sessão é `DELETE` de uma linha; revogar um JWT exige infraestrutura extra (blacklist, TTL curto + refresh token). Para uma aplicação web first-party como o agendaGo, sessão é mais simples **e** mais segura.

O atributo `HttpOnly` do cookie impede que JavaScript no browser leia o token — é a defesa de primeira linha contra roubo de sessão via XSS.

**Para estudar:**
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [MDN — Using HTTP cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies) (atributos `HttpOnly`, `Secure`, `SameSite`)
- [Auth0 — Sessions vs. Tokens](https://auth0.com/blog/is-jwt-better-than-session-authentication/) (comparação prática dos dois modelos)

### Timing attacks e enumeração de usuários

Repare em `internal/usecase/auth/auth.go`: quando o email não existe, o código ainda executa um `Verificar` contra um hash dummy antes de retornar erro. Sem isso, um invasor poderia medir o tempo de resposta e descobrir quais emails estão cadastrados (busca no banco + hash é mais lento que só retornar erro).

**Para estudar:**
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html) (seção sobre respostas genéricas de erro)

### Rate limiting: go-chi/httprate

Middleware de limitação de requisições por IP, da própria família do chi. Aplicado em `cmd/api/main.go` sobre as rotas de **login** (mitiga brute-force e rajadas de Argon2id, que é caro de CPU por design) e sobre o **agendamento de convidado** (rota pública — sem teto, uma rajada encheria a agenda de um prestador com reservas falsas). Os limites vêm de env vars (`RATE_LIMIT_*_POR_MINUTO`, 0 desliga — ver `config/server.go`); em dev ficam desligados porque os testes e2e disparam dezenas de logins do mesmo IP.

Complementa o limite de **tamanho de corpo** (`internal/adapter/http/middleware/body.go`, via `http.MaxBytesReader`): a API só troca JSONs pequenos, então qualquer corpo acima de 1 MiB é rejeitado antes de ocupar memória.

**Para estudar:**
- [go-chi/httprate](https://github.com/go-chi/httprate)
- [OWASP — Denial of Service Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Denial_of_Service_Cheat_Sheet.html)

---

## 4. Banco de dados e infraestrutura

### PostgreSQL

Banco relacional open-source, escolhido pela maturidade, suporte a `UUID` nativo e o ecossistema de ferramentas (Testcontainers, Flyway) que o projeto usa nos testes.

**Para estudar:**
- [PostgreSQL — Tutorial oficial](https://www.postgresql.org/docs/current/tutorial.html)

### pgx

Driver Postgres para Go — usado via `pgxpool` (pool de conexões) em vez de `database/sql` puro, porque expõe recursos específicos do protocolo Postgres com melhor performance. Ver `internal/adapter/repository/provider_postgres.go`.

**Para estudar:**
- [pgx — documentação oficial](https://pkg.go.dev/github.com/jackc/pgx/v5)

### Flyway

Cada mudança de schema é um arquivo SQL versionado (`backend/migrations/V1__...sql`, `V2__...sql`) aplicado em ordem, uma única vez, e nunca editado depois de mergeado. É o princípio de **migrations imutáveis**, que garante que qualquer ambiente (dev, CI, produção) chegue ao mesmo schema pela mesma sequência de passos.

**Para estudar:**
- [Flyway — Como funciona](https://documentation.red-gate.com/fd/migrations-184127470.html)

### Docker Compose

Orquestra Postgres + Flyway + API (com hot reload via [Air](https://github.com/air-verse/air)) + frontend em um único `docker compose up`, documentado no `docker-compose.yml` da raiz.

**Para estudar:**
- [Docker Compose — visão geral](https://docs.docker.com/compose/)

---

## 5. Frontend

### Svelte 5 (runes) e SvelteKit

Svelte se diferencia de React/Vue por ser um **compilador**: o código que você escreve vira JavaScript imperativo otimizado em build-time, sem Virtual DOM em runtime. A versão 5 introduziu **runes** (`$state`, `$derived`, `$effect`) — reatividade explícita via funções especiais, em vez de inferida pelo compilador a partir de atribuições. O agendaGo usa runes em todo o frontend, inclusive na store de sessão (`frontend/src/lib/stores/session.svelte.ts`), que é reatividade compartilhada fora de um componente `.svelte`.

SvelteKit é o meta-framework por cima do Svelte: roteamento baseado em arquivos (`src/routes/`), SSR por padrão, e arquivos `+page.ts` para lógica de carregamento de dados (`load`). O projeto desabilita SSR explicitamente em `/login`, `/cadastro` e `/painel` (`export const ssr = false`) — o motivo está documentado no próprio código: o cookie de sessão é `HttpOnly` e a API roda em outra origem, então o servidor de SSR nunca teria acesso a ele.

**Para estudar:**
- [Svelte 5 — documentação oficial](https://svelte.dev/docs/svelte/overview) (comece por "Runes")
- [SvelteKit — documentação oficial](https://svelte.dev/docs/kit/introduction)
- [Svelte 5 Runes — anúncio oficial](https://svelte.dev/blog/runes) (o "porquê" da mudança de paradigma)

### TypeScript

Tipagem estática sobre o JavaScript do frontend — os tipos em `frontend/src/lib/api/auth.ts` (`LoginResponse`, `MeResponse`) espelham deliberadamente os DTOs Go do backend, então uma mudança de contrato na API quebra a compilação do frontend em vez de falhar silenciosamente em runtime.

**Para estudar:**
- [TypeScript — Handbook oficial](https://www.typescriptlang.org/docs/handbook/intro.html)

### Vite

Dev server com Hot Module Replacement quase instantâneo e bundler de produção. É a ferramenta por trás de `npm run dev` e também hospeda a configuração de testes do Vitest (`frontend/vite.config.ts`).

**Para estudar:**
- [Vite — Guia oficial](https://vitejs.dev/guide/)

### Tailwind CSS 4

Framework utility-first: classes como `rounded-md border px-4` compõem o design diretamente no markup, sem alternar entre arquivo `.svelte` e arquivo `.css`. A versão 4 trouxe um motor CSS reescrito em Rust, bem mais rápido que a v3.

**Para estudar:**
- [Tailwind CSS — documentação oficial](https://tailwindcss.com/docs)

### adapter-node (build de produção)

O SvelteKit delega o formato do build final a um *adapter*. O projeto usa o `@sveltejs/adapter-node` (configurado em `frontend/vite.config.ts`): `npm run build` gera um servidor Node autônomo em `build/index.js`, empacotado na imagem `frontend/Dockerfile.prod`. Atenção ao detalhe de `PUBLIC_API_URL`: por ser `import.meta.env` do Vite, o valor é **embutido no build** — em produção ela é um argumento de build da imagem, não uma env de runtime. Ver `docs/producao.md`.

**Para estudar:**
- [SvelteKit — Adapters](https://svelte.dev/docs/kit/adapters)
- [SvelteKit — adapter-node](https://svelte.dev/docs/kit/adapter-node)

---

## 6. Testes

O agendaGo segue a pirâmide de testes clássica, visível na própria estrutura de pastas:

```
backend/test/
├── domain/       regras de negócio puras (mais rápidos, mais numerosos)
├── usecase/      orquestração com repositórios em memória
├── handler/      contrato HTTP via httptest
└── repository/   integração real contra Postgres (mais lentos, mais caros)

frontend/
├── src/lib/**/*.test.ts   unitários (Vitest)
└── e2e/                   fluxos completos no browser (Playwright)
```

### Testcontainers

Cada teste de integração (`backend/test/repository/provider_postgres_test.go`) sobe um container Postgres **efêmero e real** via Docker, aplica as migrations, roda o teste, e destrói o container. Isso elimina a categoria inteira de bugs "passou no mock, quebrou em produção" — o SQL que roda no teste é o mesmo SQL que roda no banco de verdade.

**Para estudar:**
- [Testcontainers for Go — Quickstart](https://golang.testcontainers.org/quickstart/)

### Vitest

Test runner para o frontend, com API compatível com Jest mas rodando sobre a infraestrutura do Vite (mesma configuração, sem duplicar setup). Os testes de `frontend/src/lib/api/auth.test.ts` mockam `fetch` para verificar o fallback de login (tenta prestador, cai para cliente) sem precisar de rede real.

**Para estudar:**
- [Vitest — Guia oficial](https://vitest.dev/guide/)

### Playwright

Framework de testes E2E que controla um browser real (Chromium/Firefox/WebKit). Os specs em `frontend/e2e/` cobrem os fluxos ponta-a-ponta que unitários não alcançam: cadastro → painel, sessão persistindo entre navegações, logout limpando o cookie.

**Para estudar:**
- [Playwright — documentação oficial](https://playwright.dev/docs/intro)

---

## 7. Notificações por email

### go-mail

Cliente SMTP para Go. A biblioteca padrão (`net/smtp`) está em modo *frozen* (sem novas features) e não lida bem com STARTTLS obrigatório nem com autenticação moderna — go-mail resolve isso com uma API pequena por cima do protocolo SMTP. Usado em `internal/adapter/email/smtp.go` (`MailerSMTP`), que monta a política de TLS (`TLSMandatory` em produção, `NoTLS` contra o Mailpit) e só ativa autenticação quando usuário/senha estão configurados.

**Para estudar:**
- [go-mail — documentação oficial](https://pkg.go.dev/github.com/wneessen/go-mail)
- [RFC 3207 — STARTTLS para SMTP](https://www.rfc-editor.org/rfc/rfc3207)

### Mailpit

Servidor SMTP fake para desenvolvimento: captura todo email enviado pela aplicação e mostra numa UI web (`http://localhost:8025`), sem entregar nada de verdade. Roda como serviço no `docker-compose.yml`; a API aponta para ele via `SMTP_HOST=mailpit` por padrão. A vantagem central é que o código de produção e o de desenvolvimento são **exatamente o mesmo** — só a env var `SMTP_HOST` muda.

**Para estudar:**
- [Mailpit — documentação oficial](https://mailpit.axllent.org/docs/)

### Brevo

Provedor de envio transacional de email escolhido pelo plano gratuito generoso (300 emails/dia, sem cartão de crédito) e configuração simples via SMTP puro — não exige domínio próprio verificado, só o email remetente. Ver `docs/regra-de-negocio.md` para o passo a passo de configuração da conta.

**Para estudar:**
- [Brevo — documentação da API SMTP](https://developers.brevo.com/docs/smtp-integration)

### Primeiro worker em background e envio assíncrono

Até esta feature, o único ponto concorrente do backend era o próprio `http.Server`; toda regra de negócio rodava síncrona dentro do request (inclusive expiração de sessões e de solicitações, resolvidas de forma *lazy* na leitura). O lembrete de agendamento (`internal/usecase/appointment/lembrar.go` + `internal/adapter/worker/reminder.go`) introduz o primeiro `time.Ticker` de fundo do projeto — não dava para resolver "avise 24h antes" de forma lazy, porque não existe uma leitura garantida naquele momento.

O envio de email em si (`internal/adapter/email/notificador.go`) também é assíncrono: o use case chama o notificador de forma síncrona, mas o adapter dispara o envio numa goroutine via um `executar func(func())` injetado — `ExecutorGoroutine` em produção (registrado num `sync.WaitGroup` compartilhado com o worker, para o desligamento gracioso esperar o que estiver pendente) e `ExecutorSincrono` nos testes (permite `assert` logo após o `Executar` do use case, sem `sleep`). Falha de envio nunca falha a operação que a disparou — só é logada.

**Para estudar:**
- [Go — pacote `sync` (WaitGroup)](https://pkg.go.dev/sync#WaitGroup)
- [Go — pacote `time` (Ticker)](https://pkg.go.dev/time#Ticker)

---

## Como usar este documento

Não precisa ler tudo de uma vez. Sugestão de ordem se você está começando do zero:

1. **Go** (Tour of Go) → depois olhe `internal/domain/provider/provider.go` para ver Go real
2. **Arquitetura Hexagonal** → releia a estrutura de pastas do backend com esse conceito em mente
3. **Argon2id + sessões** → é a parte mais "por que fizemos assim" do projeto
4. **Svelte 5 runes** → compare `session.svelte.ts` com qualquer store Redux/Vuex que você já tenha visto
5. **Testcontainers** → rode `make test-all` (documentado no README) e observe o container subindo nos logs
