#!/usr/bin/env bash
# Volta a stack para uma versão já implantada, com a checagem que impede o
# rollback silenciosamente quebrado. Rode no servidor, como o usuário agendago:
#
#   ./scripts/rollback.sh <sha-do-commit>   # fixa aquela versão
#   ./scripts/rollback.sh latest            # volta a acompanhar o deploy
#
# scripts/implantar.sh entrega cada imagem com a tag do SHA, e o servidor guarda
# as anteriores, então voltar código é trocar uma variável e subir de novo —
# segundos. Só dá para voltar para um SHA que já passou por um deploy aqui. O que NÃO volta sozinho é o
# banco: o Flyway é forward-only, e uma migration já aplicada continua aplicada.
#
# Por isso este script compara a versão de schema que a imagem alvo espera com a
# que o banco tem, e RECUSA o rollback que deixaria o código antigo falando com
# um schema à frente dele. Sem essa checagem, o `up -d` sobe normalmente e a
# quebra só aparece no primeiro INSERT — em produção, com usuário na frente.

set -euo pipefail

PASTA_STACK="${PASTA_STACK:-$HOME/agendago}"
# Sobrescrito por scripts/testar-rollback.sh. Em produção nunca muda.
COMPOSE_FILE="${ARQUIVO_COMPOSE:-$PASTA_STACK/docker-compose.prod.yml}"

log() { printf '%s  %s\n' "$(date '+%F %T')" "$*"; }

alvo="${1:-}"
forcar=false
[ "${2:-}" = "--forcar" ] && forcar=true

if [ -z "$alvo" ]; then
	log "uso: $0 <sha-do-commit|latest> [--forcar]"
	exit 1
fi

cd "$PASTA_STACK"

set -a
# shellcheck disable=SC1091
. ./.env
set +a

compose() { docker compose -f "$COMPOSE_FILE" "$@"; }

repo="${IMAGE_REPO:-localhost/agendago}"

# ── A checagem que dá sentido ao script ──────────────────────────────────────
#
# Vem ANTES de mexer em qualquer coisa, de propósito: um rollback recusado não
# deixa nada para trás.
#
# A imagem de migrations carrega os .sql dentro dela, então a maior versão que
# ela conhece é a versão de schema daquela release. Comparar com o que o banco
# tem responde a única pergunta que importa antes de voltar código.
imagem_migrations="$repo/agendago-migrations:$alvo"
# Sem `docker pull` de fallback: as imagens não estão em registry nenhum, e um
# pull só iria procurar o nome em outro lugar.
if ! docker image inspect "$imagem_migrations" >/dev/null 2>&1; then
	log "ERRO: $imagem_migrations não existe neste servidor (docker image ls $repo/agendago-migrations)"
	exit 1
fi

versao_da_imagem=$(docker run --rm --entrypoint sh "$imagem_migrations" \
	-c "ls /flyway/sql | sed -n 's/^V\([0-9]\+\)__.*/\1/p' | sort -n | tail -1" 2>/dev/null || true)
versao_do_banco=$(compose exec -T postgres psql -tAX -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
	-c "SELECT version FROM flyway_schema_history WHERE success ORDER BY installed_rank DESC LIMIT 1" \
	2>/dev/null | tr -d '[:space:]' || true)

log "schema: imagem alvo conhece até V${versao_da_imagem:-?}, banco está em V${versao_do_banco:-?}"

if [ -n "$versao_da_imagem" ] && [ -n "$versao_do_banco" ] &&
	[ "$versao_da_imagem" -lt "$versao_do_banco" ] 2>/dev/null; then
	if [ "$forcar" = false ]; then
		cat >&2 <<AVISO

O banco está À FRENTE da versão que você quer subir (V$versao_do_banco > V$versao_da_imagem).

O Flyway não desfaz migration. Duas saídas, nesta ordem de preferência:

  1. Restaurar o backup do estado anterior, que já traz a imagem casada:
       ./scripts/restaurar.sh <backup-schema$versao_da_imagem.sql.gz> --sim-eu-quero

  2. Se você TEM CERTEZA de que as migrations entre V$versao_da_imagem e
     V$versao_do_banco são compatíveis com o código antigo (só colunas
     anuláveis, nada removido), repita com --forcar.

AVISO
		exit 1
	fi
	log "AVISO: prosseguindo por --forcar, com o banco à frente do código"
fi

for imagem in agendago-api agendago-web; do
	if ! docker image inspect "$repo/$imagem:$alvo" >/dev/null 2>&1; then
		log "ERRO: $repo/$imagem:$alvo não existe neste servidor (docker image ls $repo/$imagem)"
		exit 1
	fi
done

# ── Fixa a tag ───────────────────────────────────────────────────────────────
#
# Reescreve a linha em vez de acrescentar. Um `>> .env` repetido acumula várias
# IMAGE_TAG no arquivo: o Compose usa a última e "funciona", mas o arquivo passa
# a mentir sobre o que está no ar, e ninguém sabe qual linha apagar depois.
if grep -q '^IMAGE_TAG=' .env; then
	sed -i "s|^IMAGE_TAG=.*|IMAGE_TAG=$alvo|" .env
else
	printf 'IMAGE_TAG=%s\n' "$alvo" >>.env
fi
log "IMAGE_TAG fixado em $alvo no .env"

log "subindo a versão $alvo"
IMAGE_TAG="$alvo" compose up -d

# ── Verificação ──────────────────────────────────────────────────────────────
#
# Mesma checagem do deploy: API e frontend, pelo Caddy local — sem depender do
# túnel. Rollback que sobe quebrado sem ninguém perceber é o mesmo problema do
# deploy no escuro. Pulada quando a stack não tem Caddy (o ensaio de
# scripts/testar-rollback.sh).
if compose config --services | grep -qx caddy; then
	no_ar=false
	for _ in $(seq 1 12); do
		if curl -fsS -o /dev/null "http://127.0.0.1:${PORTA_CADDY:-8481}/api/health" 2>/dev/null &&
			curl -fsS -o /dev/null "http://127.0.0.1:${PORTA_CADDY:-8481}/" 2>/dev/null; then
			no_ar=true
			break
		fi
		sleep 5
	done
	if [ "$no_ar" = false ]; then
		log "ERRO: a aplicação não respondeu após o rollback"
		exit 1
	fi
	log "ok: aplicação no ar na versão $alvo"
else
	log "ok: stack no ar na versão $alvo (sem Caddy nesta stack; verificação HTTP pulada)"
fi

# implantar.sh passa IMAGE_TAG por ambiente, e ambiente vence o .env no
# Compose: o pin sobrevive a um `up -d` manual, mas NÃO ao próximo deploy.
log "lembrete: o próximo scripts/implantar.sh ignora este pin — reverta o commit ruim antes"
