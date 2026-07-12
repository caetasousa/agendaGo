# Produção

Como buildar e rodar o agendaGo fora do ambiente de desenvolvimento. O
`docker-compose.yml` da raiz é **só para dev** (hot reload, portas expostas,
rate limit desligado) — produção usa as imagens dedicadas abaixo.

## Imagens

| Imagem | Dockerfile | O que gera |
|---|---|---|
| API | `backend/Dockerfile.prod` | binário Go estático (multi-stage, usuário sem privilégios, healthcheck em `/health`, tzdata embutido via `-tags timetzdata`) |
| Web | `frontend/Dockerfile.prod` | servidor Node autônomo do adapter-node (`node build`, porta 3000) |

```bash
# API
docker build -f backend/Dockerfile.prod -t agendago-api backend/

# Web — PUBLIC_API_URL é resolvida em tempo de BUILD (import.meta.env do Vite)
docker build -f frontend/Dockerfile.prod \
  --build-arg PUBLIC_API_URL=https://api.seudominio.com \
  -t agendago-web frontend/
```

## Variáveis de ambiente da API

| Variável | Obrigatória | Padrão | Para quê |
|---|---|---|---|
| `DB_HOST` / `DB_PORT` / `DB_NAME` / `DB_USER` / `DB_PASSWORD` | sim | — | conexão com o Postgres |
| `APP_ENV` | sim (produção) | — | `production` liga o atributo `Secure` do cookie de sessão |
| `FRONTEND_ORIGIN` | sim (produção) | `http://localhost:5173` | origem permitida no CORS **e** base do link nos emails (ex.: recuperação de senha) — aponte para o domínio real do frontend, senão os links dos emails apontam para `localhost` |
| `PORT` | não | `8080` | porta do servidor HTTP |
| `ADMIN_EMAIL` / `ADMIN_SENHA` | não | — | semeiam o admin no boot (vazias = sem admin) |
| `RATE_LIMIT_LOGIN_POR_MINUTO` | não | `10` | teto de logins por IP/minuto (0 desliga) |
| `RATE_LIMIT_CONVIDADO_POR_MINUTO` | não | `10` | teto de agendamentos de convidado por IP/minuto (0 desliga) |
| `SMTP_HOST` | não | — | servidor SMTP (ex.: `smtp-relay.brevo.com`). **Vazio desliga o envio**: os emails são só logados, não enviados |
| `SMTP_PORT` | não | `587` | porta do servidor SMTP |
| `SMTP_USER` / `SMTP_PASSWORD` | não | — | credenciais SMTP (vazias = sem autenticação, ex.: Mailpit) |
| `SMTP_STARTTLS` | não | `true` | exige STARTTLS na conexão (desligue só contra o Mailpit) |
| `EMAIL_REMETENTE` / `EMAIL_REMETENTE_NOME` | não | — | remetente dos emails (precisa ser o email verificado no provedor) |

## Migrations

As migrations de `backend/migrations/` devem rodar **antes** de subir a API —
em produção, o mesmo Flyway do compose serve como job pontual:

```bash
docker run --rm -v ./backend/migrations:/flyway/sql \
  -e FLYWAY_URL=jdbc:postgresql://SEU_HOST:5432/agendago \
  -e FLYWAY_USER=... -e FLYWAY_PASSWORD=... \
  flyway/flyway:10-alpine migrate
```

## Comportamento em produção

- **Desligamento gracioso**: a API responde a `SIGTERM`/`SIGINT` parando de
  aceitar conexões novas e dando até 10s para as requisições em andamento
  concluírem — deploys não derrubam requests no meio.
- **Timeouts**: read/write 15s, idle 60s, cabeçalho e corpo limitados a 1 MiB.
- **TLS**: a API fala HTTP puro; em produção coloque um proxy reverso
  (Caddy, nginx, load balancer) na frente para terminar TLS.
- **Postgres**: não exponha a porta 5432 publicamente — em produção o banco
  só precisa ser alcançável pela API e pelo job de migration.
- **Swagger**: `/swagger/*` fica exposto na API; se não quiser documentação
  pública, bloqueie o caminho no proxy reverso.
