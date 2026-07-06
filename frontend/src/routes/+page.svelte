<script lang="ts">
	import { sessao } from '$lib/stores/session.svelte';
</script>

<section class="relative">
	<div class="atmos" aria-hidden="true"></div>

	<div class="relative py-20 text-center sm:py-28">
		<span
			class="inline-flex items-center gap-2 rounded-full border border-hairline-strong bg-surface-elevated px-3 py-1 text-xs text-body"
		>
			<span class="h-2 w-2 rounded-full bg-accent-green"></span>
			API no ar
		</span>

		<h1 class="display mx-auto mt-8 max-w-3xl text-6xl text-ink sm:text-7xl">
			Agendamento reimaginado
		</h1>

		<p class="mx-auto mt-6 max-w-xl text-lg text-body">
			A agenda entre clientes e prestadores de serviço — uma API limpa em Go, pronta para crescer.
		</p>

		<div class="mt-10 flex justify-center gap-3">
			{#if sessao.usuario}
				<a
					href="/painel"
					class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90"
				>
					Ir para o painel
				</a>
			{:else}
				<a
					href="/cadastro"
					class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90"
				>
					Criar conta
				</a>
				<a
					href="/login"
					class="inline-flex h-9 items-center rounded-md border border-hairline-strong px-4 text-sm font-medium text-ink transition hover:bg-surface-elevated"
				>
					Entrar
				</a>
			{/if}
		</div>
	</div>
</section>
