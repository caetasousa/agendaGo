# Produção

Como o agendaGo vai para o ar, na ordem em que as coisas precisam acontecer.
Este documento é o roteiro que foi realmente percorrido — inclusive os erros que
apareceram no caminho e o que os causou. A referência seca (tabela de variáveis,
comandos de operação) está no fim.

O `docker-compose.yml` da raiz é **só para desenvolvimento** (hot reload, portas
expostas, Mailpit). Produção usa o **`docker-compose.prod.yml`**.

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

O Caddy serve **frontend e API na mesma origem**: `/api/*` vai para a API (o
prefixo é removido antes de repassar) e todo o resto vai para o frontend.

Isso não é estética. O cookie de sessão é `SameSite=Lax`, e é a origem única que
faz o navegador enviá-lo de volta nas chamadas do front para a API — sem mudar
código e sem CORS. Se front e API ficassem em domínios diferentes, o login
quebraria silenciosamente. Essa decisão também é o que permite o frontend usar
`/api` relativo e a imagem publicada não ficar amarrada a um domínio.

Emails saem por um SMTP externo (ex.: Brevo) — não há servidor de email na stack.

## O host não recebe o código-fonte

As imagens de produção (API, web e migrations) são buildadas pelo **CI** e
publicadas no **GHCR**. O `docker-compose.prod.yml` só as consome: ele não tem
nenhuma diretiva `build`. Na VPS ficam **três arquivos**, e nada além disso:

```
~/agendago/
├── docker-compose.prod.yml
├── Caddyfile
└── .env
```

O ganho não é arrumação. Buildar no servidor exigiria repositório, toolchain Go
e Node, e RAM/CPU para compilar — num plano de 1 vCPU, cada deploy competiria
com o site no ar por vários minutos. Puxando imagem pronta, o deploy é um
download de segundos, e o que roda em produção é exatamente o artefato que
passou no CI, não uma recompilação que pode divergir.

---

# Roteiro do primeiro deploy

## 1. Um nome de domínio

**Por que não dá para usar o IP puro:** nenhuma autoridade pública emite
certificado para endereço IP. O Caddy, ao receber um IP como endereço do site,
cai na CA interna dele — e o handshake TLS falha na prática, porque clientes não
enviam SNI quando o destino é um IP, e sem SNI o servidor não sabe qual
certificado apresentar. O resultado é um redirect para HTTPS que morre em
`tlsv1 alert internal error`.

**E servir em HTTP puro também não resolve:** com `APP_ENV=production` o cookie
de sessão vai com o atributo `Secure`, e o navegador se recusa a guardar cookie
`Secure` recebido por HTTP. O site abriria e **ninguém conseguiria logar**, sem
mensagem de erro.

Sem domínio próprio, a saída gratuita é o [DuckDNS](https://www.duckdns.org):
entre com GitHub, crie um subdomínio e coloque o IP da VPS no campo *current
ip*. Confirme antes de seguir:

```bash
getent hosts SEU_SUBDOMINIO.duckdns.org
```

Use `getent` em vez de `dig`: o `dig` não vem no Ubuntu mínimo. A resposta tem
de ser o IP público da VPS, que você confere com `curl -s https://api.ipify.org`.

**Não pule esta verificação.** Se o Caddy tentar emitir com o DNS errado, o
Let's Encrypt bloqueia novas tentativas por uma hora depois de cinco falhas.

## 2. Preparar o servidor

Conecte como root e instale o Docker:

```bash
curl -fsSL https://get.docker.com | sh
```

Crie um usuário sem privilégios para a aplicação e o deploy:

```bash
adduser --disabled-password --gecos '' deploy
usermod -aG docker deploy
```

`--disabled-password` significa que ninguém entra nesse usuário por senha, só
por chave SSH. `-aG docker` (com o `a`, de *append* — sem ele os grupos atuais
seriam substituídos) dá acesso ao daemon do Docker sem `sudo`.

> **Armadilha:** o `deploy` não tem senha e não está no grupo `sudo`. Qualquer
> `sudo` executado por ele falha com "3 incorrect password attempts". É
> intencional: comandos de sistema são do root, comandos de Docker e da
> aplicação são do `deploy`.

Firewall com o mínimo aberto:

```bash
ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp && ufw --force enable
```

A **porta 80 não é opcional**: é por onde chega o desafio de validação do Let's
Encrypt. Fechada, não sai certificado — mesmo que o site seja todo HTTPS.
Postgres (5432) e API (8080) não são expostos; os containers conversam por uma
rede interna do Docker.

Confira:

```bash
docker --version && ufw status && id deploy
```

O `id deploy` precisa mostrar `docker` entre os grupos.

## 3. Liberar as imagens no GHCR

Pacotes publicados no GHCR nascem **privados**, mesmo em repositório público.
Sem este passo o `docker compose up` falha com `denied`.

GitHub → seu perfil → *Packages* → cada `agendago-*` → *Package settings* →
*Danger Zone* → *Change visibility* → **Public**.

**Isso é seguro?** Com repositório público, sim: a imagem é o código-fonte
compilado, e o fonte já está aberto. Nenhum segredo entra nas imagens — os
Dockerfiles buildam a partir de `./backend` e `./frontend`, onde não existe
arquivo `.env`. A alternativa (pacotes privados) exige `docker login` na VPS com
um PAT, que fica gravado em texto quase claro em `~/.docker/config.json` e
expira sem avisar. Se um dia o repositório virar privado, torne os pacotes
privados junto.

## 4. Instalar a stack

Como `deploy`:

```bash
su - deploy
mkdir -p ~/agendago && cd ~/agendago

BASE=https://raw.githubusercontent.com/caetasousa/agendaGo/main
curl -O $BASE/docker-compose.prod.yml
curl -O $BASE/Caddyfile
curl -o .env $BASE/.env.prod.example
```

O `-` em `su - deploy` carrega o ambiente de login do zero, o que inclui a
participação nova no grupo `docker`. `-O` salva com o nome original; `-o .env`
renomeia o exemplo para o arquivo real.

Gere as senhas e preencha:

```bash
openssl rand -base64 24    # POSTGRES_PASSWORD
openssl rand -base64 24    # ADMIN_SENHA
nano .env
chmod 600 .env
```

No mínimo: `DOMINIO`, `POSTGRES_PASSWORD`, `ADMIN_EMAIL`, `ADMIN_SENHA` e
`GOOGLE_REDIRECT_URL`. O `chmod 600` restringe a leitura ao dono — o arquivo tem
a senha do banco, a do admin e as credenciais de SMTP e OAuth.

Suba **de dentro da pasta**:

```bash
cd ~/agendago
docker compose -f docker-compose.prod.yml up -d
```

> **Armadilha:** o Compose lê o `.env` do **diretório atual**, não do diretório
> onde está o arquivo de compose. Rodando de outro lugar, todas as variáveis
> chegam vazias: o Caddy pede certificado para um domínio em branco e o Postgres
> sobe sem senha — e nada disso diz "faltou o .env".

Para ver a configuração já resolvida, sem subir nada:

```bash
docker compose -f docker-compose.prod.yml config | grep -E "DOMINIO|image:"
```

## 5. Conferir

```bash
docker compose -f docker-compose.prod.yml ps -a
```

Esperado: `postgres` e `api` como *healthy*, `web` e `caddy` como *Up*, e
`flyway` como **Exited (0)**. O `-a` é necessário justamente para enxergar o
`flyway`: ele é um job pontual que aplica as migrations e sai — **sair é o
comportamento correto**, não é falha.

```bash
docker compose -f docker-compose.prod.yml logs caddy | grep -i certificate
```

Você quer `certificate obtained successfully` com `"issuer"` citando
**letsencrypt**. Se aparecer `"issuer":"local"`, a emissão pública falhou e o
Caddy caiu na CA interna — quase sempre DNS errado ou porta 80 bloqueada (na
Hostinger, confira também o firewall do painel, que é separado do `ufw`).

```bash
curl -s https://SEU_DOMINIO/api/health         # {"status":"ok"}
curl -sI http://SEU_DOMINIO/ | head -1         # 308 Permanent Redirect
curl -sI https://SEU_DOMINIO/ | grep -i strict-transport
curl -so /dev/null -w '%{http_code}\n' https://SEU_DOMINIO/api/swagger/index.html   # 404
```

Dois detalhes que valem atenção:

- **Use GET, não HEAD, no `/health`.** `curl -I` manda HEAD, e a rota é
  registrada como `GET` — a resposta é `405`, que parece defeito e não é.
- **Sem `-k`.** Durante o ensaio local o `-k` é necessário (certificado
  interno). Aqui a ausência dele é o próprio teste: se responder, o certificado
  é publicamente confiável e nenhum visitante verá aviso.

No navegador, confirme no DevTools que o cookie `agendago_session` veio com
**Secure**, **HttpOnly** e **SameSite=Lax**, e que as chamadas XHR vão para
`https://SEU_DOMINIO/api/...` — nunca para `localhost:8080`.

## 6. Login social com Google

Cadastre a URL de callback em *Authorized redirect URIs*, no cliente OAuth do
[Google Cloud Console](https://console.cloud.google.com/apis/credentials):

```
https://SEU_DOMINIO/api/auth/google/callback
```

> **Armadilha:** o `/api` é obrigatório. O callback é rota da **API**, e atrás do
> Caddy a API vive sob esse prefixo. Sem ele, o Google devolve o usuário no
> frontend, que não tem essa rota: o consentimento funciona e o login morre num
> 404 — depois de a conta já ter sido vinculada do lado do Google.

O Google compara a URL caractere a caractere; divergiu, ele recusa com
`redirect_uri_mismatch`. O mesmo cliente OAuth aceita várias URIs, então dev,
ensaio local e produção convivem:

```
http://localhost:8080/auth/google/callback      (dev, API direta)
https://localhost/api/auth/google/callback      (ensaio local)
https://SEU_DOMINIO/api/auth/google/callback    (produção)
```

Para conferir o que a API manda ao Google, sem abrir o navegador:

```bash
curl -sD - -o /dev/null https://SEU_DOMINIO/api/auth/client/google/start | grep -i location
```

## 7. Email

O remetente **não pode** ser `@gmail`/`@outlook`/`@yahoo`: quem envia é a Brevo,
o DMARC desses provedores não a autoriza, e a mensagem cai em spam ou é
rejeitada. É preciso um endereço de **domínio próprio** autenticado na Brevo
(SPF + DKIM).

**Um subdomínio DuckDNS não resolve isso** — você não controla o DNS de
`duckdns.org` e não consegue publicar os registros. Enquanto não houver domínio
próprio, confirmação de cadastro, recuperação de senha e cancelamento por link
não chegam de forma confiável. Se email é requisito, inclua um domínio no
orçamento.

Credenciais SMTP ficam em Brevo → *SMTP & API* → *SMTP* (não é a senha da conta).
Para validar sem disparar mensagem, basta o handshake de autenticação:

```bash
U=$(printf '%s' "$SMTP_USER" | base64 -w0); P=$(printf '%s' "$SMTP_PASSWORD" | base64 -w0)
printf 'EHLO teste\r\nAUTH LOGIN\r\n%s\r\n%s\r\nQUIT\r\n' "$U" "$P" \
  | openssl s_client -quiet -starttls smtp -connect smtp-relay.brevo.com:587 2>/dev/null | grep '^235'
```

`235` significa credenciais aceitas.

---

# Deploy automático (CI → VPS)

O job `implantar` do `.github/workflows/ci.yml` fecha o ciclo: depois que as
imagens são publicadas, ele entra na VPS por SSH, sincroniza
`docker-compose.prod.yml` e `Caddyfile`, manda puxar as imagens **fixadas no SHA
do commit** e sobe a stack. Por fim chama `/api/health` e falha se a aplicação
não responder.

Três coisas que ele **não** faz, de propósito: não envia código-fonte (as
imagens vêm do GHCR), não toca no `.env` do servidor (é lá que vivem os
segredos) e não roda nada se os segredos não existirem — o job passa com um
aviso, para o repositório funcionar antes de a VPS existir.

**1. Uma chave SSH exclusiva do CI**, gerada na sua máquina:

```bash
ssh-keygen -t ed25519 -f ~/.ssh/agendago_deploy -N '' -C 'github-actions'
```

**2. Autorizar a chave pública**, na VPS como root:

```bash
mkdir -p /home/deploy/.ssh
echo 'CONTEUDO_DE_agendago_deploy.pub' >> /home/deploy/.ssh/authorized_keys
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh && chmod 600 /home/deploy/.ssh/authorized_keys
```

As permissões importam: o SSH **recusa a chave silenciosamente** se a pasta ou o
arquivo estiverem legíveis por outros usuários.

Teste antes de seguir:

```bash
ssh -i ~/.ssh/agendago_deploy deploy@SEU_IP 'docker ps --format "{{.Names}}"'
```

Listar os containers prova três coisas de uma vez: a chave funciona, o usuário
entra sem senha e tem acesso ao Docker.

**3. Cadastrar no GitHub** (*Settings → Secrets and variables → Actions*):

| Nome | Aba | Conteúdo |
|---|---|---|
| `VPS_HOST` | Secrets | IP ou hostname da VPS |
| `VPS_USER` | Secrets | `deploy` |
| `VPS_SSH_KEY` | Secrets | conteúdo de `~/.ssh/agendago_deploy` (chave **privada**, com as linhas `BEGIN`/`END`) |
| `VPS_PORT` | Secrets | opcional, padrão `22` |
| `VPS_KNOWN_HOSTS` | Secrets | saída de `ssh-keyscan SEU_IP` |
| `DOMINIO` | **Variables** | seu domínio, usado na verificação pós-deploy |

`VPS_KNOWN_HOSTS` é opcional mas recomendado: sem ele o CI aceita a identidade
que o servidor apresentar na hora, o que abre janela para man-in-the-middle no
primeiro contato.

**4. Testar sem commit:** *Actions → CI → Run workflow*, habilitado pelo
`workflow_dispatch`.

---

# Ensaiar na sua máquina antes

A stack inteira roda local com `DOMINIO=localhost`, o que permite testar o
artefato de produção de verdade — foi assim que quatro defeitos que só se
manifestariam em produção apareceram antes do deploy.

Reproduza o host: uma pasta **fora do repositório** com os mesmos três arquivos.
Assim o comando é idêntico ao do servidor e o `.env` de desenvolvimento fica
intocado.

```bash
mkdir -p ~/agendago-vps && cd ~/agendago-vps
cp ~/agendaGo/docker-compose.prod.yml ~/agendaGo/Caddyfile .
cp ~/agendaGo/.env.prod.example .env
# no .env: DOMINIO=localhost e senhas descartáveis
docker compose -f docker-compose.prod.yml up -d
```

Enquanto o CI não tiver publicado as imagens, builde-as antes com as tags que o
compose procura — é o único passo que **não** existe na VPS de verdade:

```bash
cd ~/agendaGo
docker build -t ghcr.io/caetasousa/agendago-api:latest        -f backend/Dockerfile.prod       backend/
docker build -t ghcr.io/caetasousa/agendago-web:latest        -f frontend/Dockerfile.prod      frontend/
docker build -t ghcr.io/caetasousa/agendago-migrations:latest -f backend/Dockerfile.migrations backend/
```

Com `DOMINIO=localhost` o Caddy emite um certificado da **CA interna** dele, sem
Let's Encrypt e sem DNS. Por isso os comandos de verificação locais usam `curl -k`.

No navegador o certificado interno gera aviso, e o cabeçalho HSTS faz o Chrome
**remover o botão de prosseguir**. Duas saídas: digitar `thisisunsafe` na tela
de erro, ou instalar a CA do Caddy no sistema:

```bash
docker cp agendago-vps-caddy-1:/data/caddy/pki/authorities/local/root.crt .
# Windows (repositório do usuário, não exige admin):
certutil.exe -user -addstore Root root.crt
```

A confiança quebra se você apagar o volume (`down -v` recria a CA). Nada disso
existe em produção, onde o certificado é público.

---

# Operação

## Atualizar

Com o deploy automático configurado, um push na `main` que passe nos testes já
sobe a versão nova. Manualmente:

```bash
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

O Flyway aplica só as migrations novas; a API sobe depois. O desligamento é
gracioso (até 10s para as requisições em andamento terminarem), então o redeploy
não derruba requests no meio.

## Rollback

O CI tagueia cada imagem com o SHA do commit:

```bash
echo 'IMAGE_TAG=<sha-do-commit-bom>' >> .env
docker compose -f docker-compose.prod.yml up -d
```

Migrations não voltam sozinhas: se o commit ruim adicionou uma, o rollback do
schema é manual.

## Logs

Em produção a API emite **JSON estruturado** em stdout, com rotação já
configurada no compose (3 arquivos de 10 MB por serviço).

```bash
docker compose -f docker-compose.prod.yml logs -f api
docker compose -f docker-compose.prod.yml logs api | jq 'select(.level=="ERROR")'
docker compose -f docker-compose.prod.yml logs api | jq 'select(.level=="WARN")'
```

Cada requisição gera uma linha com `request_id`, `rota` (o padrão, não o caminho
— tokens não vão para o log), `status`, `duracao` e `ip` do cliente real
(resolvido do `X-Real-IP` que o Caddy define). O mesmo `request_id` liga a linha
de acesso ao erro correspondente.

## Backup

```bash
docker compose -f docker-compose.prod.yml exec postgres \
  pg_dump -U agendago agendago > backup-$(date +%F).sql
```

Agende por cron. Backup de VM do provedor restaura a máquina inteira — serve
para desastre, não para "recuperar o banco de ontem" nem para levar os dados
para outro lugar.

## Certificado

O Caddy renova sozinho; os certificados persistem no volume `caddy_data`.

---

# Dimensionamento

Medições reais da stack de produção, com os containers limitados a 2 vCPUs e o
banco vazio:

| Rota | Req/s | p50 | p99 |
|---|---|---|---|
| `/api/health` | 12.235 | 3,5 ms | 15,6 ms |
| `/api/providers` (Postgres) | 4.665 | 8,7 ms | 1,55 s |
| `/` (página do frontend) | 2.749 | 15,5 ms | 233 ms |
| `/auth/provider/login` (Argon2id) | 85 | 226 ms | 544 ms |

Em repouso, os quatro containers somam **99 MiB**: Postgres 41, web 26, API 18,
Caddy 13. Com o SO e o daemon do Docker, ~400 MB.

O gargalo é o **login**: cada verificação Argon2id custa ~24 ms de CPU e 19 MB
de RAM, de propósito — é o que torna quebra de senha por força bruta cara. Sob
20 logins simultâneos, a API sozinha chegou a **365 MiB**. Num host de 1 GB isso
pede swap; com 4 GB não é preocupação. O rate limit por IP existe exatamente
para conter esse pico.

Uma VPS de 1 vCPU e 4 GB atende com folga: o recurso escasso passa a ser CPU, e
mesmo assim sobram ordens de grandeza para o perfil de uso de um app de
agendamento. Como o host não compila nada, o requisito de RAM caiu — era o build
que exigia 2 GB.

---

# Referência: variáveis de ambiente

| Variável | Obrigatória | Padrão | Para quê |
|---|---|---|---|
| `DOMINIO` | sim | — | hostname público; usado pelo Caddy (TLS) e como `FRONTEND_ORIGIN` |
| `IMAGE_REPO` | não | `ghcr.io/caetasousa` | de onde vêm as imagens |
| `IMAGE_TAG` | não | `latest` | versão das imagens (use o SHA para fixar/rollback) |
| `POSTGRES_DB/USER/PASSWORD` | sim | — | banco (a API recebe como `DB_*`) |
| `APP_ENV` | sim | `production` (fixo no compose) | liga o `Secure` do cookie, o log em JSON e **remove a rota do Swagger** |
| `FRONTEND_ORIGIN` | sim | `https://${DOMINIO}` (fixo) | origem no CORS e base dos links dos emails |
| `ADMIN_EMAIL/SENHA` | recomendado | — | semeiam o admin no boot (vazias = sem admin) |
| `RATE_LIMIT_LOGIN_POR_MINUTO` | não | `10` | teto de logins por IP/min (0 desliga — **não use 0**) |
| `RATE_LIMIT_CONVIDADO_POR_MINUTO` | não | `10` | teto de agendamentos de convidado por IP/min |
| `SMTP_HOST` | não | — | servidor SMTP. **Vazio desliga o envio** (emails só logados) |
| `SMTP_PORT` | não | `587` | porta SMTP |
| `SMTP_USER/PASSWORD` | não | — | credenciais SMTP |
| `SMTP_STARTTLS` | não | `true` | exige STARTTLS |
| `EMAIL_REMETENTE/_NOME` | não | — | remetente (precisa ser de domínio autenticado) |
| `EMAIL_REPLY_TO` | não | — | endereço de resposta |
| `GOOGLE_CLIENT_ID/SECRET` | não | — | login social; **vazio desliga** o recurso |
| `GOOGLE_REDIRECT_URL` | se usar Google | — | callback **com o prefixo `/api`** |

# Checklist de go-live

- [ ] **`APP_ENV=production`** — já fixado no compose; liga `Secure` no cookie
- [ ] **Certificado da Let's Encrypt** — `curl` sem `-k` responde
- [ ] **Rate limit > 0** — `RATE_LIMIT_*` não podem ser `0`
- [ ] **`ADMIN_SENHA` e `POSTGRES_PASSWORD` fortes** — `openssl rand -base64 24`
- [ ] **`.env` com `chmod 600`**
- [ ] **SMTP com remetente de domínio autenticado** — se já usou a chave em outro lugar, rotacione
- [ ] **Postgres sem porta pública** — o compose não expõe; confirme o firewall
- [ ] **Swagger fora do ar** — duas camadas: a API não monta a rota em produção e o Caddy responde 404
- [ ] **Backup do banco agendado**

# Armadilhas conhecidas

Compilado do que quebrou de verdade, para consulta rápida:

| Sintoma | Causa |
|---|---|
| `405` no `/api/health` | `curl -I` manda HEAD; a rota é `GET` |
| `flyway` como *Exited* | é o esperado: job pontual que aplica migrations e sai |
| Variáveis vazias no compose | rodou de fora da pasta que contém o `.env` |
| `denied` ao puxar imagens | pacotes do GHCR ainda privados |
| `"issuer":"local"` no log do Caddy | DNS errado ou porta 80 fechada |
| Login social em 404 após consentimento | `GOOGLE_REDIRECT_URL` sem o prefixo `/api` |
| `redirect_uri_mismatch` | URL não cadastrada, ou diferente, no Google Cloud Console |
| Chave SSH ignorada sem erro claro | permissões frouxas em `~/.ssh` ou `authorized_keys` |
| `sudo` falha para o `deploy` | intencional: usuário sem senha e fora do grupo `sudo` |
| API não sobe, erro de parse de URL | senha do banco com caractere reservado (corrigido: a DSN agora é montada com `net/url`) |
| Frontend chamando `localhost:8080` | `PUBLIC_API_URL` não exposta no build (corrigido: `envPrefix` no `vite.config.ts`) |
