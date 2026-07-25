<!-- Linha de um agendamento, usada tanto na home do painel quanto na tela de
     Agendamentos, para cliente e prestador. Sem borda própria: quem delimita o
     grupo é a superfície da lista, e as linhas se separam por divisória.
     A faixa colorida à esquerda do horário codifica o estado. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { Agendamento } from '$lib/api/appointments';
	import { dataLonga, minutosParaHHMM, rotuloStatus } from '$lib/format';

	let {
		agendamento,
		ehPrestador,
		mostrarData = false,
		extra,
		acoes
	}: {
		agendamento: Agendamento;
		ehPrestador: boolean;
		mostrarData?: boolean;
		/** Conteúdo auxiliar entre os dados e as ações (ex.: a mensagem pronta
		 *  para enviar ao cliente, nas marcações feitas pelo prestador). */
		extra?: Snippet;
		acoes?: Snippet;
	} = $props();

	const rotulo = $derived(rotuloStatus(agendamento.status));

	// A faixa repete a cor do marcador de status, para o estado ser legível de
	// relance sem depender de ler a pílula.
	const corDaFaixa = $derived.by(() => {
		switch (agendamento.status) {
			case 'SOLICITADO':
				return 'bg-accent-yellow';
			case 'CONFIRMADO':
				return 'bg-accent-green';
			case 'REALIZADO':
				return 'bg-accent-blue';
			case 'NAO_COMPARECEU':
				return 'bg-accent-orange';
			default:
				return 'bg-hairline-strong';
		}
	});
</script>

<li
	data-agendamento={agendamento.id}
	data-status={agendamento.status}
	class="flex gap-3 px-4 py-3"
>
	<div class="w-14 shrink-0 pt-0.5">
		<div class="text-sm font-medium text-ink tabular-nums">
			{minutosParaHHMM(agendamento.inicioMinutos)}
		</div>
		<div class="text-xs text-mute tabular-nums">{minutosParaHHMM(agendamento.fimMinutos)}</div>
	</div>

	<div class="w-[3px] shrink-0 rounded-full {corDaFaixa}" aria-hidden="true"></div>

	<div class="min-w-0 flex-1">
		<div class="flex flex-wrap items-start justify-between gap-x-3 gap-y-1.5">
			<div class="min-w-0">
				{#if mostrarData}
					<p class="text-xs text-mute">{dataLonga(agendamento.data)}</p>
				{/if}
				<p class="text-sm text-ink">
					{ehPrestador ? agendamento.nomeCliente : agendamento.nomePrestador}
				</p>
				{#if ehPrestador && (agendamento.telefoneCliente || agendamento.emailCliente)}
					<p class="mt-0.5 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-mute">
						{#if agendamento.telefoneCliente}
							<a href="tel:{agendamento.telefoneCliente}" class="transition hover:text-ink">
								{agendamento.telefoneCliente}
							</a>
						{/if}
						{#if agendamento.emailCliente}
							<a href="mailto:{agendamento.emailCliente}" class="transition hover:text-ink">
								{agendamento.emailCliente}
							</a>
						{/if}
					</p>
				{/if}
				{#if agendamento.observacao}
					<p class="mt-0.5 text-xs text-mute">{agendamento.observacao}</p>
				{/if}
			</div>

			<span
				class="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-hairline-strong bg-surface-elevated px-2.5 py-0.5 text-xs text-body"
			>
				<span class="h-1.5 w-1.5 rounded-full {rotulo.cor}"></span>
				{rotulo.texto}
			</span>
		</div>

		{#if extra}
			{@render extra()}
		{/if}

		{#if acoes}
			{@render acoes()}
		{/if}
	</div>
</li>
