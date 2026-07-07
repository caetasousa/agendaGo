<script lang="ts">
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import { atualizarPreferencias } from '$lib/api/preferences';
	import { sessao } from '$lib/stores/session.svelte';

	let { data }: { data: PageData } = $props();

	// Estado local editável inicializado a partir do load — intencional: o
	// usuário edita os campos independentemente do valor original do servidor.
	// svelte-ignore state_referenced_locally
	let aceitaAgendamentos = $state(data.aceitaAgendamentos);
	// svelte-ignore state_referenced_locally
	let descansoMinutos = $state(data.descansoMinutos);

	let enviando = $state(false);
	let erro = $state<string | null>(null);
	let sucesso = $state(false);

	const descansoInvalido = $derived(descansoMinutos < 0);

	async function enviar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;
		sucesso = false;
		enviando = true;

		try {
			const salvo = await atualizarPreferencias({ aceitaAgendamentos, descansoMinutos });
			aceitaAgendamentos = salvo.aceitaAgendamentos;
			descansoMinutos = salvo.descansoMinutos;
			if (sessao.usuario) {
				sessao.definir({ ...sessao.usuario, ...salvo });
			}
			sucesso = true;
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível salvar as preferências.';
		} finally {
			enviando = false;
		}
	}

	const inputClasse =
		'mt-2 h-10 w-full rounded-md border border-hairline-strong bg-surface-card px-3.5 text-sm text-ink outline-none transition placeholder:text-mute focus:border-ink';
</script>

<div class="mx-auto max-w-xl">
	<a href="/painel" class="text-sm text-mute transition hover:text-ink">← Voltar ao painel</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Preferências</h1>
	<p class="mt-3 text-body">Configure como você recebe agendamentos.</p>

	<div class="mt-8 rounded-lg border border-hairline-strong bg-surface-card p-8">
		<form class="space-y-5" novalidate onsubmit={enviar}>
			{#if erro}
				<div
					class="flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
					<span class="text-body">{erro}</span>
				</div>
			{/if}

			{#if sucesso}
				<div
					class="flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-green"></span>
					<span class="text-body">Preferências salvas.</span>
				</div>
			{/if}

			<label
				for="aceita-agendamentos"
				class="flex cursor-pointer items-center justify-between gap-4 rounded-md border border-hairline-strong bg-surface-elevated p-3.5"
			>
				<span class="text-sm font-medium text-ink">Aceitar agendamentos</span>
				<input
					id="aceita-agendamentos"
					type="checkbox"
					bind:checked={aceitaAgendamentos}
					class="h-5 w-5 rounded border-hairline-strong accent-primary"
				/>
			</label>

			<div>
				<label for="descanso-minutos" class="block text-sm font-medium text-ink">
					Descanso entre atendimentos (minutos)
				</label>
				<input
					id="descanso-minutos"
					type="number"
					min="0"
					step="1"
					bind:value={descansoMinutos}
					required
					aria-invalid={descansoInvalido}
					class={inputClasse}
				/>
				{#if descansoInvalido}
					<p class="mt-1.5 text-sm text-accent-red">O descanso não pode ser negativo.</p>
				{/if}
			</div>

			<button
				type="submit"
				disabled={enviando || descansoInvalido}
				class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{enviando ? 'Salvando…' : 'Salvar'}
			</button>
		</form>
	</div>
</div>
