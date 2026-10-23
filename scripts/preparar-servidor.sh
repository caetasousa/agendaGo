#!/usr/bin/env bash
# Prepara a distro ubuntu-server do WSL para receber o agendaGo: o usuário
# `agendago`, com acesso só por chave SSH e um Docker rootless próprio. Roda
# uma vez, como root no servidor, a partir da distro de desenvolvimento — pelo
# SSH de um usuário administrador que já exista lá (o do studygo):
#
#   ssh -i ~/.ssh/studygo_ci -p 2222 studygo@127.0.0.1 \
#     sudo bash -s -- agendago "\"$(cat ~/.ssh/agendago_servidor.pub)\"" \
#     < scripts/preparar-servidor.sh
#
# O sshd na 2222, o túnel da Cloudflare e os pacotes base são do servidor
# inteiro, montados pelo Ansible do studygo — este script só confere que
# existem e cuida do que é do agendaGo.
#
# Docker rootless, e não o Docker de sistema: todas as distros do WSL2 dividem
# a mesma rede, e um segundo dockerd de sistema brigaria com o da distro de
# desenvolvimento pelo docker0 e pelo iptables. Cada projeto tem o seu usuário,
# com daemon, imagens e volumes próprios — um não enxerga nem derruba o outro.
#
# Idempotente: rodar de novo num servidor pronto não muda nada.

set -euo pipefail

usuario=${1:?informe o usuário do projeto}
chave=${2:?informe a chave pública de deploy}

[ "$(id -u)" = 0 ] || { echo "rode como root"; exit 1; }
case "$chave" in ssh-*) ;; *) echo "a chave não parece uma chave pública SSH"; exit 1 ;; esac

mudou=0
feito() { echo "  mudou: $*"; mudou=$((mudou + 1)); }

# cron para o backup diário, curl para a verificação do deploy; o resto é o
# que o modo rootless precisa (pacotes do Ubuntu, como os do studygo).
faltam=()
for p in docker.io docker-compose-v2 rootlesskit uidmap slirp4netns dbus-user-session cron curl; do
	dpkg -s "$p" >/dev/null 2>&1 || faltam+=("$p")
done
if [ ${#faltam[@]} -gt 0 ]; then
	DEBIAN_FRONTEND=noninteractive apt-get update -qq
	DEBIAN_FRONTEND=noninteractive apt-get install -y -qq "${faltam[@]}" >/dev/null
	feito "instalados: ${faltam[*]}"
fi
systemctl is-active --quiet cron || { systemctl enable --now cron; feito "cron ligado"; }

# Sem sudo: o deploy só fala com o próprio Docker e escreve no próprio home.
if ! id "$usuario" >/dev/null 2>&1; then
	useradd -m -s /bin/bash "$usuario"
	feito "usuário $usuario criado"
fi
home=$(getent passwd "$usuario" | cut -d: -f6)
uid=$(id -u "$usuario")

install -d -m 700 -o "$usuario" -g "$usuario" "$home/.ssh"
if [ "$(cat "$home/.ssh/authorized_keys" 2>/dev/null)" != "$chave" ]; then
	printf '%s\n' "$chave" >"$home/.ssh/authorized_keys"
	chown "$usuario:$usuario" "$home/.ssh/authorized_keys"
	chmod 600 "$home/.ssh/authorized_keys"
	feito "chave de deploy de $usuario"
fi

# O rootless mapeia os usuários dos containers para esta faixa. O useradd do
# Ubuntu já a reserva; sem ela, o daemon nem sobe.
grep -q "^$usuario:" /etc/subuid || { echo "falta a faixa de $usuario em /etc/subuid"; exit 1; }

# Mantém o daemon do usuário de pé sem sessão aberta — senão a stack morre no
# primeiro logout.
if [ ! -e "/var/lib/systemd/linger/$usuario" ]; then
	loginctl enable-linger "$usuario"
	feito "linger de $usuario"
fi
systemctl start "user@$uid.service"

como_usuario() {
	sudo -u "$usuario" env XDG_RUNTIME_DIR="/run/user/$uid" \
		DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$uid/bus" \
		PATH="/usr/share/docker.io/contrib:/usr/bin:/bin" "$@"
}

# --skip-iptables: a rede do daemon é a do slirp4netns. O instalador termina
# chamando um `docker version` num caminho que o pacote do Ubuntu não tem — o
# serviço já está criado quando isso falha (o mesmo que o Ansible do studygo
# contorna).
if [ ! -f "$home/.config/systemd/user/docker.service" ]; then
	(cd "$home" && como_usuario dockerd-rootless-setuptool.sh install --skip-iptables >/dev/null 2>&1) || true
	[ -f "$home/.config/systemd/user/docker.service" ] || { echo "o instalador do rootless não criou o serviço"; exit 1; }
	feito "Docker rootless de $usuario instalado"
fi
if ! como_usuario systemctl --user is-active --quiet docker; then
	como_usuario systemctl --user enable --now docker
	feito "Docker rootless de $usuario ligado"
fi

# O contexto padrão vale para o SSH não interativo e para o cron, que não leem
# o .bashrc — sem ele, `docker` procuraria o daemon de sistema, desligado.
if ! (cd "$home" && como_usuario docker context inspect rootless >/dev/null 2>&1); then
	(cd "$home" && como_usuario docker context create rootless \
		--docker "host=unix:///run/user/$uid/docker.sock" >/dev/null)
	feito "contexto docker rootless"
fi
if [ "$(cd "$home" && como_usuario docker context show)" != rootless ]; then
	(cd "$home" && como_usuario docker context use rootless >/dev/null)
	feito "contexto rootless como padrão"
fi

(cd "$home" && como_usuario docker info --format '  docker {{.ServerVersion}} rootless: {{.SecurityOptions}}')

if [ "$mudou" = 0 ]; then
	echo "nada a mudar: o servidor já está como este script descreve"
else
	echo "$mudou mudança(s). Próximo passo: criar ~/agendago/.env e rodar scripts/implantar.sh"
fi
