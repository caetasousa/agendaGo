<script lang="ts">
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const ehPrestador = $derived(data.usuario.tipo === 'provider');
</script>

<div class="mx-auto max-w-2xl">
	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Olá, {data.usuario.nome}</h1>
	<p class="mt-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-body">
		<span
			class="inline-flex items-center gap-1.5 rounded-full border border-hairline-strong bg-surface-elevated px-2.5 py-0.5 text-xs font-medium text-ink"
		>
			<span class="h-1.5 w-1.5 rounded-full bg-accent-green"></span>
			{ehPrestador ? 'Conta de prestador' : 'Conta de cliente'}
		</span>
		<span class="text-sm text-mute">{data.usuario.email}</span>
	</p>

	{#if ehPrestador}
		<div class="mt-10 grid gap-4 sm:grid-cols-2">
			<a
				href="/painel/disponibilidade"
				class="group rounded-xl border border-hairline-strong bg-surface-card p-6 transition hover:border-ink/40"
			>
				<span class="block h-2 w-2 rounded-full bg-accent-green" aria-hidden="true"></span>
				<h2 class="mt-4 flex items-center justify-between text-base font-semibold text-ink">
					Disponibilidade
					<span class="text-mute transition group-hover:translate-x-0.5 group-hover:text-ink">→</span>
				</h2>
				<p class="mt-2 text-sm text-body">
					Veja o mês no calendário, bloqueie dias ou defina horários próprios para datas
					específicas.
				</p>
			</a>

			<a
				href="/painel/preferencias"
				class="group rounded-xl border border-hairline-strong bg-surface-card p-6 transition hover:border-ink/40"
			>
				<span class="block h-2 w-2 rounded-full bg-accent-blue" aria-hidden="true"></span>
				<h2 class="mt-4 flex items-center justify-between text-base font-semibold text-ink">
					Preferências
					<span class="text-mute transition group-hover:translate-x-0.5 group-hover:text-ink">→</span>
				</h2>
				<p class="mt-2 text-sm text-body">
					Ative a agenda, ajuste o descanso entre atendimentos e monte seu expediente padrão.
				</p>
			</a>
		</div>
	{:else}
		<div class="mt-10 rounded-xl border border-hairline-strong bg-surface-card p-8">
			<div class="flex items-center gap-2">
				<span class="h-2 w-2 rounded-full bg-accent-orange"></span>
				<h2 class="text-base font-semibold text-ink">Agendamentos em breve</h2>
			</div>
			<p class="mt-2 text-sm text-body">
				Logo você vai poder buscar prestadores, ver horários livres e solicitar um atendimento por
				aqui.
			</p>
		</div>
	{/if}
</div>
