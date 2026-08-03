#!/usr/bin/env bash
# Ensaia o ciclo de rollback num ambiente descartável, sem tocar em produção:
#
#   ./scripts/testar-rollback.sh
#
# O que ele prova, que nenhum outro teste do projeto cobre:
#
#   1. o backup carimba schema e imagem, e o carimbo entra no nome do arquivo;
#   2. a deduplicação enxerga MUDANÇA DE VERSÃO, não só mudança de dado — sem
#      isso o retrato pré-migration seria descartado justamente quando importa;
#   3. testar-restore.sh reprova um manifesto que discorda do dump;
#   4. rollback.sh RECUSA voltar para uma imagem cujo schema é anterior ao do
#      banco, que é a quebra silenciosa que o script existe para impedir.
#
# Sobe o próprio Postgres e a própria imagem de migrations, e destrói os dois no
# fim. Não lê nem escreve nada da VPS.

set -euo pipefail

RAIZ=$(cd "$(dirname "$0")/.." && pwd)
IMAGEM_POSTGRES="${IMAGEM_POSTGRES:-postgres:16-alpine}"
PROJETO="agendago_teste_rollback_$$"
REPO_TESTE="agendago-teste-$$"

log() { printf '%s  %s\n' "$(date '+%F %T')" "$*"; }
falhou=0
verificar() {
	if [ "$2" = "$3" ]; then
		printf '  ✓ %s\n' "$1"
	else
		printf '  ✗ %s\n     esperado: %s\n     obtido:   %s\n' "$1" "$2" "$3"
		falhou=1
	fi
}

TRABALHO=$(mktemp -d)
limpar() {
	docker compose -f "$TRABALHO/docker-compose.prod.yml" down -v >/dev/null 2>&1 || true
	docker rmi -f "$REPO_TESTE/agendago-migrations:antiga" >/dev/null 2>&1 || true
	rm -rf "$TRABALHO"
}
trap limpar EXIT

# ── Ambiente descartável ─────────────────────────────────────────────────────
#
# Só o postgres: backup.sh procura o container `api` para carimbar a imagem e
# lida com a ausência dele (carimba "desconhecida"), que é um caminho que também
# vale testar.
cat >"$TRABALHO/docker-compose.prod.yml" <<COMPOSE
services:
  postgres:
    image: $IMAGEM_POSTGRES
    environment:
      POSTGRES_DB: agendago
      POSTGRES_USER: agendago
      POSTGRES_PASSWORD: teste
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U agendago']
      interval: 2s
      timeout: 5s
      retries: 15
COMPOSE

cat >"$TRABALHO/.env" <<'ENV'
POSTGRES_DB=agendago
POSTGRES_USER=agendago
POSTGRES_PASSWORD=teste
ENV

export PASTA_STACK="$TRABALHO"
export PASTA_BACKUP="$TRABALHO/backups"
export ARQUIVO_COMPOSE="$TRABALHO/docker-compose.prod.yml"
# Os scripts sob teste chamam `docker compose` por conta própria, sem -p. Sem
# fixar o projeto no ambiente, eles falariam com um stack diferente do que este
# teste subiu — e o erro seria "service postgres is not running".
export COMPOSE_PROJECT_NAME="$PROJETO"
mkdir -p "$PASTA_BACKUP"

compose() { docker compose -f "$ARQUIVO_COMPOSE" "$@"; }
psql_teste() { compose exec -T postgres psql -tAX -U agendago -d agendago "$@"; }

log "subindo o Postgres descartável"
compose up -d --wait >/dev/null 2>&1 || compose up -d >/dev/null
for _ in $(seq 1 30); do
	compose exec -T postgres pg_isready -U agendago >/dev/null 2>&1 && break
	sleep 1
done

# Aplica as migrations reais e monta um flyway_schema_history equivalente ao que
# o Flyway produziria — é dele que sai o carimbo de versão.
log "aplicando as migrations do repositório"
psql_teste -c "CREATE TABLE flyway_schema_history (installed_rank INT, version VARCHAR(50), success BOOLEAN)" >/dev/null
ultima=0
for arquivo in $(find "$RAIZ/backend/migrations" -name 'V*.sql' | sort -t V -k2 -n); do
	versao=$(basename "$arquivo" | sed -n 's/^V\([0-9]\+\)__.*/\1/p')
	compose exec -T postgres psql -v ON_ERROR_STOP=1 -q -U agendago -d agendago <"$arquivo" >/dev/null
	psql_teste -c "INSERT INTO flyway_schema_history VALUES ($versao, '$versao', true)" >/dev/null
	ultima=$versao
done
log "banco no schema V$ultima"

# ── 1. O backup carimba ──────────────────────────────────────────────────────
log "teste 1: o backup carimba schema e imagem"
"$RAIZ/scripts/backup.sh" >/dev/null
backup=$(find "$PASTA_BACKUP" -name 'agendago-*.sql.gz' | head -1)
verificar "nome do arquivo traz o schema" "sim" "$([ -n "$backup" ] && [[ "$backup" == *"-schema$ultima.sql.gz" ]] && echo sim || echo nao)"
verificar "manifesto registra o schema" "$ultima" "$(sed -n 's/.*"schema_version": *"\([^"]*\)".*/\1/p' "$backup.json")"
verificar "manifesto registra a imagem" "desconhecida" "$(sed -n 's/.*"image_tag": *"\([^"]*\)".*/\1/p' "$backup.json")"

# ── 2. Deduplicação sensível à versão ────────────────────────────────────────
log "teste 2: deduplicação enxerga mudança de versão"
"$RAIZ/scripts/backup.sh" >/dev/null
verificar "dado e versão iguais: nenhum arquivo novo" "1" "$(find "$PASTA_BACKUP" -name 'agendago-*.sql.gz' | wc -l)"

# Mesmo dado, schema novo: PRECISA gerar um ponto de restauração próprio, senão
# o retrato pré-migration se perde exatamente no cenário do rollback.
psql_teste -c "INSERT INTO flyway_schema_history VALUES (99, '99', true)" >/dev/null
"$RAIZ/scripts/backup.sh" >/dev/null
verificar "schema novo com o mesmo dado: arquivo novo" "2" "$(find "$PASTA_BACKUP" -name 'agendago-*.sql.gz' | wc -l)"
psql_teste -c "DELETE FROM flyway_schema_history WHERE version = '99'" >/dev/null
novo=$(find "$PASTA_BACKUP" -name 'agendago-*schema99.sql.gz' | head -1)
rm -f "$novo" "$novo.sha256" "$novo.json"

# ── 3. Manifesto mentiroso é reprovado ───────────────────────────────────────
#
# O manifesto precisa ser adulterado DEPOIS do dump: mexer no
# flyway_schema_history antes faria o dump e o manifesto concordarem no valor
# novo, e não haveria divergência nenhuma para detectar.
log "teste 3: testar-restore.sh reprova manifesto que discorda do dump"
adulterado="$PASTA_BACKUP/agendago-adulterado-schema1.sql.gz"
cp "$backup" "$adulterado"
cp "$backup.sha256" "$adulterado.sha256"
sed "s/\"schema_version\": \"$ultima\"/\"schema_version\": \"1\"/" "$backup.json" >"$adulterado.json"

if PASTA_BACKUP="$PASTA_BACKUP" PASTA_STACK="$TRABALHO" "$RAIZ/scripts/testar-restore.sh" "$adulterado" >/dev/null 2>&1; then
	verificar "restore com manifesto divergente" "reprovado" "aprovado"
else
	verificar "restore com manifesto divergente" "reprovado" "reprovado"
fi
rm -f "$adulterado" "$adulterado.sha256" "$adulterado.json"

log "teste 3b: testar-restore.sh aprova o backup íntegro"
if PASTA_BACKUP="$PASTA_BACKUP" PASTA_STACK="$TRABALHO" "$RAIZ/scripts/testar-restore.sh" "$backup" >/dev/null 2>&1; then
	verificar "restore do backup carimbado" "aprovado" "aprovado"
else
	verificar "restore do backup carimbado" "aprovado" "reprovado"
fi

# ── 4. O guard do rollback ───────────────────────────────────────────────────
#
# Uma imagem de migrations "antiga": só as migrations até a metade do histórico.
# É o retrato de uma release anterior, que é o que se tenta subir num rollback.
log "teste 4: rollback.sh recusa voltar para schema anterior ao do banco"
metade=$((ultima / 2))
mkdir -p "$TRABALHO/sql-antigo"
for arquivo in "$RAIZ"/backend/migrations/V*.sql; do
	versao=$(basename "$arquivo" | sed -n 's/^V\([0-9]\+\)__.*/\1/p')
	[ "$versao" -le "$metade" ] && cp "$arquivo" "$TRABALHO/sql-antigo/"
done
docker build -q -t "$REPO_TESTE/agendago-migrations:antiga" -f - "$TRABALHO/sql-antigo" >/dev/null <<'DOCKERFILE'
FROM alpine:3
COPY . /flyway/sql
DOCKERFILE

saida=$(IMAGE_REPO="$REPO_TESTE" PASTA_STACK="$TRABALHO" ARQUIVO_COMPOSE="$ARQUIVO_COMPOSE" \
	"$RAIZ/scripts/rollback.sh" antiga 2>&1 || true)
verificar "recusa o rollback" "sim" "$(echo "$saida" | grep -q 'banco está À FRENTE' && echo sim || echo nao)"
verificar "aponta o restaurar.sh" "sim" "$(echo "$saida" | grep -q 'restaurar.sh' && echo sim || echo nao)"
verificar "não fixou IMAGE_TAG no .env" "nao" "$(grep -q '^IMAGE_TAG=' "$TRABALHO/.env" && echo sim || echo nao)"

echo
if [ "$falhou" -eq 0 ]; then
	log "ok: o ciclo de rollback está protegido"
else
	log "ERRO: há verificações falhando acima"
	exit 1
fi
