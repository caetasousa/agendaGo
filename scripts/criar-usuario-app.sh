#!/usr/bin/env bash
# Cria (ou atualiza) o usuário de banco que a API usa em produção.
#
# Por que existe: por padrão a API se conecta com o DONO do banco, que pode
# criar, alterar e derrubar qualquer tabela. Uma falha de execução remota na API
# herdaria esse poder. Este usuário tem só o que a aplicação precisa —
# SELECT/INSERT/UPDATE/DELETE — e nenhum DDL. Quem aplica migration continua
# sendo o dono, pelo Flyway.
#
# Rode na VPS, dentro da pasta da stack, como o usuário `deploy`:
#
#   ./scripts/criar-usuario-app.sh
#
# Depois acrescente ao .env e suba de novo:
#
#   DB_USER=agendago_app
#   DB_PASSWORD=<a senha que o script pediu>
#
# É idempotente: rodar de novo só atualiza a senha e reaplica os GRANTs.

set -euo pipefail

PASTA_STACK="${PASTA_STACK:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
USUARIO_APP="${USUARIO_APP:-agendago_app}"

cd "$PASTA_STACK"

set -a
# shellcheck disable=SC1091
. ./.env
set +a

if [ -z "${DB_PASSWORD:-}" ]; then
	echo "Defina a senha do usuário da aplicação antes de rodar:"
	echo
	echo "  DB_PASSWORD=\$(openssl rand -base64 24) ./scripts/criar-usuario-app.sh"
	echo
	echo "e guarde o valor — ele precisa ir para o .env como DB_PASSWORD."
	exit 1
fi

compose() { docker compose -f "$PASTA_STACK/docker-compose.prod.yml" "$@"; }

echo "criando/atualizando o usuário ${USUARIO_APP} no banco ${POSTGRES_DB}"

# A senha entra por variável do psql (:'senha'), nunca interpolada na string
# SQL — assim um caractere especial não vira parte do comando.
compose exec -T postgres psql -v ON_ERROR_STOP=1 \
	-U "$POSTGRES_USER" -d "$POSTGRES_DB" \
	-v usuario="$USUARIO_APP" -v senha="$DB_PASSWORD" \
	-v dono="$POSTGRES_USER" -v banco="$POSTGRES_DB" <<'SQL'
-- CREATE ROLE não tem IF NOT EXISTS: monta o comando e só executa se faltar.
-- (\gexec roda cada linha do resultado como SQL; não dá para usar um bloco
-- DO $$ ... $$ aqui, porque o psql não substitui variáveis dentro dele.)
SELECT format('CREATE ROLE %I LOGIN', :'usuario')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'usuario')
\gexec

-- sem SUPERUSER, CREATEDB, CREATEROLE, REPLICATION ou BYPASSRLS
ALTER ROLE :"usuario" WITH LOGIN PASSWORD :'senha'
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;

GRANT CONNECT ON DATABASE :"banco" TO :"usuario";
GRANT USAGE ON SCHEMA public TO :"usuario";

-- o que a aplicação faz: ler e escrever linhas. Nada de DDL.
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO :"usuario";
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO :"usuario";

-- as tabelas que as PRÓXIMAS migrations criarem também já nascem acessíveis,
-- senão todo deploy com migration exigiria rodar este script de novo
ALTER DEFAULT PRIVILEGES FOR ROLE :"dono" IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO :"usuario";
ALTER DEFAULT PRIVILEGES FOR ROLE :"dono" IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO :"usuario";
SQL

echo
echo "ok. agora ajuste o .env da stack:"
echo "  DB_USER=${USUARIO_APP}"
echo "  DB_PASSWORD=<a senha usada acima>"
echo
echo "e recrie a API:  docker compose -f docker-compose.prod.yml up -d api"
