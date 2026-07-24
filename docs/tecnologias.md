# Tecnologias do agendaGo — guia de estudo

Este documento existe para quem quer entender **por que** cada peça do stack foi escolhida e **onde estudá-la a fundo**. Não é uma lista de dependências — é um roteiro de aprendizado organizado por camada, com ponteiros para o código real do projeto e fontes primárias (documentação oficial, RFCs, artigos de referência) em vez de tutoriais de terceiros.

---

## Visão geral

| Camada | Tecnologia | Papel no projeto | Versão |
|---|---|---|---|
| Linguagem (backend) | [Go](https://go.dev) | API HTTP, domínio, persistência | 1.26 |
| Arquitetura | Hexagonal (Ports & Adapters) | Organização do backend em `domain/usecase/adapter` | — |
| Roteamento HTTP | [chi](https://github.com/go-chi/chi) | Router e middlewares da API | v5.3 |
| CORS | [go-chi/cors](https://github.com/go-chi/cors) | Controle de origens permitidas nas respostas da API | v1.2 |
| Rate limiting | [go-chi/httprate](https://github.com/go-chi/httprate) | Teto de requisições por IP (login, convidado, tokens) | v0.16 |
| Logging | [log/slog](https://pkg.go.dev/log/slog) | Logs estruturados (JSON em produção) com correlação por requisição | stdlib |
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
| Proxy reverso (produção) | [Caddy](https://caddyserver.com/) | HTTPS automático e origem única para frontend e API | 2 (alpine) |

---

## 1. Linguagem e arquitetura do backend

### Go

Go é a linguagem escolhida pela simplicidade da sintaxe, tooling embutido (`go test`, `go fmt`, `go vet`) e concorrência nativa via goroutines — mesmo que o agendaGo ainda não explore concorrência pesada, é o tipo de decisão que compensa à medida que o projeto cresce (ex.: processar múltiplos agendamentos, notificações assíncronas).

**Para estudar:**
- [A Tour of Go](https://go.dev/tour/) — interativo, cobre a sintaxe do zero
- [Effective Go](https://go.dev/doc/effective_go) — como escrever Go idiomático (nomenclatura, erros, interfaces)
- [Go by Example](https://gobyexample.com/) — referência rápida por tópico
- [How to Write Go Code](https://go.dev/doc/code) — módulos, pacotes e a estrutura de um projeto Go
- [Go Proverbs](https://go-proverbs.github.io/) — os princípios de design da linguagem, por Rob Pike

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

Senhas nunca são armazenadas em texto puro: a coluna `senha_hash` guarda só o hash (ver `internal/adapter/security/argon2id.go` e as migrations `V1__cria_tabela_providers.sql`/`V2__cria_tabela_clients.sql`, que já nascem com essa coluna). Argon2id é o algoritmo **recomendado atualmente** pela OWASP para hash de senha — venceu a Password Hashing Competition (2015) justamente por ser resistente a ataques com hardware especializado (GPU/ASIC), já que seu custo é dominado por acesso à memória, não só processamento.

Os parâmetros usados (19 MiB, 2 iterações, salt de 16 bytes) seguem exatamente a recomendação mínima da OWASP para 2024+.

**Para estudar:**
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) (a referência prática nº 1)
- [RFC 9106 — Argon2 Memory-Hard Function](https://www.rfc-editor.org/rfc/rfc9106.html) (a especificação formal do algoritmo)
- [Password Hashing Competition](https://www.password-hashing.net/) (o concurso que elegeu o Argon2, com os finalistas e critérios)

### Sessões server-side + cookie HttpOnly

O login (`internal/usecase/auth/login_provider.go`) gera um token opaco de 256 bits (`internal/pkg/token/token.go`), guarda apenas o **hash SHA-256** dele no banco (tabela `sessions`), e entrega o token puro só no cookie. Essa escolha — sessão em vez de JWT — foi deliberada: revogar uma sessão é `DELETE` de uma linha; revogar um JWT exige infraestrutura extra (blacklist, TTL curto + refresh token). Para uma aplicação web first-party como o agendaGo, sessão é mais simples **e** mais segura.

O atributo `HttpOnly` do cookie impede que JavaScript no browser leia o token — é a defesa de primeira linha contra roubo de sessão via XSS.

**Para estudar:**
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [MDN — Using HTTP cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies) (atributos `HttpOnly`, `Secure`, `SameSite`)
- [Auth0 — Sessions vs. Tokens](https://auth0.com/blog/is-jwt-better-than-session-authentication/) (comparação prática dos dois modelos)

### Login social: OpenID Connect (OIDC)

O login com Google (`internal/adapter/oauth/google.go`, `internal/usecase/auth/login_social.go`) usa **OpenID Connect**, o protocolo padrão de identidade construído sobre o OAuth 2.0 — o OAuth por si só autoriza acesso a um recurso (ex.: "ler minha agenda"), mas não prova identidade; o OIDC adiciona o `id_token`, um JWT assinado pelo provedor que atesta quem é o usuário. O fluxo aqui é o **Authorization Code Flow**: o backend redireciona ao Google (`/auth/client|provider/google/start`), recebe um código de uso único no callback, e o troca (server-to-server) por um `id_token`, que é então **verificado** contra as chaves públicas (JWKS) do Google via `github.com/coreos/go-oidc` — nunca confiamos no token sem essa verificação criptográfica.

Duas proteções do fluxo valem o estudo: o parâmetro **`state`** (gerado com o mesmo `token.Gerar/Hash` das sessões, guardado num cookie curto e numa tabela de uso único `oauth_states`) evita CSRF — sem ele, um atacante poderia induzir a vítima a completar o login de uma sessão iniciada por ele; o **`nonce`** embutido no `id_token` evita replay do mesmo token em outra sessão. Como o Google não fornece telefone e o domínio exige um valor para prestadores, a criação via login social usa um telefone placeholder (`internal/usecase/auth/login_social.go`) que o prestador completa depois em Preferências — e como o domínio de `Client`/`Provider` exige uma senha, uma senha aleatória de 256 bits é gerada e hasheada (nunca comunicada) só para satisfazer essa invariante.

**Para estudar:**
- [OpenID Connect — site oficial](https://openid.net/developers/how-connect-works/) (visão geral do protocolo sobre OAuth 2.0)
- [Google Identity — OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect) (a implementação específica que o projeto consome)
- [RFC 6749 — The OAuth 2.0 Authorization Framework](https://www.rfc-editor.org/rfc/rfc6749) (a base sobre a qual o OIDC é construído)
- [OWASP — Unvalidated Redirects and Forwards Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Unvalidated_Redirects_and_Forwards_Cheat_Sheet.html) (por que o `?voltar=` do callback só aceita caminhos internos)

### Tokens de uso único (recuperação de senha, confirmação de cadastro, cancelamento)

O mesmo padrão da sessão — token opaco de 256 bits, guardado só como hash SHA-256 — reaparece em quatro fluxos por email, cada um numa tabela própria (`password_reset_tokens`, `cadastros_pendentes`, `pre_cadastro_tokens`, `cancelamento_tokens`). Três decisões de ciclo de vida se repetem em todos e valem o estudo:

- **Expiração**: todo token tem prazo (`expira_em`) — 1h para recuperação de senha, 24h para confirmação/pré-cadastro. Um segredo que cria conta ou troca senha não pode valer para sempre.
- **Uso único de verdade**: o consumo é atômico via `DELETE ... RETURNING` (ver `internal/adapter/repository/*_postgres.go`), então o token some no mesmo instante em que é usado, mesmo sob concorrência — não dá para reusar o link.
- **Limpeza**: um worker em background (`internal/adapter/worker/cleanup.go`) remove periodicamente os tokens vencidos, para PII de contato não se acumular indefinidamente no banco.

Repare também na postura **anti-enumeração**: o cadastro e a recuperação de senha respondem sempre igual, exista ou não o email (o aviso "você já tem conta" vai por email, não na resposta HTTP), para não revelar quais endereços estão cadastrados.

**Para estudar:**
- [OWASP Forgot Password Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html) (token de uso único, expiração, resposta genérica)
- [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html) (por que guardar o hash, não o token)

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
- [Use The Index, Luke!](https://use-the-index-luke.com/) — como índices funcionam (relevante para os índices em `expira_em`, `email` etc. das migrations)

### pgx

Driver Postgres para Go — usado via `pgxpool` (pool de conexões) em vez de `database/sql` puro, porque expõe recursos específicos do protocolo Postgres com melhor performance. Ver `internal/adapter/repository/provider_postgres.go`.

**Para estudar:**
- [pgx — documentação oficial](https://pkg.go.dev/github.com/jackc/pgx/v5)

### Flyway

Cada mudança de schema é um arquivo SQL versionado (`backend/migrations/V1__...sql`, `V2__...sql`) aplicado em ordem, uma única vez, e nunca editado depois de mergeado. É o princípio de **migrations imutáveis**, que garante que qualquer ambiente (dev, CI, produção) chegue ao mesmo schema pela mesma sequência de passos.

**Para estudar:**
- [Flyway — Como funciona](https://documentation.red-gate.com/fd/migrations-184127470.html)

### Docker Compose

Orquestra Postgres + Flyway + API (com hot reload via [Air](https://github.com/air-verse/air)) + frontend em um único `docker compose up`, documentado no `docker-compose.yml` da raiz. Produção tem um compose próprio (`docker-compose.prod.yml`) com as imagens `Dockerfile.prod` e o Caddy na frente — ver `docs/producao.md`.

**Para estudar:**
- [Docker Compose — visão geral](https://docs.docker.com/compose/)
- [Docker — build multi-stage](https://docs.docker.com/build/building/multi-stage/) (como o `Dockerfile.prod` gera uma imagem final mínima)

### Caddy (proxy reverso de produção)

Em produção, um único Caddy termina o TLS (certificado Let's Encrypt **automático**) e serve frontend e API na **mesma origem**: `/api/*` vai para a API (o prefixo é removido antes de repassar) e o resto para o frontend. Origem única não é detalhe estético — é o que faz o cookie de sessão `SameSite=Lax` ser enviado nas chamadas do front para a API sem precisar mudar código nem afrouxar o cookie para `SameSite=None`. Ver `Caddyfile` e `docs/producao.md`.

**Para estudar:**
- [Caddy — Getting Started](https://caddyserver.com/docs/getting-started)
- [Caddy — HTTPS automático](https://caddyserver.com/docs/automatic-https)
- [MDN — SameSite cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite) (por que a mesma origem importa)

### Logging estruturado: log/slog

O sistema usa o `log/slog` (structured logging da biblioteca padrão, Go 1.21+) em vez do `log` puro — configurado em `internal/pkg/logging/logging.go`. Em **produção** emite JSON (uma linha = um objeto, parseável campo a campo por agregadores como Loki/CloudWatch/Datadog); em **desenvolvimento**, texto legível no terminal. A escolha entre os dois é `APP_ENV=production`.

Três decisões tornam o log útil em produção, não só ruído:

- **Correlação por requisição**: um middleware (`middleware.RequestID` do chi) gera um `request_id` por requisição, e `logging.RequisicaoLogger(r)` anexa esse id (mais a rota) a todos os logs daquela requisição — inclusive o log de acesso e o log de erro, que passam a casar pelo mesmo id.
- **O erro real nunca some**: o cliente recebe sempre `{"erro":"erro interno"}` num 500 (não vaza detalhes internos), mas `responderErroInterno` (`internal/adapter/http/handler/provider.go`) loga o erro de verdade (ex.: falha de conexão com o Postgres) em nível ERROR, com o request_id — sem isso, um 500 em produção seria uma caixa-preta.
- **Rota, não caminho**: o log de acesso registra o padrão da rota (`/agendamentos/cancelar/{token}`), não o caminho real, para que tokens em path não vão parar nos logs.

Eventos de segurança (login falho, tentativa de conta banida) saem em **WARN** com tipo/email/IP — a senha nunca é logada —, para permitir detectar brute-force. O IP real do cliente chega via `X-Real-IP` que o Caddy define de forma não-forjável (`internal/adapter/http/middleware/real_ip.go`); sem isso, atrás do proxy, tanto o log quanto o rate limit por IP veriam só o IP do container do Caddy.

**Para estudar:**
- [Go — pacote log/slog](https://pkg.go.dev/log/slog) (handlers, níveis, atributos estruturados)
- [The Twelve-Factor App — Logs](https://12factor.net/logs) (por que logar para stdout como fluxo de eventos, não para arquivo)
- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) (o que logar — e o que nunca logar — em eventos de segurança)

---

## 5. Frontend

### Svelte 5 (runes) e SvelteKit

Svelte se diferencia de React/Vue por ser um **compilador**: o código que você escreve vira JavaScript imperativo otimizado em build-time, sem Virtual DOM em runtime. A versão 5 introduziu **runes** (`$state`, `$derived`, `$effect`) — reatividade explícita via funções especiais, em vez de inferida pelo compilador a partir de atribuições. O agendaGo usa runes em todo o frontend, inclusive na store de sessão (`frontend/src/lib/stores/session.svelte.ts`), que é reatividade compartilhada fora de um componente `.svelte`.

SvelteKit é o meta-framework por cima do Svelte: roteamento baseado em arquivos (`src/routes/`), SSR por padrão, e arquivos `+page.ts` para lógica de carregamento de dados (`load`). O projeto desabilita SSR explicitamente em páginas como `/login`, `/cadastro` e `/redefinir-senha` (`export const ssr = false`) — o motivo está no próprio código: com SSR existe uma janela em que o HTML já chegou mas o JavaScript ainda não hidratou os handlers, e um clique no formulário nesse instante dispararia o submit nativo (GET com os campos na URL) em vez do handler `onsubmit`. Renderizar só no cliente elimina essa janela. Além disso, esses fluxos dependem do cookie de sessão `HttpOnly`, que o servidor de SSR não enxerga.

> Em **produção**, frontend e API ficam atrás do mesmo proxy (mesma origem — ver `docs/producao.md`), então o cookie `SameSite=Lax` é enviado normalmente nas chamadas do front para a API. Em desenvolvimento eles rodam em portas diferentes.

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
