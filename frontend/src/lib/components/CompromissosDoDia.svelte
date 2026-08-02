<!--
  Compromissos pessoais de uma data, dentro do modal de disponibilidade.

  Vive num componente próprio de propósito: a página de disponibilidade já é o
  maior arquivo do projeto, e embutir listagem + formulário + remoção ali
  passaria de 700 linhas. Aqui o estado é só desta seção e some junto com ela.
-->
<script lang="ts">
	import { ApiError } from '$lib/api/client';
	import {
		criarOcupacao,
		listarOcupacoes,
		removerOcupacao,
		type Ocupacao
	} from '$lib/api/ocupacoes';

	// data no formato YYYY-MM-DD; podeEditar espelha a regra de antecedência da
	// página, para o formulário não oferecer o que o backend vai recusar.
	let { data, podeEditar = true }: { data: string; podeEditar?: boolean } = $props();

	let compromissos = $state<Ocupacao[]>([]);
	let carregando = $state(true);
	let erro = $state<string | null>(null);
	let salvando = $state(false);

	let inicio = $state('14:00');
	let fim = $state('15:00');
	let titulo = $state('');

	function hhmmParaMinutos(hhmm: string): number {
		const [h, m] = hhmm.split(':').map(Number);
		return h * 60 + m;
	}

	function minutosParaHHMM(minutos: number): string {
		const h = Math.floor(minutos / 60)
			.toString()
			.padStart(2, '0');
		return `${h}:${(minutos % 60).toString().padStart(2, '0')}`;
	}

	async function carregar() {
		carregando = true;
		erro = null;
		try {
			const resp = await listarOcupacoes(data, data);
			compromissos = resp.ocupacoes;
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível carregar os compromissos.';
		} finally {
			carregando = false;
		}
	}

	// Recarrega quando a data muda: o modal é reusado entre dias sem remontar.
	$effect(() => {
		data;
		carregar();
	});

	async function adicionar() {
		erro = null;
		salvando = true;
		try {
			await criarOcupacao({
				data,
				inicioMinutos: hhmmParaMinutos(inicio),
				fimMinutos: hhmmParaMinutos(fim),
				titulo
			});
			titulo = '';
			await carregar();
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível criar o compromisso.';
		} finally {
			salvando = false;
		}
	}

	async function remover(id: string) {
		erro = null;
		salvando = true;
		try {
			await removerOcupacao(id);
			await carregar();
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível remover o compromisso.';
		} finally {
			salvando = false;
		}
	}
</script>

<div class="mt-6 border-t border-hairline-strong pt-4">
	<h4 class="text-sm font-semibold text-ink">Compromissos pessoais</h4>
	<p class="mt-1 text-xs text-mute">
		O horário some da sua oferta sem alterar o expediente do dia.
	</p>

	{#if erro}
		<div
			class="mt-3 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
		>
			<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
			<span class="text-body">{erro}</span>
		</div>
	{/if}

	{#if carregando}
		<p class="mt-3 text-sm text-mute">Carregando…</p>
	{:else if compromissos.length > 0}
		<ul class="mt-3 space-y-2">
			{#each compromissos as c (c.id)}
				<li
					class="flex items-center justify-between gap-3 rounded-md border border-hairline-strong bg-surface-elevated px-3 py-2"
				>
					<span class="text-sm text-body">
						{minutosParaHHMM(c.inicioMinutos)}–{minutosParaHHMM(c.fimMinutos)}
						{#if c.titulo}<span class="text-mute">· {c.titulo}</span>{/if}
					</span>
					<button
						type="button"
						disabled={salvando || !podeEditar}
						onclick={() => remover(c.id)}
						class="text-sm text-accent-red transition hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
					>
						Remover
					</button>
				</li>
			{/each}
		</ul>
	{:else}
		<p class="mt-3 text-sm text-mute">Nenhum compromisso neste dia.</p>
	{/if}

	{#if podeEditar}
		<div class="mt-4 flex flex-wrap items-end gap-2">
			<label class="flex flex-col gap-1">
				<span class="text-xs font-medium tracking-wide text-mute uppercase">Início</span>
				<input type="time" step="900" bind:value={inicio} class="campo-linha h-9 w-28 px-2 text-sm" />
			</label>
			<label class="flex flex-col gap-1">
				<span class="text-xs font-medium tracking-wide text-mute uppercase">Fim</span>
				<input type="time" step="900" bind:value={fim} class="campo-linha h-9 w-28 px-2 text-sm" />
			</label>
			<label class="flex min-w-40 flex-1 flex-col gap-1">
				<span class="text-xs font-medium tracking-wide text-mute uppercase">Descrição</span>
				<input
					type="text"
					maxlength="120"
					placeholder="Opcional"
					bind:value={titulo}
					class="campo-linha h-9 px-2 text-sm"
				/>
			</label>
			<button
				type="button"
				disabled={salvando}
				onclick={adicionar}
				class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{salvando ? 'Salvando…' : 'Adicionar'}
			</button>
		</div>
	{/if}
</div>
