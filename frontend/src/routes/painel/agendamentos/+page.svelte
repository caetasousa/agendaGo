<script lang="ts">
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import {
		cancelarAgendamento,
		confirmarAgendamento,
		listarAgendamentosDoCliente,
		listarAgendamentosDoPrestador,
		marcarNaoCompareceu,
		marcarRealizado,
		recusarAgendamento,
		type Agendamento
	} from '$lib/api/appointments';
	import { chaveData } from '$lib/holidays';
	import { dataLonga, minutosParaHHMM, rotuloStatus } from '$lib/format';

	let { data }: { data: PageData } = $props();

	// svelte-ignore state_referenced_locally
	let agendamentos = $state<Agendamento[]>(data.agendamentos);
	let agindo = $state<string | null>(null);
	let erro = $state<string | null>(null);

	// O tipo do usuário não muda dentro da sessão da página — capturar o valor
	// inicial é intencional.
	// svelte-ignore state_referenced_locally
	const ehPrestador = data.tipo === 'provider';
	const chaveHoje = chaveData(new Date());

	const pendentes = $derived(agendamentos.filter((a) => a.status === 'SOLICITADO'));
	const confirmados = $derived(agendamentos.filter((a) => a.status === 'CONFIRMADO'));
	const historico = $derived(
		agendamentos.filter((a) => a.status !== 'SOLICITADO' && a.status !== 'CONFIRMADO')
	);

	// jaComecou marca confirmados cujo horário já chegou — habilita o
	// desfecho (realizado / não compareceu) para o prestador.
	function jaComecou(a: Agendamento): boolean {
		if (a.data < chaveHoje) return true;
		if (a.data > chaveHoje) return false;
		const agora = new Date();
		return a.inicioMinutos <= agora.getHours() * 60 + agora.getMinutes();
	}

	async function executar(id: string, acao: (id: string) => Promise<void>) {
		if (agindo) return;
		erro = null;
		agindo = id;
		try {
			await acao(id);
			const resposta = ehPrestador
				? await listarAgendamentosDoPrestador()
				: await listarAgendamentosDoCliente();
			agendamentos = resposta.agendamentos;
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível concluir a ação.';
		} finally {
			agindo = null;
		}
	}

	const botaoBase =
		'inline-flex h-8 items-center rounded-md px-3 text-xs font-medium transition disabled:cursor-not-allowed disabled:opacity-60';
	const botaoPrimario = `${botaoBase} bg-primary text-primary-on hover:opacity-90`;
	const botaoContorno = `${botaoBase} border border-hairline-strong text-ink hover:bg-surface-elevated`;
	const botaoPerigo = `${botaoBase} border border-hairline-strong text-accent-red hover:bg-surface-elevated`;
</script>

{#snippet cartao(a: Agendamento)}
	{@const rotulo = rotuloStatus(a.status)}
	<li
		data-agendamento={a.id}
		data-status={a.status}
		class="rounded-md border border-hairline-strong p-4"
	>
		<div class="flex flex-wrap items-center justify-between gap-2">
			<div>
				<p class="text-sm font-medium text-ink">
					{dataLonga(a.data)} · {minutosParaHHMM(a.inicioMinutos)}–{minutosParaHHMM(a.fimMinutos)}
				</p>
				<p class="mt-0.5 text-sm text-body">
					{ehPrestador ? a.nomeCliente : a.nomePrestador}
				</p>
				{#if ehPrestador && (a.telefoneCliente || a.emailCliente)}
					<p class="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-mute">
						{#if a.telefoneCliente}
							<a href="tel:{a.telefoneCliente}" class="transition hover:text-ink">{a.telefoneCliente}</a>
						{/if}
						{#if a.emailCliente}
							<a href="mailto:{a.emailCliente}" class="transition hover:text-ink">{a.emailCliente}</a>
						{/if}
					</p>
				{/if}
			</div>
			<span
				class="inline-flex items-center gap-1.5 rounded-full border border-hairline-strong bg-surface-elevated px-2.5 py-0.5 text-xs text-body"
			>
				<span class="h-1.5 w-1.5 rounded-full {rotulo.cor}"></span>
				{rotulo.texto}
			</span>
		</div>

		<div class="mt-3 flex flex-wrap gap-2 empty:hidden">
			{#if ehPrestador && a.status === 'SOLICITADO'}
				<button
					type="button"
					disabled={agindo !== null}
					onclick={() => executar(a.id, confirmarAgendamento)}
					class={botaoPrimario}
				>
					Confirmar
				</button>
				<button
					type="button"
					disabled={agindo !== null}
					onclick={() => executar(a.id, recusarAgendamento)}
					class={botaoPerigo}
				>
					Recusar
				</button>
			{/if}

			{#if !ehPrestador && a.status === 'SOLICITADO'}
				<button
					type="button"
					disabled={agindo !== null}
					onclick={() => executar(a.id, cancelarAgendamento)}
					class={botaoPerigo}
				>
					Cancelar solicitação
				</button>
			{/if}

			{#if a.status === 'CONFIRMADO' && !jaComecou(a)}
				<button
					type="button"
					disabled={agindo !== null}
					onclick={() => executar(a.id, cancelarAgendamento)}
					class={botaoPerigo}
				>
					Cancelar
				</button>
			{/if}

			{#if ehPrestador && a.status === 'CONFIRMADO' && jaComecou(a)}
				<button
					type="button"
					disabled={agindo !== null}
					onclick={() => executar(a.id, marcarRealizado)}
					class={botaoPrimario}
				>
					Realizado
				</button>
				<button
					type="button"
					disabled={agindo !== null}
					onclick={() => executar(a.id, marcarNaoCompareceu)}
					class={botaoContorno}
				>
					Não compareceu
				</button>
			{/if}
		</div>
	</li>
{/snippet}

<div class="mx-auto max-w-2xl">
	<a href="/painel" class="text-sm text-mute transition hover:text-ink">← Voltar ao painel</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Agendamentos</h1>
	<p class="mt-3 text-body">
		{ehPrestador
			? 'Solicitações recebidas e atendimentos confirmados — você decide o que entra na agenda.'
			: 'Acompanhe suas solicitações e atendimentos confirmados.'}
	</p>

	{#if erro}
		<div
			class="mt-6 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
		>
			<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
			<span class="text-body">{erro}</span>
		</div>
	{/if}

	{#if agendamentos.length === 0}
		<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-8">
			<p class="text-sm text-body">
				{ehPrestador
					? 'Nenhum agendamento recebido ainda. Ative a agenda em Preferências para aparecer aos clientes.'
					: 'Você ainda não tem agendamentos.'}
				{#if !ehPrestador}
					<a href="/painel/agendar" class="font-medium text-ink underline">Agendar agora</a>
				{/if}
			</p>
		</div>
	{/if}

	{#if pendentes.length > 0}
		<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-5 sm:p-8">
			<h2 class="text-lg font-semibold text-ink">Aguardando confirmação</h2>
			<ul class="mt-4 space-y-3">
				{#each pendentes as a (a.id)}
					{@render cartao(a)}
				{/each}
			</ul>
		</div>
	{/if}

	{#if confirmados.length > 0}
		<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-5 sm:p-8">
			<h2 class="text-lg font-semibold text-ink">Confirmados</h2>
			<ul class="mt-4 space-y-3">
				{#each confirmados as a (a.id)}
					{@render cartao(a)}
				{/each}
			</ul>
		</div>
	{/if}

	{#if historico.length > 0}
		<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-5 sm:p-8">
			<h2 class="text-lg font-semibold text-ink">Histórico</h2>
			<ul class="mt-4 space-y-3">
				{#each historico as a (a.id)}
					{@render cartao(a)}
				{/each}
			</ul>
		</div>
	{/if}
</div>
