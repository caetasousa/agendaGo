#!/usr/bin/env bash
# Restaura o backup mais recente num Postgres descartável e confere se o banco
# volta utilizável. Rode na VPS, junto de scripts/backup.sh:
#
#   ./scripts/testar-restore.sh
#
# Por que existe: scripts/backup.sh verifica a INTEGRIDADE do dump (que o
# arquivo não saiu truncado), o que não é a mesma coisa que provar que ele
# RESTAURA. Backup nunca restaurado é fé, não garantia — e a hora de descobrir
# que não volta não pode ser a hora em que ele é necessário.
#
# Não toca no banco de produção em momento nenhum: sobe um container próprio,
# em porta aleatória, e o destrói ao final.

set -euo pipefail

PASTA_STACK="${PASTA_STACK:-$HOME/agendago}"
PASTA_BACKUP="${PASTA_BACKUP:-$HOME/backups}"
IMAGEM_POSTGRES="${IMAGEM_POSTGRES:-postgres:16-alpine}"
CONTAINER="agendago_teste_restore_$$"

log() { printf '%s  %s\n' "$(date '+%F %T')" "$*"; }

# O papel do container descartável precisa ser o MESMO de produção. O pg_dump
# escreve `ALTER TABLE ... OWNER TO <dono>` no arquivo, e restaurar sob outro
# nome falha com `role "..." does not exist` — o ensaio reprovaria um backup
# perfeitamente bom, ou (pior) nunca teria chance de aprovar nenhum.
USUARIO_TESTE=teste
if [ -f "$PASTA_STACK/.env" ]; then
	do_env=$(sed -n 's/^POSTGRES_USER=//p' "$PASTA_STACK/.env" | tail -1)
	USUARIO_TESTE="${do_env:-teste}"
fi

arquivo="${1:-}"
if [ -z "$arquivo" ]; then
	arquivo=$(find "$PASTA_BACKUP" -maxdepth 1 -name 'agendago-*.sql.gz' -printf '%T@ %p\n' 2>/dev/null |
		sort -rn | head -1 | cut -d' ' -f2-)
fi

if [ -z "$arquivo" ] || [ ! -f "$arquivo" ]; then
	log "ERRO: nenhum backup encontrado em $PASTA_BACKUP"
	exit 1
fi

log "testando o restore de $(basename "$arquivo")"

# Confere o hash antes de restaurar: se o arquivo corrompeu no disco depois de
# gravado, o erro aparece aqui, com a causa clara, em vez de virar um erro de
# sintaxe do psql no meio da restauração.
if [ -f "$arquivo.sha256" ]; then
	esperado=$(cat "$arquivo.sha256")
	atual=$(gunzip -c "$arquivo" | grep -v '^\\\(un\)\?restrict ' | sha256sum | cut -d' ' -f1)
	if [ "$esperado" != "$atual" ]; then
		log "ERRO: hash não confere — o arquivo mudou depois de gravado"
		exit 1
	fi
	log "hash confere"
fi

limpar() { docker rm -f "$CONTAINER" >/dev/null 2>&1 || true; }
trap limpar EXIT

docker run -d --name "$CONTAINER" \
	-e POSTGRES_DB=restore_teste \
	-e POSTGRES_USER="$USUARIO_TESTE" \
	-e POSTGRES_PASSWORD=teste \
	"$IMAGEM_POSTGRES" >/dev/null

log "aguardando o Postgres descartável subir"
for _ in $(seq 1 30); do
	if docker exec "$CONTAINER" pg_isready -U "$USUARIO_TESTE" >/dev/null 2>&1; then
		break
	fi
	sleep 1
done
if ! docker exec "$CONTAINER" pg_isready -U "$USUARIO_TESTE" >/dev/null 2>&1; then
	log "ERRO: o Postgres descartável não subiu"
	exit 1
fi

log "restaurando"
# ON_ERROR_STOP: sem ele o psql segue depois de um erro e termina com status 0,
# o que faria um restore quebrado passar por bom — exatamente o que este script
# existe para impedir.
if ! gunzip -c "$arquivo" | docker exec -i "$CONTAINER" \
	psql -v ON_ERROR_STOP=1 -U "$USUARIO_TESTE" -d restore_teste >/dev/null; then
	log "ERRO: a restauração falhou"
	exit 1
fi

consultar() { docker exec "$CONTAINER" psql -tAX -U "$USUARIO_TESTE" -d restore_teste -c "$1"; }

# Sanidade: as tabelas do domínio existem e o histórico do Flyway veio junto.
# Sem checar o histórico, um dump de um banco sem migrations aplicadas passaria.
esperadas="providers clients appointments date_exceptions sessions admins"
faltando=""
for tabela in $esperadas; do
	existe=$(consultar "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='$tabela'")
	[ "$existe" = "1" ] || faltando="$faltando $tabela"
done
if [ -n "$faltando" ]; then
	log "ERRO: tabelas ausentes após o restore:$faltando"
	exit 1
fi

migrations=$(consultar "SELECT count(*) FROM flyway_schema_history WHERE success" 2>/dev/null || echo 0)
if [ "$migrations" -lt 1 ]; then
	log "ERRO: nenhuma migration bem sucedida no histórico do Flyway"
	exit 1
fi

prestadores=$(consultar "SELECT count(*) FROM providers")
agendamentos=$(consultar "SELECT count(*) FROM appointments")

log "ok: restore íntegro — $migrations migrations, $prestadores prestador(es), $agendamentos agendamento(s)"
