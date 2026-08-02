<script lang="ts">
	import { page } from '$app/state';
	import type { PageData } from './$types';
	import type { Agendamento } from '$lib/api/appointments';
	import { chaveData } from '$lib/holidays';
	import Icone from '$lib/components/Icone.svelte';
	import LinhaAgendamento from '$lib/components/LinhaAgendamento.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Indicadores from '$lib/components/Indicadores.svelte';

	let { data }: { data: PageData } = $props();

	const ehPrestador = $derived(data.usuario.tipo === 'provider');
	// O link público aponta para a AGENDA, não para a conta de quem está logado.
	// Os dois ids coincidem nos prestadores anteriores à separação (a migração
	// reusou o UUID), mas divergem em toda conta criada depois — usar o da
	// conta aqui levaria a um link para um prestador que não existe.
	const linkAgendamento = $derived(`${page.url.origin}/agendar/${data.usuario.provider?.slug ?? ''}`);

	// Só o primeiro nome na saudação: o nome completo vive na sidebar, e é o que
	// fazia o título quebrar em duas linhas quando era longo.
	const primeiroNome = $derived(data.usuario.nome.trim().split(/\s+/)[0]);

	let copiado = $state(false);

	async function copiarLink() {
		try {
			await navigator.clipboard.writeText(linkAgendamento);
			copiado = true;
			setTimeout(() => (copiado = false), 2000);
		} catch {
			// clipboard indisponível (ex: contexto sem HTTPS): o link continua visível para cópia manual
		}
	}

	const chaveHoje = chaveData(new Date());

	// Ordena por data e hora de início — a agenda é lida no tempo, não na ordem
	// em que a API devolveu.
	function porQuandoComeca(a: Agendamento, b: Agendamento): number {
		return a.data === b.data ? a.inicioMinutos - b.inicioMinutos : a.data.localeCompare(b.data);
	}

	const pendentes = $derived(
		data.agendamentos.filter((a) => a.status === 'SOLICITADO').sort(porQuandoComeca)
	);
	const proximos = $derived(
		data.agendamentos
			.filter((a) => a.status === 'CONFIRMADO' && a.data >= chaveHoje)
			.sort(porQuandoComeca)
	);
	const realizadosNoMes = $derived(
		data.agendamentos.filter(
			(a) => a.status === 'REALIZADO' && a.data.slice(0, 7) === chaveHoje.slice(0, 7)
		)
	);

	// Mesma faixa de três colunas para os dois perfis: só o rótulo do primeiro
	// muda, porque quem aguarda é diferente — o prestador precisa agir, o cliente
	// está esperando a resposta dele.
	const indicadores = $derived([
		{
			rotulo: ehPrestador ? 'Aguardando' : 'Solicitados',
			valor: pendentes.length,
			destaque: ehPrestador && pendentes.length > 0
		},
		{ rotulo: 'Confirmados', valor: proximos.length },
		{ rotulo: 'Realizados no mês', valor: realizadosNoMes.length }
	]);

	// O prestador precisa confirmar; o cliente está do outro lado da mesma espera.
	const tituloPendentes = $derived(
		ehPrestador ? 'Aguardando sua confirmação' : 'Aguardando confirmação do prestador'
	);

	const semNada = $derived(pendentes.length === 0 && proximos.length === 0);
</script>

{#snippet secao(titulo: string, itens: Agendamento[], destaque = false)}
	<section class="mt-4">
		<div class="mb-2 flex items-baseline justify-between gap-3">
			<h2 class="text-sm font-semibold text-ink">{titulo}</h2>
			<a
				href="/painel/agendamentos"
				class="text-xs text-mute transition hover:text-ink"
			>
				Ver todos
			</a>
		</div>
		<ul
			class="divide-y divide-hairline overflow-hidden rounded-xl border bg-surface-card {destaque
				? 'border-accent-yellow/40'
				: 'border-hairline-strong'}"
		>
			{#each itens.slice(0, 5) as a (a.id)}
				<LinhaAgendamento agendamento={a} {ehPrestador} mostrarData />
			{/each}
		</ul>
	</section>
{/snippet}

<PageHeader titulo="Olá, {primeiroNome}" descricao={data.usuario.email} />

<Indicadores itens={indicadores} />

{#if pendentes.length > 0}
	{@render secao(tituloPendentes, pendentes, ehPrestador)}
{/if}

{#if proximos.length > 0}
	{@render secao('Próximos atendimentos', proximos)}
{/if}

{#if semNada}
	<div class="mt-4 rounded-xl border border-hairline-strong bg-surface-card px-5 py-8 text-center">
		<p class="text-sm text-body">
			{#if ehPrestador}
				Nenhum atendimento na fila. Compartilhe seu link para começar a receber pedidos.
			{:else}
				Você ainda não tem agendamentos.
			{/if}
		</p>
		{#if !ehPrestador}
			<a
				href="/painel/agendar"
				class="mt-4 inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90"
			>
				Agendar agora
			</a>
		{/if}
	</div>
{/if}

{#if ehPrestador}
	<div
		class="mt-4 flex flex-wrap items-center gap-x-4 gap-y-3 rounded-xl border border-hairline-strong bg-surface-card p-4"
	>
		<div class="min-w-0 flex-1">
			<p class="text-xs font-medium tracking-wide text-mute uppercase">Seu link de agendamento</p>
			<p class="mt-1 truncate font-mono text-sm text-ink" data-link-agendamento>
				{linkAgendamento}
			</p>
			<p class="mt-1 text-xs text-mute">
				Compartilhe no Instagram ou WhatsApp — quem abrir vê seus horários livres e agenda.
			</p>
		</div>
		<button
			type="button"
			onclick={copiarLink}
			class="inline-flex h-9 shrink-0 items-center gap-2 rounded-md border border-hairline-strong px-4 text-sm font-medium text-ink transition hover:bg-surface-elevated"
		>
			<Icone nome={copiado ? 'check' : 'copiar'} tamanho={15} />
			{copiado ? 'Copiado' : 'Copiar'}
		</button>
	</div>
{/if}
