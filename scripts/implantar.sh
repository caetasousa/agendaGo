#!/usr/bin/env bash
# Publica o commit atual no servidor (a distro ubuntu-server do WSL). Roda na
# distro de desenvolvimento, da raiz do repositório:
#
#   ./scripts/implantar.sh
#
# É o antigo job `implantar` do GitHub Actions, sem a esteira: builda as três
# imagens aqui, entrega ao Docker rootless do usuário `agendago` por SSH e sobe
# a stack. O servidor continua sem código-fonte — recebe as imagens prontas e
# os três arquivos de configuração (docker-compose.prod.yml, Caddyfile e .env;
# o .env mora só lá, com os segredos).
#
# Sem esteira, ninguém mais roda os testes antes: rode `make test` (e, havendo
# tempo, `make test-all`) antes de implantar.

set -euo pipefail

RAIZ=$(cd "$(dirname "$0")/.." && pwd)
SERVIDOR="${SERVIDOR:-agendago@127.0.0.1}"
PORTA_SSH="${PORTA_SSH:-2222}"
CHAVE_SSH="${CHAVE_SSH:-$HOME/.ssh/agendago_servidor}"
PORTA_CADDY="${PORTA_CADDY:-8481}"
REPO="${IMAGE_REPO:-localhost/agendago}"
# Horário do backup diário, no fuso do SERVIDOR (o cron do Ubuntu ignora
# CRON_TZ).
AGENDA_BACKUP="${AGENDA_BACKUP:-0 3 * * *}"

log() { printf '%s  %s\n' "$(date '+%F %T')" "$*"; }
remoto() { ssh -o ConnectTimeout=15 -i "$CHAVE_SSH" -p "$PORTA_SSH" "$SERVIDOR" "$@"; }

cd "$RAIZ"

# A tag é o SHA do commit, e é ela que backup.sh carimba e rollback.sh usa para
# voltar. Com alteração não commitada, a imagem levaria um SHA que não contém o
# que está nela — e o rollback para "aquela versão" subiria outra coisa.
if [ -n "$(git status --porcelain)" ]; then
	log "ERRO: há alterações não commitadas; implante só o que está num commit"
	git status --short
	exit 1
fi
tag=$(git rev-parse HEAD)

# nome da imagem | contexto | Dockerfile — as três que vão para produção.
imagens=(
	"agendago-api|backend|backend/Dockerfile.prod"
	"agendago-web|frontend|frontend/Dockerfile.prod"
	"agendago-migrations|backend|backend/Dockerfile.migrations"
)

nomes=()
for linha in "${imagens[@]}"; do
	IFS='|' read -r nome contexto dockerfile <<<"$linha"
	log "build de $nome:${tag:0:12}"
	docker build -q -t "$REPO/$nome:$tag" -f "$dockerfile" "$contexto" >/dev/null
	nomes+=("$REPO/$nome:$tag")
done

# docker save numa chamada só: as camadas comuns às imagens (a base Alpine)
# atravessam uma vez. `latest` acompanha o deploy para que um `up -d` manual ou
# o backup do cron enxerguem a versão no ar.
log "enviando as imagens ao servidor"
docker save "${nomes[@]}" | gzip -1 | remoto "gunzip | docker load -q >/dev/null"
remoto "for img in ${nomes[*]}; do docker tag \"\$img\" \"\${img%:*}:latest\"; done"

# Os scripts de operação também vão: backup.sh roda por cron; rollback.sh e
# restaurar.sh são os caminhos de volta, e precisam estar no servidor ANTES do
# incidente que os torna necessários. lib-banco.sh vai junto porque os dois
# últimos a carregam (não recebe +x: é sourced). testar-rollback.sh fica: ele
# monta o ensaio a partir de backend/migrations, que o servidor não tem.
log "sincronizando a configuração da stack"
remoto 'mkdir -p ~/agendago/scripts ~/backups'
scp -q -o ConnectTimeout=15 -i "$CHAVE_SSH" -P "$PORTA_SSH" \
	docker-compose.prod.yml Caddyfile "$SERVIDOR:agendago/"
scp -q -o ConnectTimeout=15 -i "$CHAVE_SSH" -P "$PORTA_SSH" \
	scripts/backup.sh scripts/testar-restore.sh scripts/rollback.sh scripts/restaurar.sh \
	scripts/lib-banco.sh scripts/criar-usuario-app.sh "$SERVIDOR:agendago/scripts/"
remoto 'cd ~/agendago/scripts && chmod +x backup.sh testar-restore.sh rollback.sh restaurar.sh criar-usuario-app.sh'

if ! remoto 'test -f ~/agendago/.env'; then
	log "ERRO: falta o ~/agendago/.env no servidor (modelo em .env.prod.example)"
	exit 1
fi

# Retrato do banco ANTES da migration deste deploy: sem ele, o ponto de retorno
# mais recente seria o backup da madrugada. Não trava o deploy — no primeiro,
# a stack ainda não existe.
log "backup do banco antes de migrar"
remoto '~/agendago/scripts/backup.sh' ||
	log "AVISO: backup pré-deploy não executado (a stack já existe no servidor?)"

# IMAGE_TAG pelo ambiente, que vence o .env: sobe exatamente o que acabou de
# ser buildado, e sobrescreve um pin deixado por rollback.sh.
log "subindo a stack"
remoto "cd ~/agendago && IMAGE_TAG=$tag docker compose -f docker-compose.prod.yml up -d --quiet-pull \
	&& docker image prune -f >/dev/null"

# Idempotente: reescreve só a linha do agendago, preservando o resto do crontab.
remoto "( crontab -l 2>/dev/null | grep -v 'agendago/scripts/backup.sh'
	echo '$AGENDA_BACKUP \$HOME/agendago/scripts/backup.sh >> \$HOME/backups/backup.log 2>&1'
) | crontab -"
log "backup agendado: '$AGENDA_BACKUP' no fuso $(remoto 'timedatectl show --property=Timezone --value 2>/dev/null || readlink -f /etc/localtime')"

# Confere pelo próprio servidor, sem passar pela Cloudflare: um problema no
# túnel não deve reprovar um deploy que subiu. API e página inicial — só a API
# deixaria passar um `web` quebrado.
log "verificando a aplicação"
for _ in $(seq 1 24); do
	if remoto "curl -fsS -o /dev/null http://127.0.0.1:$PORTA_CADDY/api/health \
		&& curl -fsS -o /dev/null http://127.0.0.1:$PORTA_CADDY/"; then
		log "ok: ${tag:0:12} no ar (API e frontend)"
		exit 0
	fi
	sleep 5
done
log "ERRO: a aplicação não respondeu em /api/health e/ou / após o deploy"
remoto 'cd ~/agendago && docker compose -f docker-compose.prod.yml ps'
exit 1
