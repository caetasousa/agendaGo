<script lang="ts">
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import {
		definirGradeSemanal,
		criarExcecao,
		removerExcecao,
		type Bloco,
		type Excecao
	} from '$lib/api/availability';

	let { data }: { data: PageData } = $props();

	const nomesDias = ['Domingo', 'Segunda', 'Terça', 'Quarta', 'Quinta', 'Sexta', 'Sábado'];

	function minutosParaHHMM(minutos: number): string {
		const h = Math.floor(minutos / 60)
			.toString()
			.padStart(2, '0');
		const m = (minutos % 60).toString().padStart(2, '0');
		return `${h}:${m}`;
	}

	function hhmmParaMinutos(hhmm: string): number {
		const [h, m] = hhmm.split(':').map(Number);
		return h * 60 + m;
	}

	// grade[dia] guarda os blocos daquele dia da semana (0=domingo..6=sábado),
	// inicializado a partir do load — editável independentemente depois.
	// svelte-ignore state_referenced_locally
	let grade = $state<Record<number, Bloco[]>>(
		Object.fromEntries(
			[0, 1, 2, 3, 4, 5, 6].map((d) => [d, data.dias.find((x) => x.diaSemana === d)?.blocos ?? []])
		)
	);

	let enviandoGrade = $state(false);
	let erroGrade = $state<string | null>(null);
	let sucessoGrade = $state(false);

	function adicionarBloco(dia: number) {
		grade[dia] = [...grade[dia], { inicioMinutos: 480, fimMinutos: 720 }];
	}

	function removerBloco(dia: number, index: number) {
		grade[dia] = grade[dia].filter((_, i) => i !== index);
	}

	async function salvarGrade(evento: SubmitEvent) {
		evento.preventDefault();
		erroGrade = null;
		sucessoGrade = false;
		enviandoGrade = true;

		try {
			const dias = Object.entries(grade)
				.filter(([, blocos]) => blocos.length > 0)
				.map(([diaSemana, blocos]) => ({ diaSemana: Number(diaSemana), blocos }));

			const salvo = await definirGradeSemanal(dias);
			grade = Object.fromEntries(
				[0, 1, 2, 3, 4, 5, 6].map((d) => [d, salvo.dias.find((x) => x.diaSemana === d)?.blocos ?? []])
			);
			sucessoGrade = true;
		} catch (e) {
			erroGrade = e instanceof ApiError ? e.message : 'Não foi possível salvar a grade.';
		} finally {
			enviandoGrade = false;
		}
	}

	// svelte-ignore state_referenced_locally
	let excecoes = $state<Excecao[]>(data.excecoes);
	let novaData = $state('');
	let novoTipo = $state<'bloqueio' | 'extra'>('bloqueio');
	let novoBlocoInicio = $state('08:00');
	let novoBlocoFim = $state('12:00');
	let erroExcecao = $state<string | null>(null);
	let enviandoExcecao = $state(false);

	async function adicionarExcecao(evento: SubmitEvent) {
		evento.preventDefault();
		erroExcecao = null;
		enviandoExcecao = true;

		try {
			const blocos: Bloco[] =
				novoTipo === 'extra'
					? [{ inicioMinutos: hhmmParaMinutos(novoBlocoInicio), fimMinutos: hhmmParaMinutos(novoBlocoFim) }]
					: [];
			const criada = await criarExcecao({ data: novaData, tipo: novoTipo, blocos });
			excecoes = [...excecoes, criada];
			novaData = '';
		} catch (e) {
			erroExcecao = e instanceof ApiError ? e.message : 'Não foi possível criar a exceção.';
		} finally {
			enviandoExcecao = false;
		}
	}

	async function excluirExcecao(id: string) {
		await removerExcecao(id);
		excecoes = excecoes.filter((e) => e.id !== id);
	}

	const inputClasse =
		'mt-2 h-10 w-full rounded-md border border-hairline-strong bg-surface-card px-3.5 text-sm text-ink outline-none transition placeholder:text-mute focus:border-ink';
</script>

<div class="mx-auto max-w-2xl">
	<a href="/painel" class="text-sm text-mute transition hover:text-ink">← Voltar ao painel</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Disponibilidade</h1>
	<p class="mt-3 text-body">Defina os dias e horários em que você atende, e exceções pontuais.</p>

	<div class="mt-8 rounded-lg border border-hairline-strong bg-surface-card p-8">
		<h2 class="text-lg font-semibold text-ink">Grade semanal</h2>

		<form class="mt-4 space-y-5" novalidate onsubmit={salvarGrade}>
			{#if erroGrade}
				<div
					class="flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
					<span class="text-body">{erroGrade}</span>
				</div>
			{/if}
			{#if sucessoGrade}
				<div
					class="flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-green"></span>
					<span class="text-body">Grade semanal salva.</span>
				</div>
			{/if}

			{#each nomesDias as nomeDia, dia (dia)}
				<div class="rounded-md border border-hairline-strong p-4">
					<p class="text-sm font-medium text-ink">{nomeDia}</p>

					<div class="mt-3 space-y-2">
						{#each grade[dia] as bloco, index (index)}
							<div class="flex items-center gap-2">
								<input
									type="time"
									step="900"
									value={minutosParaHHMM(bloco.inicioMinutos)}
									onchange={(e) => (grade[dia][index].inicioMinutos = hhmmParaMinutos(e.currentTarget.value))}
									class={inputClasse}
								/>
								<span class="text-mute">–</span>
								<input
									type="time"
									step="900"
									value={minutosParaHHMM(bloco.fimMinutos)}
									onchange={(e) => (grade[dia][index].fimMinutos = hhmmParaMinutos(e.currentTarget.value))}
									class={inputClasse}
								/>
								<button
									type="button"
									onclick={() => removerBloco(dia, index)}
									class="text-sm text-mute transition hover:text-accent-red"
								>
									Remover
								</button>
							</div>
						{/each}
					</div>

					<button
						type="button"
						onclick={() => adicionarBloco(dia)}
						class="mt-3 text-sm font-medium text-ink underline"
					>
						+ Adicionar bloco
					</button>
				</div>
			{/each}

			<button
				type="submit"
				disabled={enviandoGrade}
				class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{enviandoGrade ? 'Salvando…' : 'Salvar grade'}
			</button>
		</form>
	</div>

	<div class="mt-8 rounded-lg border border-hairline-strong bg-surface-card p-8">
		<h2 class="text-lg font-semibold text-ink">Exceções por data</h2>
		<p class="mt-1 text-sm text-body">Bloqueie um dia específico ou libere um horário extra.</p>

		{#if excecoes.length > 0}
			<ul class="mt-4 space-y-2">
				{#each excecoes as excecao (excecao.id)}
					<li class="flex items-center justify-between rounded-md border border-hairline-strong p-3 text-sm">
						<span class="text-ink">
							{excecao.data} — {excecao.tipo === 'bloqueio' ? 'Bloqueio' : 'Extra'}
							{#if excecao.tipo === 'extra'}
								({excecao.blocos.map((b) => `${minutosParaHHMM(b.inicioMinutos)}-${minutosParaHHMM(b.fimMinutos)}`).join(', ')})
							{/if}
						</span>
						<button
							type="button"
							onclick={() => excluirExcecao(excecao.id)}
							class="text-mute transition hover:text-accent-red"
						>
							Remover
						</button>
					</li>
				{/each}
			</ul>
		{/if}

		<form class="mt-5 space-y-4" novalidate onsubmit={adicionarExcecao}>
			{#if erroExcecao}
				<div
					class="flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
					<span class="text-body">{erroExcecao}</span>
				</div>
			{/if}

			<div class="flex flex-wrap items-end gap-3">
				<div>
					<label for="nova-data" class="block text-sm font-medium text-ink">Data</label>
					<input id="nova-data" type="date" bind:value={novaData} required class={inputClasse} />
				</div>

				<div role="radiogroup" aria-label="Tipo de exceção" class="flex gap-2">
					<label
						class="cursor-pointer rounded-md border px-3 py-2 text-sm {novoTipo === 'bloqueio'
							? 'border-ink bg-surface-elevated text-ink'
							: 'border-hairline-strong text-mute'}"
					>
						<input type="radio" name="tipo" value="bloqueio" bind:group={novoTipo} class="sr-only" />
						Bloqueio
					</label>
					<label
						class="cursor-pointer rounded-md border px-3 py-2 text-sm {novoTipo === 'extra'
							? 'border-ink bg-surface-elevated text-ink'
							: 'border-hairline-strong text-mute'}"
					>
						<input type="radio" name="tipo" value="extra" bind:group={novoTipo} class="sr-only" />
						Extra
					</label>
				</div>

				{#if novoTipo === 'extra'}
					<div class="flex items-center gap-2">
						<input type="time" step="900" bind:value={novoBlocoInicio} class={inputClasse} />
						<span class="text-mute">–</span>
						<input type="time" step="900" bind:value={novoBlocoFim} class={inputClasse} />
					</div>
				{/if}
			</div>

			<button
				type="submit"
				disabled={enviandoExcecao || !novaData}
				class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{enviandoExcecao ? 'Adicionando…' : 'Adicionar exceção'}
			</button>
		</form>
	</div>
</div>
