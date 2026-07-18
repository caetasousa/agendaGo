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
	let telefone = $state(data.telefone);
	// svelte-ignore state_referenced_locally
	let aceitaAgendamentos = $state(data.aceitaAgendamentos);
	// svelte-ignore state_referenced_locally
	let descansoMinutos = $state(data.descansoMinutos);
	// svelte-ignore state_referenced_locally
	let duracaoAtendimentoMinutos = $state(data.duracaoAtendimentoMinutos);
	// svelte-ignore state_referenced_locally
	let horariosPadrao = $state<Bloco[]>(data.horariosPadrao.map((b) => ({ ...b })));
	// svelte-ignore state_referenced_locally
	let permiteMarcacaoPeloPrestador = $state(data.permiteMarcacaoPeloPrestador);

	let enviando = $state(false);
	let erro = $state<string | null>(null);
	let sucesso = $state(false);

	const descansoInvalido = $derived(descansoMinutos < 0);
	const duracaoInvalida = $derived(duracaoAtendimentoMinutos < 15 || duracaoAtendimentoMinutos > 1440);

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
		sucesso = false;
	}

	function removerBloco(index: number) {
		horariosPadrao = horariosPadrao.filter((_, i) => i !== index);
		sucesso = false;
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
			const salvo = await atualizarPreferencias({
				telefone,
				aceitaAgendamentos,
				descansoMinutos,
				duracaoAtendimentoMinutos,
				horariosPadrao,
				permiteMarcacaoPeloPrestador
			});
			telefone = salvo.telefone;
			aceitaAgendamentos = salvo.aceitaAgendamentos;
			descansoMinutos = salvo.descansoMinutos;
			duracaoAtendimentoMinutos = salvo.duracaoAtendimentoMinutos;
			horariosPadrao = salvo.horariosPadrao;
			permiteMarcacaoPeloPrestador = salvo.permiteMarcacaoPeloPrestador;
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

	// numeroClasse é o estilo dos campos numéricos (duração/descanso).
	const numeroClasse =
		'h-10 w-full rounded-md border border-hairline-strong bg-surface-card px-3.5 text-sm text-ink outline-none transition focus:border-ink';
</script>

<div class="mx-auto max-w-xl pb-24">
	<a href="/painel" class="text-sm text-mute transition hover:text-ink">← Voltar ao painel</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Preferências</h1>
	<p class="mt-3 text-body">Configure como você recebe agendamentos.</p>

	<form class="mt-8 space-y-6" novalidate onsubmit={enviar}>
		<!-- Aceitar agendamentos: toggle em destaque -->
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
			<label for="aceita-agendamentos" class="flex cursor-pointer items-start justify-between gap-4">
				<span class="min-w-0">
					<span class="block text-sm font-semibold text-ink">Aceitar agendamentos</span>
					<span class="mt-1 block text-sm text-body">
						Quando ativo, seus horários livres aparecem para os clientes agendarem.
					</span>
				</span>
				<input
					id="aceita-agendamentos"
					type="checkbox"
					bind:checked={aceitaAgendamentos}
					onchange={() => (sucesso = false)}
					class="peer sr-only"
				/>
				<span
					class="relative mt-0.5 h-6 w-11 shrink-0 rounded-full border border-hairline-strong bg-surface-elevated transition-colors peer-checked:border-accent-green peer-checked:bg-accent-green/30 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-link after:absolute after:top-0.5 after:left-0.5 after:h-5 after:w-5 after:rounded-full after:bg-mute after:transition-transform after:content-[''] peer-checked:after:translate-x-5 peer-checked:after:bg-accent-green"
				></span>
			</label>
		</div>

		<!-- Contato -->
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
			<h2 class="text-sm font-semibold text-ink">Contato</h2>
			<div class="mt-4">
				<label for="telefone" class="block text-sm font-medium text-ink">Telefone</label>
				<input
					id="telefone"
					type="tel"
					bind:value={telefone}
					onchange={() => (sucesso = false)}
					required
					minlength="8"
					placeholder="(11) 99999-8888"
					class="mt-2 h-10 w-full rounded-md border border-hairline-strong bg-surface-card px-3.5 text-sm text-ink outline-none transition placeholder:text-mute focus:border-ink"
				/>
				<p class="mt-1.5 text-xs text-mute">Como os clientes entram em contato com você.</p>
			</div>
		</div>

		<!-- Marcar para um cliente: toggle -->
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
			<label
				for="permite-marcacao-pelo-prestador"
				class="flex cursor-pointer items-start justify-between gap-4"
			>
				<span class="min-w-0">
					<span class="block text-sm font-semibold text-ink">Marcar para um cliente</span>
					<span class="mt-1 block text-sm text-body">
						Quando ativo, você pode registrar você mesmo um agendamento na sua agenda (ex.:
						cliente que ligou).
					</span>
				</span>
				<input
					id="permite-marcacao-pelo-prestador"
					type="checkbox"
					bind:checked={permiteMarcacaoPeloPrestador}
					onchange={() => (sucesso = false)}
					class="peer sr-only"
				/>
				<span
					class="relative mt-0.5 h-6 w-11 shrink-0 rounded-full border border-hairline-strong bg-surface-elevated transition-colors peer-checked:border-accent-green peer-checked:bg-accent-green/30 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-link after:absolute after:top-0.5 after:left-0.5 after:h-5 after:w-5 after:rounded-full after:bg-mute after:transition-transform after:content-[''] peer-checked:after:translate-x-5 peer-checked:after:bg-accent-green"
				></span>
			</label>
		</div>

		<!-- Duração e descanso -->
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
			<h2 class="text-sm font-semibold text-ink">Atendimento</h2>
			<div class="mt-4 grid gap-5 sm:grid-cols-2">
				<div>
					<label for="duracao-atendimento" class="block text-sm font-medium text-ink">Duração</label>
					<div class="mt-2 flex items-center gap-2">
						<input
							id="duracao-atendimento"
							type="number"
							min="15"
							max="1440"
							step="15"
							bind:value={duracaoAtendimentoMinutos}
							onchange={() => (sucesso = false)}
							required
							aria-invalid={duracaoInvalida}
							class={numeroClasse}
						/>
						<span class="shrink-0 text-sm text-mute">min</span>
					</div>
					<p class="mt-1.5 text-xs text-mute">Tamanho de cada horário ofertado.</p>
					{#if duracaoInvalida}
						<p class="mt-1.5 text-xs text-accent-red">Entre 15 minutos e um dia.</p>
					{/if}
				</div>

				<div>
					<label for="descanso-minutos" class="block text-sm font-medium text-ink">Descanso</label>
					<div class="mt-2 flex items-center gap-2">
						<input
							id="descanso-minutos"
							type="number"
							min="0"
							step="5"
							bind:value={descansoMinutos}
							onchange={() => (sucesso = false)}
							required
							aria-invalid={descansoInvalido}
							class={numeroClasse}
						/>
						<span class="shrink-0 text-sm text-mute">min</span>
					</div>
					<p class="mt-1.5 text-xs text-mute">Intervalo entre um atendimento e o próximo.</p>
					{#if descansoInvalido}
						<p class="mt-1.5 text-xs text-accent-red">Não pode ser negativo.</p>
					{/if}
				</div>
			</div>
		</div>

		<!-- Expediente padrão -->
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
			<h2 class="text-sm font-semibold text-ink">Expediente padrão</h2>
			<p class="mt-1 text-sm text-body">
				Períodos em que você atende de segunda a sexta — a base do seu calendário.
			</p>

			<div class="mt-4 space-y-3">
				{#each horariosPadrao as bloco, index (index)}
					<div class="rounded-lg border border-hairline bg-surface-elevated p-4">
						<div class="flex items-center justify-between">
							<span class="text-xs font-semibold tracking-wide text-mute uppercase">
								Período {index + 1}
							</span>
							<button
								type="button"
								onclick={() => removerBloco(index)}
								class="text-xs font-medium text-mute transition hover:text-accent-red"
							>
								Remover
							</button>
						</div>

						<div class="mt-3 grid grid-cols-[1fr_auto_1fr] items-end gap-2">
							<div>
								<label for="expediente-inicio-{index}" class="mb-1.5 block text-xs text-mute">Início</label>
								<TimeSelect
									id="expediente-inicio-{index}"
									bind:valor={horariosPadrao[index].inicioMinutos}
								/>
							</div>
							<span class="pb-2.5 text-mute">–</span>
							<div>
								<label for="expediente-fim-{index}" class="mb-1.5 block text-xs text-mute">Fim</label>
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
				class="mt-3 flex h-10 w-full items-center justify-center rounded-lg border border-dashed border-hairline-strong text-sm font-medium text-ink transition hover:border-ink/40 hover:bg-surface-elevated"
			>
				+ Adicionar período
			</button>

			{#if horariosPadrao.length === 0}
				<p class="mt-3 text-sm text-mute">
					Nenhum período — os dias úteis ficam indisponíveis até você adicionar um.
				</p>
			{:else}
				<p class="mt-4 flex items-center gap-1.5 text-xs text-mute">
					<span class="h-1.5 w-1.5 rounded-full bg-accent-green"></span>
					Nos dias úteis você atende: {resumoExpediente}.
				</p>
			{/if}
		</div>

		{#if erro}
			<div
				class="flex items-start gap-2 rounded-md border border-accent-red/40 bg-accent-red/10 p-3 text-sm"
			>
				<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
				<span class="text-body">{erro}</span>
			</div>
		{/if}

		<!-- Barra de ação fixa: botão sempre à vista, com o feedback ao lado -->
		<div
			class="sticky bottom-0 -mx-6 flex items-center gap-3 border-t border-hairline bg-canvas/90 px-6 py-4 backdrop-blur"
		>
			<button
				type="submit"
				disabled={enviando || descansoInvalido || duracaoInvalida}
				class="inline-flex h-10 items-center rounded-md bg-primary px-5 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{enviando ? 'Salvando…' : 'Salvar alterações'}
			</button>

			{#if sucesso}
				<span class="flex items-center gap-1.5 text-sm font-medium text-accent-green">
					<svg
						width="16"
						height="16"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2.5"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"
					>
						<path d="M20 6 9 17l-5-5" />
					</svg>
					Salvo
				</span>
			{/if}
		</div>
	</form>
</div>
