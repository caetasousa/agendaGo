<script lang="ts">
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import { atualizarPreferencias } from '$lib/api/preferences';
	import type { Bloco } from '$lib/api/availability';
	import TimeSelect from '$lib/components/TimeSelect.svelte';
	import { sessao } from '$lib/stores/session.svelte';

	let { data }: { data: PageData } = $props();

	// Estado local editável inicializado a partir do load — intencional: o
	// usuário edita os campos independentemente do valor original do servidor.
	// svelte-ignore state_referenced_locally
	let aceitaAgendamentos = $state(data.aceitaAgendamentos);
	// svelte-ignore state_referenced_locally
	let descansoMinutos = $state(data.descansoMinutos);
	// svelte-ignore state_referenced_locally
	let horariosPadrao = $state<Bloco[]>(data.horariosPadrao.map((b) => ({ ...b })));

	let enviando = $state(false);
	let erro = $state<string | null>(null);
	let sucesso = $state(false);

	const descansoInvalido = $derived(descansoMinutos < 0);

	function minutosParaHHMM(minutos: number): string {
		const h = Math.floor(minutos / 60)
			.toString()
			.padStart(2, '0');
		const m = (minutos % 60).toString().padStart(2, '0');
		return `${h}:${m}`;
	}

	// O novo período parte do fim do último bloco + 1h, para o prestador só
	// ajustar em vez de digitar do zero.
	function adicionarBloco() {
		const ultimo = horariosPadrao[horariosPadrao.length - 1];
		const inicio = ultimo ? Math.min(ultimo.fimMinutos + 60, 22 * 60) : 8 * 60;
		horariosPadrao = [...horariosPadrao, { inicioMinutos: inicio, fimMinutos: Math.min(inicio + 120, 23 * 60) }];
	}

	function removerBloco(index: number) {
		horariosPadrao = horariosPadrao.filter((_, i) => i !== index);
	}

	const resumoExpediente = $derived(
		horariosPadrao
			.map((b) => `${minutosParaHHMM(b.inicioMinutos)}–${minutosParaHHMM(b.fimMinutos)}`)
			.join(', ')
	);

	async function enviar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;
		sucesso = false;
		enviando = true;

		try {
			const salvo = await atualizarPreferencias({ aceitaAgendamentos, descansoMinutos, horariosPadrao });
			aceitaAgendamentos = salvo.aceitaAgendamentos;
			descansoMinutos = salvo.descansoMinutos;
			horariosPadrao = salvo.horariosPadrao;
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

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-8">
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

			<div class="rounded-md border border-hairline-strong p-4">
				<p class="text-sm font-medium text-ink">Expediente padrão (dias úteis)</p>
				<p class="mt-1 text-sm text-body">
					Períodos em que você atende de segunda a sexta. Edite os horários abaixo, remova ou
					acrescente períodos — o calendário de disponibilidade usa isso como base.
				</p>

				<div class="mt-4 space-y-3">
					{#each horariosPadrao as bloco, index (index)}
						<div class="rounded-md border border-hairline bg-surface-elevated p-3">
							<div class="flex items-center justify-between">
								<span class="text-xs font-medium tracking-wide text-mute uppercase">
									Período {index + 1}
								</span>
								<button
									type="button"
									onclick={() => removerBloco(index)}
									class="text-sm text-mute transition hover:text-accent-red"
								>
									Remover
								</button>
							</div>

							<div class="mt-2 flex items-end gap-3">
								<div class="flex-1">
									<label for="expediente-inicio-{index}" class="mb-2 block text-xs text-mute">Início</label>
									<TimeSelect
										id="expediente-inicio-{index}"
										bind:valor={horariosPadrao[index].inicioMinutos}
									/>
								</div>
								<span class="pb-2.5 text-mute">–</span>
								<div class="flex-1">
									<label for="expediente-fim-{index}" class="mb-2 block text-xs text-mute">Fim</label>
									<TimeSelect
										id="expediente-fim-{index}"
										bind:valor={horariosPadrao[index].fimMinutos}
										minimo={15}
										maximo={24 * 60}
									/>
								</div>
							</div>
						</div>
					{/each}
				</div>

				<button
					type="button"
					onclick={adicionarBloco}
					class="mt-3 flex h-10 w-full items-center justify-center rounded-md border border-dashed border-hairline-strong text-sm font-medium text-ink transition hover:bg-surface-elevated"
				>
					+ Adicionar período
				</button>

				{#if horariosPadrao.length === 0}
					<p class="mt-3 text-sm text-mute">
						Nenhum período definido — os dias úteis ficam indisponíveis até você adicionar um.
					</p>
				{:else}
					<p class="mt-3 text-xs text-mute">Nos dias úteis você atende: {resumoExpediente}.</p>
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
