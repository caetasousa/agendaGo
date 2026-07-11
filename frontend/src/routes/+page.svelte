<script lang="ts">
	import { sessao } from '$lib/stores/session.svelte';

	const recursos = [
		{
			cor: 'bg-accent-green',
			titulo: 'Calendário de disponibilidade',
			descricao: 'Bloqueie um dia ou defina horários próprios em dois cliques — o resto segue seu expediente.'
		},
		{
			cor: 'bg-accent-blue',
			titulo: 'Expediente do seu jeito',
			descricao: 'Quantos períodos quiser por dia útil, com os intervalos que fizerem sentido para você.'
		},
		{
			cor: 'bg-accent-orange',
			titulo: 'Agendamento com confirmação',
			descricao: 'Clientes solicitam horários livres e você confirma — anti-overbooking por padrão. Em breve.'
		}
	];
</script>

<section class="relative">
	<div class="atmos" aria-hidden="true"></div>

	<div class="relative py-20 text-center sm:py-28">
		<span
			class="inline-flex items-center gap-2 rounded-full border border-hairline-strong bg-surface-elevated px-3 py-1 text-xs text-body"
		>
			<span class="h-2 w-2 rounded-full bg-accent-green"></span>
			Projeto em construção
		</span>

		<h1 class="display mx-auto mt-8 max-w-3xl text-6xl text-ink sm:text-7xl">
			Agendamento reimaginado
		</h1>

		<p class="mx-auto mt-6 max-w-xl text-lg text-body">
			Você define quando atende; seus clientes marcam sem fricção. Simples para quem presta o
			serviço, claro para quem agenda.
		</p>

		<div class="mt-10 flex justify-center gap-3">
			{#if sessao.usuario}
				<a
					href="/painel"
					class="inline-flex h-10 items-center rounded-md bg-primary px-5 text-sm font-medium text-primary-on transition hover:opacity-90"
				>
					Ir para o painel
				</a>
			{:else}
				<a
					href="/cadastro"
					class="inline-flex h-10 items-center rounded-md bg-primary px-5 text-sm font-medium text-primary-on transition hover:opacity-90"
				>
					Criar conta
				</a>
				<a
					href="/login"
					class="inline-flex h-10 items-center rounded-md border border-hairline-strong px-5 text-sm font-medium text-ink transition hover:bg-surface-elevated"
				>
					Entrar
				</a>
			{/if}
		</div>
	</div>
</section>

<section class="mx-auto grid max-w-4xl gap-4 pb-12 sm:grid-cols-3">
	{#each recursos as recurso (recurso.titulo)}
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
			<span class="h-2 w-2 rounded-full {recurso.cor} block" aria-hidden="true"></span>
			<h2 class="mt-4 text-sm font-semibold text-ink">{recurso.titulo}</h2>
			<p class="mt-2 text-sm text-body">{recurso.descricao}</p>
		</div>
	{/each}
</section>
