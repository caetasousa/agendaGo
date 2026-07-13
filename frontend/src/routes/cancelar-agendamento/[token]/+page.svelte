<script lang="ts">
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import { cancelarPorToken } from '$lib/api/appointments';
	import { dataLonga, minutosParaHHMM } from '$lib/format';

	let { data }: { data: PageData } = $props();

	// Detalhe carregado pelo load; não muda após a abertura. podeCancelar
	// reflete o estado no momento da abertura (status cancelável e antecedência
	// de 24h respeitada).
	// svelte-ignore state_referenced_locally
	const detalhe = data.detalhe;

	let enviando = $state(false);
	let erro = $state<string | null>(null);
	let cancelado = $state(false);

	async function cancelar() {
		erro = null;
		enviando = true;
		try {
			await cancelarPorToken(data.token);
			cancelado = true;
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível cancelar o agendamento.';
		} finally {
			enviando = false;
		}
	}
</script>

<div class="mx-auto max-w-xl">
	<a href="/" class="text-sm text-mute transition hover:text-ink">← Início</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Cancelar agendamento</h1>

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-8">
		{#if cancelado || detalhe.status === 'CANCELADO'}
			<p class="text-body">Seu agendamento foi cancelado. Você já pode fechar esta página.</p>
		{:else}
			<p class="text-body">Você está prestes a cancelar o seguinte agendamento:</p>

			<div class="mt-6 rounded-md border border-hairline bg-surface-elevated p-4">
				<p class="text-sm text-body">
					<span class="font-medium text-ink">{dataLonga(detalhe.data)}</span>, das
					<span class="font-medium text-ink">{minutosParaHHMM(detalhe.inicioMinutos)}</span>
					às
					<span class="font-medium text-ink">{minutosParaHHMM(detalhe.fimMinutos)}</span>
					com <span class="font-medium text-ink">{detalhe.nomePrestador}</span>
				</p>
			</div>

			{#if erro}
				<div
					class="mt-6 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
					<span class="text-body">{erro}</span>
				</div>
			{/if}

			{#if detalhe.podeCancelar}
				<button
					type="button"
					disabled={enviando}
					onclick={cancelar}
					class="mt-6 flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
				>
					{enviando ? 'Cancelando…' : 'Confirmar cancelamento'}
				</button>
			{:else}
				<div
					class="mt-6 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-yellow"></span>
					<span class="text-body"
						>Este agendamento não pode mais ser cancelado — o cancelamento exige pelo menos 24
						horas de antecedência. Entre em contato com o prestador.</span
					>
				</div>
			{/if}
		{/if}
	</div>
</div>
