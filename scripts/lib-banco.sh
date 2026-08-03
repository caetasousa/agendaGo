#!/usr/bin/env bash
# Verificações de sanidade de um banco recém-restaurado, compartilhadas por
# testar-restore.sh (que restaura num container descartável) e restaurar.sh (que
# restaura o banco de produção).
#
# Não é executável nem se conecta a nada: quem chama define a função
# `consultar`, porque os dois scripts falam com bancos diferentes — um por
# `docker exec` num container próprio, outro por `docker compose exec`.

# validar_banco confere que o restore produziu um banco utilizável: as tabelas
# do domínio existem e o histórico do Flyway veio junto. Sem checar o histórico,
# um dump de um banco sem migrations aplicadas passaria por bom.
#
# Imprime o resumo e devolve status != 0 quando algo falta.
validar_banco() {
	local esperadas="providers clients appointments date_exceptions sessions admins"
	local faltando="" tabela existe

	for tabela in $esperadas; do
		existe=$(consultar "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='$tabela'")
		[ "$existe" = "1" ] || faltando="$faltando $tabela"
	done
	if [ -n "$faltando" ]; then
		echo "ERRO: tabelas ausentes após o restore:$faltando" >&2
		return 1
	fi

	local migrations
	migrations=$(consultar "SELECT count(*) FROM flyway_schema_history WHERE success" 2>/dev/null || echo 0)
	if [ "$migrations" -lt 1 ]; then
		echo "ERRO: nenhuma migration bem sucedida no histórico do Flyway" >&2
		return 1
	fi

	local prestadores agendamentos
	prestadores=$(consultar "SELECT count(*) FROM providers")
	agendamentos=$(consultar "SELECT count(*) FROM appointments")

	printf 'restore íntegro — %s migrations, %s prestador(es), %s agendamento(s)\n' \
		"$migrations" "$prestadores" "$agendamentos"
}

# versao_de_schema devolve a última migration aplicada com sucesso.
versao_de_schema() {
	consultar "SELECT version FROM flyway_schema_history WHERE success ORDER BY installed_rank DESC LIMIT 1" |
		tr -d '[:space:]'
}

# campo_manifesto lê um campo do manifesto .json de um backup. sed em vez de jq
# porque a VPS não tem jq, e o JSON é gerado por backup.sh — formato fixo.
campo_manifesto() {
	[ -f "$1" ] || return 0
	sed -n "s/.*\"$2\": *\"\([^\"]*\)\".*/\1/p" "$1"
}
