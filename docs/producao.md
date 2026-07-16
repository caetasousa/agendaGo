# Produção

Como colocar o agendaGo no ar. O `docker-compose.yml` da raiz é **só para dev**
(hot reload, portas expostas, Mailpit, rate limit desligável). Produção usa o
**`docker-compose.prod.yml`**, que sobe a stack inteira atrás de um proxy Caddy
com HTTPS automático.

## Arquitetura no ar

```
                    Internet (HTTPS)
                          │
                    ┌─────▼─────┐
                    │   Caddy   │  TLS automático (Let's Encrypt)
                    │  :80/:443 │  origem única
                    └─────┬─────┘
             /api/* ──────┤────── /*
            (sem /api)    │
              ┌───────────▼──┐   ┌──────────────┐
              │  API (Go)    │   │  Web (Svelte)│
              │   :8080      │   │    :3000     │
              └──────┬───────┘   └──────────────┘
                     │
              ┌──────▼───────┐
              │  Postgres    │  (interno; sem porta pública)
              └──────────────┘
```

Caddy serve **frontend e API na mesma origem**: `/api/*` vai para a API (o
prefixo `/api` é removido antes de repassar) e todo o resto vai para o
frontend. Isso importa: o cookie de sessão é `SameSite=Lax`, e uma origem só é
o que faz o navegador enviá-lo de volta nas chamadas do front para a API — sem
mudar código e sem CORS. (Se front e API ficassem em domínios diferentes, o
login quebraria silenciosamente.)

Emails saem por um SMTP externo (ex.: Brevo) — não há servidor de email na stack.

## Opções de hospedagem (grátis / barato)

Como tudo roda em `docker compose`, o encaixe mais direto é **um VPS pequeno**.
Precisa de ~2 GB de RAM (o build do Go + npm aperta em 1 GB; use 2 GB ou ligue
swap).

| Opção | Custo | Observações |
|---|---|---|
| **Oracle Cloud — Always Free** | **R$0** (para sempre) | VM ARM Ampere (até 4 vCPU / 24 GB). Genuinamente grátis; pede cartão no cadastro e a disponibilidade de ARM varia por região. As imagens do projeto são multi-arch (rodam em ARM64). |
| **Hetzner Cloud** | ~€3,8–4,5/mês (~R$25/mês) | CAX11 (ARM, 2 vCPU/4 GB) ou CX22 (x86). Ótimo custo, painel simples. Pede verificação de identidade. |
| **DigitalOcean / Vultr / Linode** | ~US$4–6/mês | Droplets fáceis, costumam dar crédito inicial para contas novas. |

Alternativas gerenciadas (menos trabalho de servidor, mas com as ressalvas de
cookie/custo já discutidas): Render, Railway, Fly.io — todas exigiriam ou um
domínio próprio com subdomínios (`app.` + `api.`) ou uma mudança no cookie para
`SameSite=None`. O caminho de VPS abaixo evita as duas coisas.

## Passo a passo — VPS + Caddy (recomendado)

### 1. Provisione o host

Crie a VM (Ubuntu 22.04+ serve bem), anote o **IP público** e instale o Docker:

```bash
curl -fsSL https://get.docker.com | sh
```

Abra **só** as portas 22 (SSH), 80 e 443 no firewall do provedor. **Não** exponha
5432/8080/3000 — esses serviços só precisam falar entre si.

### 2. Um domínio grátis (DuckDNS)

O Caddy precisa de um nome de domínio (não dá para emitir certificado TLS
confiável para um IP puro). Se você não tem um domínio, use o
[DuckDNS](https://www.duckdns.org) (grátis):

1. Entre com GitHub/Google e crie um subdomínio, ex.: `agendago`.
2. No campo **current ip** dele, coloque o IP público do seu VPS e salve.
3. Pronto: `agendago.duckdns.org` aponta para o seu host.

### 3. Clone e configure

```bash
git clone https://github.com/<voce>/agendaGo.git && cd agendaGo
cp .env.prod.example .env
nano .env      # preencha DOMINIO, senhas, SMTP (veja os comentários do arquivo)
```

No mínimo ajuste: `DOMINIO`, `POSTGRES_PASSWORD`, `ADMIN_SENHA` e o bloco SMTP.

### 4. Suba

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

Isso builda as imagens de produção, roda as migrations (job Flyway), sobe API +
web e o Caddy — que pega o certificado HTTPS sozinho no primeiro acesso.

### 5. Verifique

```bash
curl -I https://SEU_DOMINIO/api/health     # 200, e cadeado válido
```

Abra `https://SEU_DOMINIO` no navegador, crie um prestador, ative a agenda,
agende como convidado e confirme que o email chega (e que o link dele aponta
para o seu domínio, não para localhost).

## Checklist de go-live (segurança)

Antes de anunciar o link, confirme:

- [ ] **`APP_ENV=production`** — já fixado no compose; liga o `Secure` do cookie.
- [ ] **Rate limit > 0** — `RATE_LIMIT_*` não podem ser `0` em produção (brute-force).
- [ ] **`ADMIN_SENHA` forte** — é a chave do painel de moderação. `openssl rand -base64 24`.
- [ ] **`POSTGRES_PASSWORD` forte** — idem.
- [ ] **SMTP com remetente verificado** — nunca um `@gmail/@outlook/@yahoo` (DMARC → spam). Se você já usou uma chave SMTP em outro lugar/commit, **rotacione**.
- [ ] **Postgres sem porta pública** — o compose de produção já não expõe 5432; confirme que o firewall também não.
- [ ] **Swagger fora do ar** — o Caddyfile já responde 404 em `/api/swagger*`.

## Variáveis de ambiente

O `docker-compose.prod.yml` já injeta na API os valores certos a partir do
`.env`. Referência do que cada uma faz:

| Variável | Obrigatória | Padrão | Para quê |
|---|---|---|---|
| `DOMINIO` | sim | — | hostname público; usado pelo Caddy (TLS), pelo build do front (`PUBLIC_API_URL`) e como `FRONTEND_ORIGIN` |
| `POSTGRES_DB/USER/PASSWORD` | sim | — | banco (a API recebe como `DB_*`) |
| `APP_ENV` | sim | `production` (fixo no compose) | liga o `Secure` do cookie de sessão |
| `FRONTEND_ORIGIN` | sim | `https://${DOMINIO}` (fixo) | origem no CORS **e** base dos links dos emails |
| `ADMIN_EMAIL/SENHA` | recomendado | — | semeiam o admin no boot (vazias = sem admin) |
| `RATE_LIMIT_LOGIN_POR_MINUTO` | não | `10` | teto de logins por IP/min (0 desliga — **não use 0**) |
| `RATE_LIMIT_CONVIDADO_POR_MINUTO` | não | `10` | teto de agendamentos de convidado por IP/min |
| `SMTP_HOST` | não | — | servidor SMTP. **Vazio desliga o envio** (emails só logados) |
| `SMTP_PORT` | não | `587` | porta SMTP |
| `SMTP_USER/PASSWORD` | não | — | credenciais SMTP |
| `SMTP_STARTTLS` | não | `true` | exige STARTTLS |
| `EMAIL_REMETENTE/_NOME` | não | — | remetente (precisa ser verificado no provedor) |
| `EMAIL_REPLY_TO` | não | — | endereço de resposta (seu email pessoal) |

## Atualizar (redeploy)

```bash
git pull
docker compose -f docker-compose.prod.yml up -d --build
```

O Flyway aplica só as migrations novas; a API sobe depois. O desligamento é
gracioso (a API dá até 10s para as requisições em andamento terminarem antes de
sair), então o redeploy não derruba requests no meio.

> Se você **mudar o `DOMINIO`**, rebuilde a imagem do front — `PUBLIC_API_URL` é
> resolvida em tempo de build. O `--build` acima já cobre isso.

## Manutenção

- **Backup do banco**: `docker compose -f docker-compose.prod.yml exec postgres pg_dump -U agendago agendago > backup.sql`. Agende via cron.
- **Logs**: `docker compose -f docker-compose.prod.yml logs -f api` (ou `caddy`, `web`).
- **Certificado TLS**: o Caddy renova sozinho; os certificados persistem no volume `caddy_data`.

## Build manual das imagens (sem compose)

Para quem for orquestrar de outro jeito (Kubernetes, PaaS), as imagens de
produção são autônomas:

```bash
# API — binário Go estático, usuário sem privilégios, healthcheck em /health
docker build -f backend/Dockerfile.prod -t agendago-api backend/

# Web — PUBLIC_API_URL resolvida em tempo de BUILD
docker build -f frontend/Dockerfile.prod \
  --build-arg PUBLIC_API_URL=https://api.seudominio.com \
  -t agendago-web frontend/
```

As migrations de `backend/migrations/` devem rodar **antes** de subir a API:

```bash
docker run --rm -v ./backend/migrations:/flyway/sql \
  -e FLYWAY_URL=jdbc:postgresql://SEU_HOST:5432/agendago \
  -e FLYWAY_USER=... -e FLYWAY_PASSWORD=... \
  flyway/flyway:10-alpine migrate
```

Neste modo você mesmo precisa terminar o TLS (proxy reverso na frente), garantir
a mesma origem para front e API (ou lidar com o cookie cross-site), e não expor
o Postgres nem o Swagger publicamente.
