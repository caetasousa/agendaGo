#!/usr/bin/env bash
# Volta a stack para uma versão publicada, com a checagem que impede o rollback
# silenciosamente quebrado. Rode na VPS:
#
#   ./scripts/rollback.sh <sha-do-commit>   # fixa aquela versão
#   ./scripts/rollback.sh latest            # volta a acompanhar o deploy
#
# O CI publica cada imagem no GHCR com a tag do SHA, então voltar código é
# trocar uma variável e subir de novo — segundos. O que NÃO volta sozinho é o
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

repo="${IMAGE_REPO:-ghcr.io/caetasousa}"

# ── A checagem que dá sentido ao script ──────────────────────────────────────
#
# Vem ANTES de puxar o resto das imagens, de propósito: um rollback que vai ser
# recusado não deveria custar o download da stack inteira.
#
# A imagem de migrations carrega os .sql dentro dela, então a maior versão que
# ela conhece é a versão de schema daquela release. Comparar com o que o banco
# tem responde a única pergunta que importa antes de voltar código.
imagem_migrations="$repo/agendago-migrations:$alvo"
docker image inspect "$imagem_migrations" >/dev/null 2>&1 || docker pull -q "$imagem_migrations" >/dev/null

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

log "puxando as imagens da versão $alvo"
IMAGE_TAG="$alvo" compose pull --quiet
IMAGE_TAG="$alvo" compose up -d

# ── Verificação ──────────────────────────────────────────────────────────────
#
# Mesma checagem do deploy: API e frontend. Rollback que sobe quebrado sem
# ninguém perceber é o mesmo problema do deploy no escuro.
if [ -n "${DOMINIO:-}" ]; then
	for _ in $(seq 1 12); do
		if curl -fsS "https://$DOMINIO/api/health" >/dev/null 2>&1 &&
			curl -fsS -o /dev/null "https://$DOMINIO/" 2>/dev/null; then
			log "ok: aplicação no ar na versão $alvo"
			exit 0
		fi
		sleep 5
	done
	log "ERRO: a aplicação não respondeu após o rollback"
	exit 1
fi

log "ok: stack no ar na versão $alvo (DOMINIO não definido; verificação HTTP pulada)"

# O próximo deploy do CI passa IMAGE_TAG por ambiente, e ambiente vence o .env
# no Compose: o pin sobrevive a um `up -d` manual, mas NÃO ao próximo push na
# main. Voltar a acompanhar o deploy é `./scripts/rollback.sh latest`.
log "lembrete: o próximo deploy do CI ignora este pin — reverta o commit ruim na main"
