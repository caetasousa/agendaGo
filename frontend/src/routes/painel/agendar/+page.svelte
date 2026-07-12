<script lang="ts">
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<div class="mx-auto max-w-2xl">
	<a href="/painel" class="text-sm text-mute transition hover:text-ink">← Voltar ao painel</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Agendar</h1>
	<p class="mt-3 text-body">
		Escolha um prestador para ver o calendário de horários livres e solicitar seu atendimento.
	</p>

	{#if data.prestadores.length === 0}
		<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-8">
			<p class="text-sm text-body">Nenhum prestador cadastrado ainda. Volte mais tarde.</p>
		</div>
	{:else}
		<div class="mt-8 grid gap-4 sm:grid-cols-2">
			{#each data.prestadores as prestador (prestador.id)}
				<a
					href="/agendar/{prestador.id}"
					class="group rounded-xl border border-hairline-strong bg-surface-card p-6 transition hover:border-ink/40"
				>
					<span
						class="block h-2 w-2 rounded-full {prestador.aceitaAgendamentos
							? 'bg-accent-green'
							: 'bg-accent-yellow'}"
						aria-hidden="true"
					></span>
					<h2 class="mt-4 flex items-center justify-between text-base font-semibold text-ink">
						{prestador.nome}
						<span class="text-mute transition group-hover:translate-x-0.5 group-hover:text-ink">→</span>
					</h2>
					<p class="mt-2 text-sm text-body">
						{prestador.aceitaAgendamentos
							? `Atendimento de ${prestador.duracaoAtendimentoMinutos} min — ver horários livres.`
							: 'Sem horários no momento.'}
					</p>
				</a>
			{/each}
		</div>
	{/if}
</div>
