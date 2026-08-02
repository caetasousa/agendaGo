<script lang="ts">
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Icone from '$lib/components/Icone.svelte';
	import { listarPrestadores, type PrestadorResumo } from '$lib/api/provider';

	let { data }: { data: PageData } = $props();

	// svelte-ignore state_referenced_locally
	let prestadores = $state<PrestadorResumo[]>(data.prestadores);
	// svelte-ignore state_referenced_locally
	let total = $state(data.total);
	let carregando = $state(false);

	// O offset é o que já está carregado; o limite fica a cargo do padrão da API.
	async function carregarMais() {
		carregando = true;
		try {
			const resposta = await listarPrestadores({ offset: prestadores.length });
			prestadores = [...prestadores, ...resposta.prestadores];
			total = resposta.total;
		} finally {
			carregando = false;
		}
	}
</script>

<div>
	<PageHeader
		titulo="Agendar"
		descricao="Escolha um prestador para ver o calendário de horários livres e solicitar seu atendimento."
	/>

	{#if prestadores.length === 0}
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-8">
			<p class="text-sm text-body">Nenhum prestador cadastrado ainda. Volte mais tarde.</p>
		</div>
	{:else}
		<div class="grid gap-3 sm:grid-cols-2">
			{#each prestadores as prestador (prestador.id)}
				<a
					href="/agendar/{prestador.slug}"
					class="group rounded-xl border border-hairline-strong bg-surface-card p-5 transition hover:border-ink/30 hover:bg-surface-elevated/40"
				>
					<div class="flex items-start justify-between gap-3">
						<h2 class="min-w-0 text-base font-semibold text-balance text-ink">{prestador.nome}</h2>
						<span
							class="shrink-0 text-mute transition group-hover:translate-x-0.5 group-hover:text-ink"
							aria-hidden="true"
						>
							<Icone nome="seta-direita" tamanho={15} />
						</span>
					</div>
					<p class="mt-2 text-sm text-body">
						{prestador.aceitaAgendamentos
							? `Atendimento de ${prestador.duracaoAtendimentoMinutos} min — ver horários livres.`
							: 'Sem horários no momento.'}
					</p>
					<span
						class="mt-3 inline-flex items-center gap-1.5 rounded-full border border-hairline-strong bg-surface-elevated px-2.5 py-0.5 text-xs text-body"
					>
						<span
							class="h-1.5 w-1.5 rounded-full {prestador.aceitaAgendamentos
								? 'bg-accent-green'
								: 'bg-accent-yellow'}"
						></span>
						{prestador.aceitaAgendamentos ? 'Agenda aberta' : 'Agenda fechada'}
					</span>
				</a>
			{/each}
		</div>
		{#if prestadores.length < total}
			<div class="mt-4 flex justify-center">
				<button
					type="button"
					onclick={carregarMais}
					disabled={carregando}
					class="rounded-full border border-hairline-strong bg-surface-card px-4 py-1.5 text-sm text-body transition hover:bg-surface-elevated/40 disabled:opacity-60"
				>
					{carregando ? 'Carregando…' : 'Carregar mais'}
				</button>
			</div>
		{/if}
	{/if}
</div>
