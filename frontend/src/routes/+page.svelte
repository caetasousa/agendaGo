<script lang="ts">
	import { sessao } from '$lib/stores/session.svelte';

	const recursos = [
		{
			cor: 'bg-accent-green',
			titulo: 'Calendário de disponibilidade',
			descricao:
				'Bloqueie um dia ou defina horários próprios em dois cliques — o resto segue seu expediente.'
		},
		{
			cor: 'bg-accent-blue',
			titulo: 'Expediente do seu jeito',
			descricao:
				'Quantos períodos quiser por dia útil, com os intervalos que fizerem sentido para você.'
		},
		{
			cor: 'bg-accent-orange',
			titulo: 'Agendamento com confirmação',
			descricao:
				'Clientes solicitam horários livres e você confirma — anti-overbooking por padrão.'
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

<!-- Dois públicos: cada um com um caminho claro, para ninguém se cadastrar no
     tipo de conta errado. -->
{#if !sessao.usuario}
	<section class="grid gap-4 pb-4 sm:grid-cols-2">
		<div class="flex flex-col rounded-2xl border border-hairline-strong bg-surface-card p-8">
			<span class="flex h-10 w-10 items-center justify-center rounded-lg bg-surface-elevated" aria-hidden="true">
				<span class="h-2.5 w-2.5 rounded-full bg-accent-green"></span>
			</span>
			<h2 class="display mt-5 text-2xl text-ink">Para quem atende</h2>
			<p class="mt-2 flex-1 text-sm text-body">
				Crie sua agenda, publique seus horários e receba pedidos de agendamento — com confirmação
				sua e sem overbooking. Sua conta de prestador leva menos de um minuto.
			</p>
			<div class="mt-6">
				<a
					href="/cadastro?tipo=prestador"
					class="inline-flex h-10 items-center rounded-md bg-primary px-5 text-sm font-medium text-primary-on transition hover:opacity-90"
				>
					Criar conta de prestador
				</a>
			</div>
		</div>

		<div class="flex flex-col rounded-2xl border border-hairline-strong bg-surface-card p-8">
			<span class="flex h-10 w-10 items-center justify-center rounded-lg bg-surface-elevated" aria-hidden="true">
				<span class="h-2.5 w-2.5 rounded-full bg-accent-blue"></span>
			</span>
			<h2 class="display mt-5 text-2xl text-ink">Para quem agenda</h2>
			<p class="mt-2 flex-1 text-sm text-body">
				Recebeu o link de um profissional? É só escolher um horário livre e solicitar — dá para
				agendar sem conta. Crie uma se quiser acompanhar e cancelar seus agendamentos em um só lugar.
			</p>
			<div class="mt-6">
				<a
					href="/cadastro?tipo=cliente"
					class="inline-flex h-10 items-center rounded-md border border-hairline-strong px-5 text-sm font-medium text-ink transition hover:bg-surface-elevated"
				>
					Criar conta de cliente
				</a>
			</div>
		</div>
	</section>
{/if}

<section class="pb-12 pt-8">
	<p class="text-xs font-medium uppercase tracking-wide text-mute">Para prestadores</p>
	<div class="mt-4 grid gap-4 sm:grid-cols-3">
		{#each recursos as recurso (recurso.titulo)}
			<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
				<span class="h-2 w-2 rounded-full {recurso.cor} block" aria-hidden="true"></span>
				<h3 class="mt-4 text-sm font-semibold text-ink">{recurso.titulo}</h3>
				<p class="mt-2 text-sm text-body">{recurso.descricao}</p>
			</div>
		{/each}
	</div>
</section>
