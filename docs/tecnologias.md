# 🧰 Tecnologias do agendaGo — guia de estudo

> Este documento existe para quem quer entender **por que** cada peça do stack foi
> escolhida e **onde estudá-la a fundo**.

Não é uma lista de dependências — é um **roteiro de aprendizado** organizado por
camada, com ponteiros para o código real do projeto e **fontes primárias**
(documentação oficial, RFCs, artigos de referência) em vez de tutoriais de
terceiros.

| Seção | O que cobre |
|---|---|
| [1. Linguagem e arquitetura](#1-linguagem-e-arquitetura-do-backend) | Go, hexagonal |
| [2. Backend HTTP](#2-backend-http) | chi, validator, Swaggo |
| [3. Segurança](#3-segurança-e-autenticação) | Argon2id, sessões, OIDC, rate limiting |
| [4. Banco e infra](#4-banco-de-dados-e-infraestrutura) | Postgres, pgx, Flyway, Docker, Caddy, GHCR, slog |
| [5. Frontend](#5-frontend) | Svelte 5, TypeScript, Vite, Tailwind, adapter-node |
| [6. Testes](#6-testes) | Go testing, Testcontainers, Vitest, Playwright |
| [7. Email](#7-notificações-por-email) | go-mail, Mailpit, Brevo, worker |

---

<details>
<summary><h2>📋 Visão geral do stack</h2></summary>

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

</details>

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

```mermaid
flowchart TD
    subgraph EXT["🌍 Mundo externo"]
        HTTP["🌐 HTTP"]
        PG[("🐘 Postgres")]
        SMTP["📧 SMTP"]
    end

    subgraph AD["🔌 adapter/ — os detalhes"]
        H["http/"]
        R["repository/"]
        S["security/ · email/"]
    end

    subgraph UC["🔀 usecase/ — orquestração"]
        U["define as <b>interfaces</b> (ports)<br/>que os adapters implementam"]
    end

    subgraph DM["🧠 domain/ — regras puras"]
        D["sem I/O, sem framework,<br/>sem saber que HTTP existe"]
    end

    HTTP --> H
    H --> U
    U --> D
    U -.->|"port"| R
    U -.->|"port"| S
    R --> PG
    S --> SMTP

    style DM fill:#2e7d32,color:#fff
    style UC fill:#1565c0,color:#fff
    style AD fill:#ef6c00,color:#fff
    style EXT fill:#455a64,color:#fff
```

| Camada | Pasta | Regra |
|---|---|---|
| 🧠 **Domínio** | `internal/domain/` | regras de negócio puras, **sem I/O** |
| 🔀 **Usecase** | `internal/usecase/` | orquestração; **declara** as interfaces que precisa |
| 🔌 **Adapter** | `internal/adapter/` | HTTP, Postgres, Argon2id, SMTP — os detalhes |

> [!IMPORTANT]
> **A interface pertence a quem precisa dela, não a quem a implementa.** Repare
> que `internal/usecase/provider/repositorio.go` declara `repositorioCadastrar`
> do lado de quem **consome** — não do lado do Postgres. Essa inversão é a marca
> registrada de Ports & Adapters.

O ganho prático: o domínio (`internal/domain/provider/provider.go`) não sabe que existe HTTP ou Postgres. Trocar o banco por outro, ou adicionar uma segunda forma de expor a API, não exige tocar em uma linha de regra de negócio.

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

É **ferramenta de desenvolvimento**: a UI publica a superfície inteira da API para quem alcançar a porta. Por isso `config.NovoRouter` só monta `/swagger/*` fora de produção — em produção a rota não existe, e o 404 do Caddy vira segunda linha de defesa em vez de única (a API pode ser servida por outro proxy, ou alcançada de dentro da rede do compose). O teste `test/config/server_test.go` trava esse comportamento.

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

Em produção o cookie ganha ainda o prefixo **`__Host-`** (`internal/adapter/http/handler/cookie.go`): é um contrato com o navegador, que só aceita um cookie com esse nome se ele vier com `Secure`, `Path=/` e **sem** atributo `Domain`. O efeito prático é amarrar a sessão a esta origem exata — um subdomínio comprometido não consegue escrever um cookie que o domínio principal aceite. Em desenvolvimento o prefixo não existe, porque sem HTTPS não há `Secure` e o navegador recusaria o cookie inteiro.

As respostas das rotas autenticadas saem com **`Cache-Control: no-store`** (`internal/adapter/http/middleware/cache.go`): sem esse cabeçalho, o navegador pode guardar em disco, por heurística própria, JSONs com dados pessoais — o histórico de agendamentos com nome, email e telefone de clientes, por exemplo.

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

A mesma disciplina vale para os **dois** cadastros (`internal/usecase/client/solicitar_cadastro.go` e `internal/usecase/provider/cadastrar.go`): a resposta é sempre a mesma, exista ou não o email, e o que muda é só a mensagem que sai por email — link de confirmação para quem é dono de um endereço livre, aviso de "esse email já está em uso" para o resto. A senha é hasheada em todos os caminhos, inclusive nos que não criam nada, para o tempo de resposta não denunciar o desfecho.

O cadastro de prestador seguiu por um tempo o caminho oposto (criava a conta na hora e respondia `409 email já cadastrado`), e isso custava duas coisas: qualquer um podia sondar quem tem conta, e dava para publicar na vitrine um prestador com o email de outra pessoa, já que ninguém provava posse do endereço.

**Para estudar:**
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html) (seção sobre respostas genéricas de erro)
- [OWASP WSTG — Testing for Account Enumeration](https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/03-Identity_Management_Testing/04-Testing_for_Account_Enumeration_and_Guessable_User_Account) (como se testa isso de fora)

### Rate limiting em três chaves: go-chi/httprate

Middleware de limitação de requisições da própria família do chi. O ponto que vale estudar aqui não é a biblioteca — é **por qual chave** contar, já que cada uma tapa um buraco que as outras deixam aberto:

| Chave | Onde | O que pega |
|---|---|---|
| **IP** | login, cadastro, convidado e leituras públicas | brute-force simples, rajadas de Argon2id (caro de CPU por design), raspagem da vitrine |
| **Conta** (email) | dentro dos handlers de login e recuperação de senha | o atacante com endereços de sobra — uma botnet ou uma faixa IPv6 inteira nunca encosta no teto por IP, porque cada tentativa chega de um endereço novo |
| **Sessão** (usuário autenticado) | rotas de escrita depois do login | abuso por quem já entrou, quando o IP deixa de identificar alguém |

Duas decisões de projeto no teto por conta (`internal/adapter/http/handler/ratelimit.go`): só **tentativa fracassada** é contabilizada, para o uso normal nunca aproximar a conta do teto; e a resposta 429 é idêntica para email existente e inexistente, senão o próprio bloqueio viraria um jeito de descobrir quem tem conta.

Estourado o teto, a janela barra **também a senha certa** — barrar só a senha errada não pararia o atacante que acerta na enésima tentativa. Isso admite um abuso conhecido: quem sabe o email de alguém consegue mantê-lo trancado enquanto insistir. A escolha é deliberada — a janela é curta e se renova sozinha, então o pior caso é um atraso de minutos, contra o risco de uma conta tomada.

O IP vem do `X-Real-IP` que o Caddy escreve (`internal/adapter/http/middleware/real_ip.go`) — o cliente não consegue forjá-lo, porque o proxy sobrescreve o cabeçalho. A chave é canonizada com `httprate.CanonicalizeIP`, que agrupa a faixa /64 de um cliente IPv6: sem isso, trocar de endereço dentro da própria faixa zeraria o contador.

Os limites vêm de env vars (`RATE_LIMIT_*`, 0 desliga cada um — ver `config/server.go`); os testes e2e desligam todos, porque a suíte dispara centenas de requisições do mesmo IP em poucos minutos.

Complementa o limite de **tamanho de corpo** (`internal/adapter/http/middleware/body.go`, via `http.MaxBytesReader`): a API só troca JSONs pequenos, então qualquer corpo acima de 1 MiB é rejeitado antes de ocupar memória.

**Para estudar:**
- [go-chi/httprate](https://github.com/go-chi/httprate)
- [OWASP — Denial of Service Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Denial_of_Service_Cheat_Sheet.html)
- [OWASP Authentication Cheat Sheet — bloqueio de conta](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html#account-lockout) (o trade-off entre travar a conta e deixar o brute-force correr)

### Content-Security-Policy

A CSP é a rede que segura um XSS que escape de tudo o mais — uma dependência comprometida, um `@html` distraído. Ela é declarada em `frontend/vite.config.ts` (`kit.csp`, com `mode: 'hash'`) e não no Caddy: o SvelteKit gera um `<script>` inline por build, calcula o hash dele e o publica na política, o que permite `script-src 'self'` sem `unsafe-inline`. Uma CSP estática no proxy teria de liberar todo inline para não quebrar a cada build — e duas CSPs na mesma resposta valem pela interseção, então a mais frouxa não ajudaria e a mais rígida quebraria o app.

Detalhe que só aparece na prática: o SvelteKit **só** hasheia os scripts que ele mesmo gera. O script de tema, que precisa rodar antes da primeira pintura, teve de sair do `app.html` para um arquivo estático (`frontend/static/tema.js`) — inline, seria bloqueado.

No Caddy ficam os cabeçalhos que não dependem do build: HSTS, `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy` e `Cross-Origin-Opener-Policy`.

**Para estudar:**
- [MDN — Content Security Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
- [SvelteKit — Content Security Policy](https://svelte.dev/docs/kit/configuration#csp) (o modo `hash` e o que ele cobre)
- [OWASP Secure Headers Project](https://owasp.org/www-project-secure-headers/) (o conjunto completo, com o que cada cabeçalho resolve)

### Varredura de dependências: govulncheck, npm audit, Trivy e Dependabot

Num projeto pequeno em produção, a via mais provável de comprometimento não é uma falha escrita aqui — é uma CVE numa biblioteca que ninguém percebeu que envelheceu. O CI (`.github/workflows/ci.yml`, job `seguranca`) cobre as três camadas onde isso mora:

- **[govulncheck](https://go.dev/blog/govulncheck)** — o diferencial dele é a análise de alcançabilidade: só acusa vulnerabilidade em código **realmente chamado** a partir do binário, em vez de listar tudo que existe na árvore de dependências. Por isso o que ele aponta trava o deploy. Cobre também a stdlib da versão de Go usada no build.
- **`npm audit`** — roda duas vezes: `--omit=dev` (só o que vai para a imagem) travando o CI, e completo em modo relatório, porque uma CVE no Vite ou no Playwright não roda em produção e não pode barrar um deploy de correção.
- **[Trivy](https://trivy.dev/)** — varre a imagem pronta: sistema base (alpine, node) e bibliotecas de sistema, que os dois anteriores não enxergam. `CRITICAL` trava a publicação; `HIGH` sai como relatório.
- **[Dependabot](https://docs.github.com/en/code-security/dependabot)** (`.github/dependabot.yml`) — apontar não corrige: sem alguém abrindo o PR, a dependência envelhece até virar incidente. Semanal para gomod, npm, GitHub Actions e as imagens base dos Dockerfiles.

**Para estudar:**
- [Go — Vulnerability Management](https://go.dev/doc/security/vuln/) (como o banco de vulnerabilidades e a análise de chamadas funcionam)
- [OWASP — Vulnerable and Outdated Components](https://owasp.org/Top10/A06_2021-Vulnerable_and_Outdated_Components/) (o item do Top 10 que essas ferramentas atacam)

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

### Paginação: LIMIT/OFFSET e o `total`

Vitrine, painel de moderação e histórico de agendamentos são listas que só crescem. Enquanto a base é pequena, uma consulta sem teto parece inofensiva — e é justamente por isso que ela sobrevive até o dia em que buscar "todos os agendamentos" significa carregar anos de histórico na memória a cada abertura de tela. Pior: numa rota pública, isso é o jeito mais barato de derrubar a API.

Toda listagem passa por `internal/pkg/paging`: limite padrão de 100, teto de 200, e um `Valida()` que os repositórios chamam antes de montar o SQL — uma `Pagina{}` zerada que escapasse de um caller viraria `LIMIT 0` e devolveria lista vazia calada, o tipo de defeito que só aparece em produção com o usuário achando que perdeu os dados. A resposta devolve `total` junto dos itens: é comparando o acumulado com o total que a tela sabe se ainda há o que carregar, sem adivinhar pelo tamanho da página.

Dois detalhes que a paginação obriga a acertar:

- **Ordem estável.** Todo `ORDER BY` termina em `id`. Sem desempate, duas linhas de mesmo nome (ou mesma data) podem trocar de posição entre uma página e a seguinte, e o item que estava na fronteira aparece duas vezes ou some.
- **Filtro no SQL, não em memória.** A vitrine mostra só prestadores ativos. Enquanto a consulta trazia tudo, dava para descartar os banidos no Go; com `LIMIT`, esse filtro devolveria páginas mais curtas que o pedido e esconderia prestadores válidos das páginas seguintes (ver `ListarAtivos` em `internal/adapter/repository/provider_postgres.go`).

**Para estudar:**
- [Use The Index, Luke! — Paginação](https://use-the-index-luke.com/sql/partial-results/fetch-next-page) (por que OFFSET fica caro e o que é paginação por keyset)

### Flyway

Cada mudança de schema é um arquivo SQL versionado (`backend/migrations/V1__...sql`, `V2__...sql`) aplicado em ordem, uma única vez, e nunca editado depois de mergeado. É o princípio de **migrations imutáveis**, que garante que qualquer ambiente (dev, CI, produção) chegue ao mesmo schema pela mesma sequência de passos.

**Para estudar:**
- [Flyway — Como funciona](https://documentation.red-gate.com/fd/migrations-184127470.html)

### Docker Compose

Orquestra Postgres + Flyway + API (com hot reload via [Air](https://github.com/air-verse/air)) + frontend em um único `docker compose up`, documentado no `docker-compose.yml` da raiz. Produção tem um compose próprio (`docker-compose.prod.yml`) que **não builda nada**: consome as imagens já publicadas no GHCR e põe o Caddy na frente — ver `docs/producao.md`.

**Para estudar:**
- [Docker Compose — visão geral](https://docs.docker.com/compose/)
- [Docker — build multi-stage](https://docs.docker.com/build/building/multi-stage/) (como o `Dockerfile.prod` gera uma imagem final mínima)

### Contenção dos containers e usuário de banco sem DDL

Container não é fronteira de segurança por si só — ele é o que o `docker run` deixou passar. O compose de produção fecha o que a aplicação não usa:

| Ajuste | O que muda |
|---|---|
| `no-new-privileges:true` | um binário `setuid` dentro da imagem não consegue escalar privilégio |
| `cap_drop: [ALL]` | API e frontend rodam sem capability nenhuma; Postgres fica com as cinco que o entrypoint precisa para largar privilégio, e o Caddy só com `NET_BIND_SERVICE`, para abrir as portas 80/443 |
| `read_only: true` + `tmpfs: /tmp` | API e frontend não escrevem em disco: um comprometimento não deixa nada gravado no container |
| `pids_limit` / `mem_limit` | um laço acidental (ou provocado) não leva o host junto |

Do lado do banco, a API se conecta com um usuário que **só faz DML** — sem `CREATE`, `ALTER` ou `DROP` (`scripts/criar-usuario-app.sh`). Quem aplica migration continua sendo o dono do banco, pelo Flyway. É o princípio do menor privilégio aplicado ao ponto que mais interessa: uma falha de execução remota na API não vira poder de derrubar tabela.

**Para estudar:**
- [Docker — Security](https://docs.docker.com/engine/security/) (capabilities, no-new-privileges e o modelo de isolamento)
- [OWASP — Docker Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html)
- [PostgreSQL — GRANT](https://www.postgresql.org/docs/current/sql-grant.html) (e `ALTER DEFAULT PRIVILEGES`, que estende os GRANTs às tabelas que ainda vão nascer)

### Registry de imagens: GHCR (GitHub Container Registry)

O CI builda as três imagens de produção (`agendago-api`, `agendago-web`, `agendago-migrations`) e publica no GHCR a cada push na `main` que passa nos testes — job `publicar-imagens` em `.github/workflows/ci.yml`. É por isso que o `docker-compose.prod.yml` só tem `image:` e nenhum `build:`.

A decisão é sobre **o que precisa existir no servidor**. Buildando no host, a VPS precisaria do repositório inteiro, de toolchain Go e Node, e de RAM/CPU para compilar — num plano de 1 vCPU, cada deploy competiria com o site no ar por minutos. Puxando imagem pronta, o host guarda três arquivos (`docker-compose.prod.yml`, `Caddyfile`, `.env`), o deploy é um `pull` de segundos, e o que roda em produção é exatamente o artefato que passou no CI — não uma recompilação que pode divergir. Como brinde, a tag por SHA do commit dá rollback sem rebuild.

Detalhe que morde: pacotes no GHCR nascem **privados** mesmo em repositório público — ou a visibilidade é trocada nas configurações do pacote, ou o host precisa de `docker login` com um PAT de `read:packages`.

**Para estudar:**
- [GitHub Packages — Working with the Container registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [docker/build-push-action](https://github.com/docker/build-push-action)
- [GitHub Actions — permissões do GITHUB_TOKEN](https://docs.github.com/en/actions/security-for-github-actions/security-guides/automatic-token-authentication) (por que o job declara `packages: write`)

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

O SvelteKit delega o formato do build final a um *adapter*. O projeto usa o `@sveltejs/adapter-node` (configurado em `frontend/vite.config.ts`): `npm run build` gera um servidor Node autônomo em `build/index.js`, empacotado na imagem `frontend/Dockerfile.prod`. Atenção ao detalhe de `PUBLIC_API_URL`: por ser `import.meta.env` do Vite, o valor é **embutido no build** — é um argumento de build da imagem, não uma env de runtime.

Dois detalhes que já custaram caro nesse caminho:

- **`envPrefix: 'PUBLIC_'` em `frontend/vite.config.ts` é obrigatório.** O Vite só expõe variáveis com prefixo `VITE_` por padrão, e o plugin do SvelteKit **não** ajusta isso para quem usa `import.meta.env` (o prefixo `PUBLIC_` só é convenção nos módulos `$env/*`). Sem a linha, `import.meta.env.PUBLIC_API_URL` sai `undefined` e o `frontend/src/lib/api/client.ts` cai calado no fallback `http://localhost:8080` — o build passa, os testes passam, e só o site publicado quebra.
- **O valor padrão é `/api`, relativo.** Uma URL absoluta amarraria a imagem a um domínio, o que impede publicá-la pronta no registry. Como o Caddy serve frontend e API na mesma origem, o caminho relativo resolve no domínio corrente; funciona porque toda página que consome a API declara `ssr = false`, então o `fetch` nunca roda no servidor Node (onde não haveria base para resolver o caminho).

Ver `docs/producao.md`.

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

## 🎓 Como usar este documento

Não precisa ler tudo de uma vez. Sugestão de ordem se você está começando do zero:

1. **Go** (Tour of Go) → depois olhe `internal/domain/provider/provider.go` para ver Go real
2. **Arquitetura Hexagonal** → releia a estrutura de pastas do backend com esse conceito em mente
3. **Argon2id + sessões** → é a parte mais "por que fizemos assim" do projeto
4. **Svelte 5 runes** → compare `session.svelte.ts` com qualquer store Redux/Vuex que você já tenha visto
5. **Testcontainers** → rode `make test-all` (documentado no README) e observe o container subindo nos logs
