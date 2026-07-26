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

## O host não recebe o código-fonte

As imagens de produção (API, web e migrations) são buildadas pelo **CI** e
publicadas no **GHCR** (`ghcr.io/caetasousa/agendago-*`). O
`docker-compose.prod.yml` só as consome — ele não tem nenhuma diretiva `build`.
Na VPS ficam **três arquivos**, e nada além disso:

```
~/agendago/
├── docker-compose.prod.yml
├── Caddyfile
└── .env
```

Isso não é só arrumação: a VPS não precisa de Go, Node nem do repositório, o
redeploy vira um `pull` de segundos em vez de uma compilação de minutos, e uma
máquina de 1 vCPU dá conta (buildar SvelteKit + Go num core único, com o site
no ar, é o caminho mais rápido para um deploy que derruba o serviço).

## Opções de hospedagem (grátis / barato)

Como tudo roda em `docker compose`, o encaixe mais direto é **um VPS pequeno**.
Como o host só puxa imagens prontas, ~1 GB de RAM basta para rodar (Postgres +
API + Node + Caddy); 2 GB dá folga confortável.

> **Arquitetura:** o CI publica imagens **`linux/amd64`**. Hosts ARM (Oracle
> Ampere, Hetzner CAX) exigiriam adicionar `linux/arm64` ao `platforms` do job
> `publicar-imagens` em `.github/workflows/ci.yml`.

| Opção | Custo | Observações |
|---|---|---|
| **Hostinger — KVM 1** | ~R$30/mês (plano de 24 meses; ~R$60 na renovação) | 1 vCPU / 4 GB / 50 GB NVMe, x86, IP dedicado e root completo. Sobra folga: o host não builda nada. |
| **Oracle Cloud — Always Free** | **R$0** (para sempre) | VM ARM Ampere (até 4 vCPU / 24 GB). Genuinamente grátis; pede cartão no cadastro e a disponibilidade de ARM varia por região. Exige publicar imagens `arm64` (veja o aviso acima). |
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

### 3. Baixe os três arquivos e configure

Sem clone: só o compose, o Caddyfile e o `.env`.

```bash
mkdir -p ~/agendago && cd ~/agendago
BASE=https://raw.githubusercontent.com/caetasousa/agendaGo/main
curl -O $BASE/docker-compose.prod.yml
curl -O $BASE/Caddyfile
curl -o .env $BASE/.env.prod.example

nano .env      # preencha DOMINIO, senhas, SMTP (veja os comentários do arquivo)
```

No mínimo ajuste: `DOMINIO`, `POSTGRES_PASSWORD`, `ADMIN_SENHA` e o bloco SMTP.

(Se o repositório for privado, o `curl` não funciona — use `scp` da sua máquina:
`scp docker-compose.prod.yml Caddyfile usuario@IP:~/agendago/`.)

### 4. Autentique no GHCR (se as imagens forem privadas)

Pacotes publicados no GHCR nascem **privados**, mesmo em repositório público.
Escolha um dos caminhos:

- **Tornar públicas** (mais simples): GitHub → seu perfil → *Packages* → cada
  pacote `agendago-*` → *Package settings* → *Change visibility* → Public. A
  VPS então dá `pull` sem login.
- **Manter privadas**: crie um *classic PAT* só com o escopo `read:packages` e,
  na VPS: `echo SEU_PAT | docker login ghcr.io -u caetasousa --password-stdin`.

### 5. Suba

```bash
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

Isso baixa as imagens publicadas pelo CI, roda as migrations (job Flyway), sobe
API + web e o Caddy — que pega o certificado HTTPS sozinho no primeiro acesso.

### 6. Verifique

```bash
# saúde da API (GET, não HEAD: a rota é registrada como GET e devolveria 405)
curl -s https://SEU_DOMINIO/api/health              # {"status":"ok"}

# HTTP tem que redirecionar para HTTPS
curl -sI http://SEU_DOMINIO/ | head -1              # 308 Permanent Redirect

# HSTS e demais cabeçalhos de segurança
curl -sI https://SEU_DOMINIO/ | grep -i strict-transport

# a documentação Swagger não pode estar exposta
curl -so /dev/null -w '%{http_code}\n' https://SEU_DOMINIO/api/swagger/index.html   # 404
```

Abra `https://SEU_DOMINIO` no navegador, crie um prestador, ative a agenda,
agende como convidado e confirme que o email chega (e que o link dele aponta
para o seu domínio, não para localhost). No DevTools, confirme que o cookie
`agendago_session` veio com **Secure**, **HttpOnly** e **SameSite=Lax**, e que
as chamadas XHR vão para `https://SEU_DOMINIO/api/...` — nunca para
`localhost:8080`.

### Ensaiar a VPS na sua máquina

O jeito fiel de ensaiar é reproduzir o host: uma pasta **fora do repositório**
com os mesmos três arquivos. Assim o comando é idêntico ao que você vai rodar
no servidor, sem flag nenhuma, e o `.env` de desenvolvimento fica intocado.

```bash
mkdir -p ~/agendago-vps && cd ~/agendago-vps
cp ~/agendaGo/docker-compose.prod.yml ~/agendaGo/Caddyfile .
cp ~/agendaGo/.env.prod.example .env
# no .env: DOMINIO=localhost (o Caddy usa a CA interna dele, sem Let's Encrypt
# e sem DNS) e senhas descartáveis.

docker compose -f docker-compose.prod.yml up -d
```

Enquanto o CI ainda não publicou as imagens, builde-as antes com as tags que o
compose procura — é o único passo que **não** existe na VPS de verdade (lá o
`up` puxa do GHCR sozinho):

```bash
cd ~/agendaGo
docker build -t ghcr.io/caetasousa/agendago-api:latest        -f backend/Dockerfile.prod       backend/
docker build -t ghcr.io/caetasousa/agendago-web:latest        -f frontend/Dockerfile.prod      frontend/
docker build -t ghcr.io/caetasousa/agendago-migrations:latest -f backend/Dockerfile.migrations backend/
```

Depois é só `https://localhost` no navegador (aceitando o aviso do certificado,
que é da CA interna) e `curl -k` nos comandos de verificação. Mudou código?
Rebuilde a imagem e rode `up -d` de novo na pasta do ensaio. Mudou o compose ou
o Caddyfile? Copie-os outra vez — a pasta é uma cópia, não um link.

## Checklist de go-live (segurança)

Antes de anunciar o link, confirme:

- [ ] **`APP_ENV=production`** — já fixado no compose; liga o `Secure` do cookie.
- [ ] **Rate limit > 0** — `RATE_LIMIT_*` não podem ser `0` em produção (brute-force).
- [ ] **`ADMIN_SENHA` forte** — é a chave do painel de moderação. `openssl rand -base64 24`.
- [ ] **`POSTGRES_PASSWORD` forte** — idem.
- [ ] **SMTP com remetente verificado** — nunca um `@gmail/@outlook/@yahoo` (DMARC → spam). Se você já usou uma chave SMTP em outro lugar/commit, **rotacione**.
- [ ] **Postgres sem porta pública** — o compose de produção já não expõe 5432; confirme que o firewall também não.
- [ ] **Swagger fora do ar** — duas camadas: com `APP_ENV=production` a API nem
      monta a rota (`config.NovoRouter`), e o Caddyfile responde 404 em
      `/api/swagger*`. Confirme com o `curl` do passo 6.

## Variáveis de ambiente

O `docker-compose.prod.yml` já injeta na API os valores certos a partir do
`.env`. Referência do que cada uma faz:

| Variável | Obrigatória | Padrão | Para quê |
|---|---|---|---|
| `DOMINIO` | sim | — | hostname público; usado pelo Caddy (TLS), pelo build do front (`PUBLIC_API_URL`) e como `FRONTEND_ORIGIN` |
| `POSTGRES_DB/USER/PASSWORD` | sim | — | banco (a API recebe como `DB_*`) |
| `APP_ENV` | sim | `production` (fixo no compose) | liga o `Secure` do cookie de sessão **e** o log em JSON |
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
| `GOOGLE_CLIENT_ID/SECRET` | não | — | login social; **vazio desliga** o recurso (rotas e botão somem) |
| `GOOGLE_REDIRECT_URL` | se usar Google | — | callback **com o prefixo `/api`** (veja abaixo) |

### Login social com Google

Duas armadilhas, ambas silenciosas até o usuário tentar entrar:

1. **A URL de callback leva `/api`.** O callback é rota da API
   (`/auth/google/callback`), e atrás do Caddy a API vive sob `/api`. O valor
   correto é `https://SEU_DOMINIO/api/auth/google/callback`. Sem o prefixo, o
   Google devolve o usuário no frontend, que não tem essa rota: o consentimento
   funciona e o login morre num 404.
2. **A URL precisa estar cadastrada no Google Cloud Console**, em *Authorized
   redirect URIs* do cliente OAuth, **idêntica** (mesmo esquema, sem barra
   final). Divergiu, o Google recusa com `redirect_uri_mismatch` antes mesmo da
   tela de consentimento. O mesmo cliente aceita várias URIs, então dev, ensaio
   local e produção convivem:

   ```
   http://localhost:8080/auth/google/callback      (dev, API direta)
   https://localhost/api/auth/google/callback      (ensaio da VPS)
   https://SEU_DOMINIO/api/auth/google/callback    (produção)
   ```

Para conferir sem abrir o navegador, veja o `redirect_uri` que a API manda ao
Google:

```bash
curl -sD - -o /dev/null https://SEU_DOMINIO/api/auth/client/google/start | grep -i location
```

### Entrega de email

O remetente **não pode** ser `@gmail`/`@outlook`/`@yahoo`: quem envia é a Brevo,
o DMARC desses provedores não a autoriza, e a mensagem cai em spam ou é
rejeitada. É preciso um endereço de **domínio próprio** autenticado na Brevo
(SPF + DKIM). Um subdomínio DuckDNS não resolve — você não controla o DNS de
`duckdns.org`, então não consegue publicar os registros. Se for usar email de
verdade (e é: confirmação de cadastro, recuperação de senha e cancelamento
dependem dele), inclua um domínio no orçamento.

Para ver os emails localmente sem enviar nada, aponte o SMTP para um Mailpit —
lembrando de **zerar usuário e senha**, porque o Mailpit não faz AUTH e a API
falha com `server does not support SMTP AUTH`:

```bash
docker run -d --name mailpit --network agendago-vps_default -p 8025:8025 axllent/mailpit
SMTP_HOST=mailpit SMTP_PORT=1025 SMTP_STARTTLS=false SMTP_USER= SMTP_PASSWORD= \
  docker compose -f docker-compose.prod.yml up -d api
# caixa de entrada em http://localhost:8025
```

Para validar as credenciais reais da Brevo sem disparar mensagem, basta o
handshake de autenticação (`235` = ok):

```bash
U=$(printf '%s' "$SMTP_USER" | base64 -w0); P=$(printf '%s' "$SMTP_PASSWORD" | base64 -w0)
printf 'EHLO teste\r\nAUTH LOGIN\r\n%s\r\n%s\r\nQUIT\r\n' "$U" "$P" \
  | openssl s_client -quiet -starttls smtp -connect smtp-relay.brevo.com:587 2>/dev/null | grep '^235'
```

## Atualizar (redeploy)

Todo push na `main` que passa nos testes publica imagens `:latest` novas. Na
VPS, o redeploy inteiro é:

```bash
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

O Flyway aplica só as migrations novas; a API sobe depois. O desligamento é
gracioso (a API dá até 10s para as requisições em andamento terminarem antes de
sair), então o redeploy não derruba requests no meio.

**Rollback** — o CI também tagueia cada imagem com o SHA do commit. Para voltar,
fixe a versão no `.env` e suba de novo:

```bash
echo 'IMAGE_TAG=<sha-do-commit-bom>' >> .env
docker compose -f docker-compose.prod.yml up -d
```

(Migrations não voltam sozinhas: se o commit ruim adicionou uma migration, o
rollback do schema é manual.)

> Trocar o `DOMINIO` **não** exige rebuild do front: `PUBLIC_API_URL` é assada
> como `/api` (relativa), então a imagem serve qualquer domínio. Basta ajustar o
> `.env` e subir de novo — o Caddy pega o certificado do novo nome.

## Manutenção

- **Backup do banco**: `docker compose -f docker-compose.prod.yml exec postgres pg_dump -U agendago agendago > backup.sql`. Agende via cron.
- **Certificado TLS**: o Caddy renova sozinho; os certificados persistem no volume `caddy_data`.

### Logs

Em produção (`APP_ENV=production`) a API emite **JSON estruturado** em stdout —
uma linha por evento, parseável por qualquer agregador (Loki, CloudWatch,
Datadog). O compose já configura rotação (`json-file`, 3 arquivos de 10 MB por
serviço), então o disco do host não enche.

```bash
docker compose -f docker-compose.prod.yml logs -f api      # segue o log da API
docker compose -f docker-compose.prod.yml logs api | jq    # formata o JSON

# só os erros (500 já trazem o erro real, com request_id):
docker compose -f docker-compose.prod.yml logs api | jq 'select(.level=="ERROR")'

# eventos de segurança (login falho, conta banida):
docker compose -f docker-compose.prod.yml logs api | jq 'select(.level=="WARN")'
```

Cada requisição gera uma linha de acesso com `request_id`, `rota` (o padrão, não
o caminho — tokens não aparecem no log), `status`, `duracao` e `ip` (o do
cliente real, resolvido do `X-Real-IP` que o Caddy define). O mesmo `request_id`
liga a linha de acesso ao log de erro/segurança da mesma requisição.

## Build manual das imagens (sem CI)

O CI faz isso a cada push na `main`, mas as três imagens são autônomas e podem
ser buildadas na sua máquina — útil para testar antes de publicar, ou para
orquestrar de outro jeito (Kubernetes, PaaS):

```bash
# API — binário Go estático, usuário sem privilégios, healthcheck em /health
docker build -f backend/Dockerfile.prod -t agendago-api backend/

# Migrations — Flyway com os .sql embutidos
docker build -f backend/Dockerfile.migrations -t agendago-migrations backend/

# Web — servidor Node do adapter-node. O padrão de PUBLIC_API_URL (/api) serve
# para qualquer domínio atrás do Caddy; só passe --build-arg se front e API
# ficarem em origens diferentes.
docker build -f frontend/Dockerfile.prod -t agendago-web frontend/
```

Para empurrar manualmente para o GHCR, taguear com o destino e dar push:

```bash
docker tag agendago-api ghcr.io/caetasousa/agendago-api:latest
docker push ghcr.io/caetasousa/agendago-api:latest
```

As migrations devem rodar **antes** de subir a API — a imagem já carrega os
`.sql`, então não precisa de volume:

```bash
docker run --rm \
  -e FLYWAY_URL=jdbc:postgresql://SEU_HOST:5432/agendago \
  -e FLYWAY_USER=... -e FLYWAY_PASSWORD=... \
  agendago-migrations migrate
```

Fora do compose de produção você mesmo precisa terminar o TLS (proxy reverso na
frente), garantir a mesma origem para front e API (ou lidar com o cookie
cross-site) e não expor o Postgres. O Swagger não exige cuidado extra: basta
`APP_ENV=production` na API, que a rota deixa de existir.
