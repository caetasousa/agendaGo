<script lang="ts">
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<div class="mx-auto max-w-xl">
	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Painel</h1>

	<div class="mt-8 rounded-lg border border-hairline-strong bg-surface-card p-8">
		<div class="flex items-center gap-2">
			<span class="h-2 w-2 rounded-full bg-accent-green"></span>
			<p class="font-medium text-ink">Você está logado!</p>
		</div>
		<p class="mt-2 text-sm text-body">Nome: {data.usuario.nome}</p>
		<p class="text-sm text-body">E-mail: {data.usuario.email}</p>
		<p class="text-sm text-body">
			Tipo: {data.usuario.tipo === 'provider' ? 'Prestador' : 'Cliente'}
		</p>

		{#if data.usuario.tipo === 'provider'}
			<div class="mt-6 flex gap-3">
				<a
					href="/painel/preferencias"
					class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90"
				>
					Preferências
				</a>
				<a
					href="/painel/disponibilidade"
					class="inline-flex h-9 items-center rounded-md border border-hairline-strong px-4 text-sm font-medium text-ink transition hover:bg-surface-elevated"
				>
					Disponibilidade
				</a>
			</div>
		{/if}
	</div>
</div>
