#!/usr/bin/env bash
# Restaura um backup NO BANCO DE PRODUÇÃO e devolve a stack à versão de código
# que gerou aquele dump. Rode na VPS:
#
#   ./scripts/restaurar.sh                        # o backup mais recente
#   ./scripts/restaurar.sh <arquivo.sql.gz>       # um backup específico
#
# Exige --sim-eu-quero para executar de fato: o passo do meio é um DROP SCHEMA,
# e um script destrutivo que roda sem confirmação é uma questão de tempo.
#
# É o par de rollback.sh. Aquele volta só o código, e recusa quando o banco está
# à frente; este é a saída para exatamente esse caso. O manifesto gravado por
# backup.sh é o que fecha o ciclo: o dump sabe de que versão de código ele veio,
# então restaurar traz de volta o PAR que funcionava junto, sem ninguém precisar
# lembrar qual imagem era.

set -euo pipefail

PASTA_STACK="${PASTA_STACK:-$HOME/agendago}"
PASTA_BACKUP="${PASTA_BACKUP:-$HOME/backups}"
COMPOSE_FILE="$PASTA_STACK/docker-compose.prod.yml"

log() { printf '%s  %s\n' "$(date '+%F %T')" "$*"; }

# shellcheck source=lib-banco.sh
. "$(dirname "$0")/lib-banco.sh"

arquivo=""
confirmado=false
for arg in "$@"; do
	case "$arg" in
	--sim-eu-quero) confirmado=true ;;
	*) arquivo="$arg" ;;
	esac
done

if [ -z "$arquivo" ]; then
	arquivo=$(find "$PASTA_BACKUP" -maxdepth 1 -name 'agendago-*.sql.gz' -printf '%T@ %p\n' 2>/dev/null |
		sort -rn | head -1 | cut -d' ' -f2-)
fi
if [ -z "$arquivo" ] || [ ! -f "$arquivo" ]; then
	log "ERRO: nenhum backup encontrado em $PASTA_BACKUP"
	exit 1
fi

schema_do_backup=$(campo_manifesto "$arquivo.json" schema_version)
imagem_do_backup=$(campo_manifesto "$arquivo.json" image_tag)
criado_em=$(campo_manifesto "$arquivo.json" criado_em)

cd "$PASTA_STACK"
set -a
# shellcheck disable=SC1091
. ./.env
set +a

compose() { docker compose -f "$COMPOSE_FILE" "$@"; }
consultar() { compose exec -T postgres psql -tAX -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$1"; }

if [ "$confirmado" = false ]; then
	cat <<RESUMO

Vai restaurar:

  arquivo   $(basename "$arquivo")
  gerado    ${criado_em:-desconhecido}
  schema    V${schema_do_backup:-?}
  imagem    ${imagem_do_backup:-desconhecida}

Sobre o banco atual:

  schema    V$(versao_de_schema)
  dados     $(consultar "SELECT count(*) FROM providers") prestador(es), $(consultar "SELECT count(*) FROM appointments") agendamento(s)

O banco atual será APAGADO e substituído pelo conteúdo do backup. Tudo que
entrou depois de ${criado_em:-a data do dump} se perde.

Para executar de verdade:

  $0 $(basename "$arquivo") --sim-eu-quero

RESUMO
	exit 1
fi

# Confere o hash antes de destruir qualquer coisa: um arquivo corrompido no
# disco viraria um erro de sintaxe do psql no meio da restauração, com o banco
# de produção já apagado.
if [ -f "$arquivo.sha256" ]; then
	esperado=$(cat "$arquivo.sha256")
	atual=$(gunzip -c "$arquivo" | grep -v '^\\\(un\)\?restrict ' | sha256sum | cut -d' ' -f1)
	if [ "$esperado" != "$atual" ]; then
		log "ERRO: hash não confere — o arquivo mudou depois de gravado; nada foi tocado"
		exit 1
	fi
	log "hash confere"
fi

# Um retrato do que existe agora, antes de apagar. Se a restauração for a
# decisão errada, este é o caminho de volta — e ele não existiria se o script
# confiasse no backup de ontem.
log "gravando um backup do estado ATUAL antes de sobrescrever"
"$(dirname "$0")/backup.sh"

# A API precisa parar: com ela escrevendo, o DROP SCHEMA trava em lock e um
# INSERT no meio do restore produziria um banco meio velho, meio novo. O
# postgres continua de pé — é ele que vai receber o dump.
log "parando api e web"
compose stop api web

log "recriando o schema"
consultar "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" >/dev/null

log "restaurando $(basename "$arquivo")"
# ON_ERROR_STOP: sem ele o psql segue depois de um erro e termina com status 0,
# o que deixaria um banco parcialmente restaurado passar por bom.
if ! gunzip -c "$arquivo" | compose exec -T postgres \
	psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null; then
	log "ERRO: a restauração falhou — a api e o web seguem parados, de propósito"
	exit 1
fi

log "$(validar_banco)"

# Aqui o manifesto se paga: sobe a imagem que gerou este dump, e não a que
# estava no ar. Restaurar dado antigo sob código novo é o mesmo problema do
# rollback, ao contrário — o código esperaria colunas que o dump não tem.
if [ -n "$imagem_do_backup" ] && [ "$imagem_do_backup" != "desconhecida" ]; then
	log "fixando IMAGE_TAG em $imagem_do_backup (a versão que gerou este backup)"
	if grep -q '^IMAGE_TAG=' .env; then
		sed -i "s|^IMAGE_TAG=.*|IMAGE_TAG=$imagem_do_backup|" .env
	else
		printf 'IMAGE_TAG=%s\n' "$imagem_do_backup" >>.env
	fi
	IMAGE_TAG="$imagem_do_backup" compose pull --quiet api web
else
	log "AVISO: o backup não registra a imagem; subindo com o IMAGE_TAG atual do .env"
fi

compose up -d

if [ -n "${DOMINIO:-}" ]; then
	for _ in $(seq 1 12); do
		if curl -fsS "https://$DOMINIO/api/health" >/dev/null 2>&1 &&
			curl -fsS -o /dev/null "https://$DOMINIO/" 2>/dev/null; then
			log "ok: aplicação no ar com o banco restaurado"
			exit 0
		fi
		sleep 5
	done
	log "ERRO: a aplicação não respondeu após a restauração"
	exit 1
fi

log "ok: stack no ar (DOMINIO não definido; verificação HTTP pulada)"
