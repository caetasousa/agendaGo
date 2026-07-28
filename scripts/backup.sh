#!/usr/bin/env bash
# Backup completo do banco de produção: dump lógico comprimido, com verificação
# de integridade, deduplicação por conteúdo e rotação. Feito para rodar por cron
# na VPS, como o usuário `deploy`:
#
#   0 3 * * * /home/deploy/agendago/scripts/backup.sh >> /home/deploy/backups/backup.log 2>&1
#
# Por que dump lógico (pg_dump) e não backup físico ou incremental: o banco é
# pequeno, o arquivo resultante é portátil entre versões do Postgres e entre
# máquinas, e a restauração é um comando só — o que importa no dia em que ela
# for necessária, que costuma ser o pior dia do projeto.

set -euo pipefail

PASTA_STACK="${PASTA_STACK:-$HOME/agendago}"
PASTA_BACKUP="${PASTA_BACKUP:-$HOME/backups}"
RETENCAO_DIAS="${RETENCAO_DIAS:-7}"
# URL_HEARTBEAT (opcional): endereço pingado ao final de uma execução bem
# sucedida, para um monitor externo alertar quando o ping parar de chegar.

log() { printf '%s  %s\n' "$(date '+%F %T')" "$*"; }

# O Compose lê o .env do diretório atual, não do diretório do arquivo de
# compose: sem este cd, as variáveis chegariam vazias.
cd "$PASTA_STACK"

set -a
# shellcheck disable=SC1091
. ./.env
set +a

compose() { docker compose -f "$PASTA_STACK/docker-compose.prod.yml" "$@"; }

# O dump cru é temporário: só o .gz fica guardado.
temporario=$(mktemp -d)
trap 'rm -rf "$temporario"' EXIT
bruto="$temporario/dump.sql"

mkdir -p "$PASTA_BACKUP"
log "iniciando dump do banco ${POSTGRES_DB}"

compose exec -T postgres pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" >"$bruto"

# pg_dump encerra o arquivo com este marcador. Sem ele, o dump saiu truncado —
# e um backup truncado que parece válido é pior do que backup nenhum.
if ! tail -5 "$bruto" | grep -q "PostgreSQL database dump complete"; then
	log "ERRO: dump incompleto; nada foi gravado"
	exit 1
fi

# Hash do CONTEÚDO, para saber se o banco realmente mudou desde o último backup.
#
# As linhas \restrict/\unrestrict são descartadas antes do cálculo: o pg_dump
# envolve o dump com um token ALEATÓRIO a cada execução (proteção contra injeção
# de meta-comandos do psql). Sem removê-las, dois dumps de um banco idêntico
# jamais teriam o mesmo hash.
#
# Cuidado com o sentido do erro: a ordem das linhas do COPY segue a ordem física
# das tuplas, que muda com UPDATE/VACUUM. Então o hash pode diferir sem mudança
# lógica (backup a mais, inofensivo) — mas dados diferentes nunca produzem o
# mesmo hash, que seria o erro perigoso.
hash=$(grep -v '^\\\(un\)\?restrict ' "$bruto" | sha256sum | cut -d' ' -f1)

anterior=$(find "$PASTA_BACKUP" -maxdepth 1 -name 'agendago-*.sql.gz' -printf '%T@ %p\n' 2>/dev/null |
	sort -rn | head -1 | cut -d' ' -f2-)

if [ -n "$anterior" ] && [ "$hash" = "$(cat "$anterior.sha256" 2>/dev/null)" ]; then
	# Nada mudou no banco: não faz sentido guardar um segundo arquivo idêntico.
	#
	# O `touch` não é detalhe — é o que impede a rotação de apagar, dias depois,
	# o único backup existente. Num banco parado por mais tempo que a retenção,
	# sem isto você terminaria com zero backups. A data de que o backup rodou
	# fica no log, já que o nome do arquivo passa a não representá-la.
	touch "$anterior" "$anterior.sha256"
	log "sem alteração desde $(basename "$anterior"); nenhum backup novo criado"
else
	# Segundos no nome: sem eles, duas execuções no mesmo minuto (um backup
	# manual logo após o do cron, por exemplo) sobrescreveriam uma à outra.
	arquivo="$PASTA_BACKUP/agendago-$(date +%F-%H%M%S).sql.gz"
	# -n omite nome e timestamp do cabeçalho gzip, para o .gz ser reprodutível.
	gzip -9n <"$bruto" >"$arquivo.parcial"
	mv "$arquivo.parcial" "$arquivo"
	printf '%s\n' "$hash" >"$arquivo.sha256"
	# O dump contém email e telefone de pessoas reais: leitura só para o dono.
	chmod 600 "$arquivo" "$arquivo.sha256"
	log "ok: $(du -h "$arquivo" | cut -f1) em $arquivo"
fi

# Rotação por idade. -print antes de -delete para conseguir contar o que saiu.
removidos=$(find "$PASTA_BACKUP" -maxdepth 1 \( -name 'agendago-*.sql.gz' -o -name 'agendago-*.sql.gz.sha256' \) \
	-mtime "+$RETENCAO_DIAS" -print -delete | wc -l)
if [ "$removidos" -gt 0 ]; then
	log "rotação: $removidos arquivo(s) com mais de $RETENCAO_DIAS dias removido(s)"
fi

# Heartbeat para um monitor externo (opcional, via URL_HEARTBEAT).
#
# O alerta nasce da AUSÊNCIA do ping, não da presença de um erro — é o que
# resolve o problema registrado em docs/producao.md: "cron esquecido não avisa".
# Um cron desativado, uma VPS desligada e um dump que falhou produzem o mesmo
# silêncio, e todos os três merecem alerta.
#
# Por isso o ping fica na última linha: com `set -e`, qualquer falha acima
# encerra o script antes de chegar aqui. Falhar o ping em si não reprova o
# backup — o arquivo já está gravado, e derrubar o script agora só produziria
# um erro no log sem nada a corrigir no banco.
if [ -n "${URL_HEARTBEAT:-}" ]; then
	if curl -fsS --max-time 10 --retry 3 "$URL_HEARTBEAT" >/dev/null 2>&1; then
		log "heartbeat enviado"
	else
		log "AVISO: heartbeat não pôde ser enviado (o backup em si deu certo)"
	fi
fi
