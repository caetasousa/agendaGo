<script lang="ts">
	import { page } from '$app/state';
	import { sessao } from '$lib/stores/session.svelte';
	import Icone from '$lib/components/Icone.svelte';

	let { children } = $props();

	const itens = [{ href: '/admin', rotulo: 'Visão geral', icone: 'moderacao' as const }];

	function ativo(href: string): boolean {
		return page.url.pathname === href;
	}

	const iniciais = $derived.by(() => {
		const partes = (sessao.usuario?.nome ?? '').trim().split(/\s+/).filter(Boolean);
		if (partes.length === 0) return '?';
		const primeira = partes[0][0];
		const ultima = partes.length > 1 ? partes[partes.length - 1][0] : '';
		return (primeira + ultima).toUpperCase();
	});
</script>

<div class="flex w-full flex-1 flex-col gap-6 md:flex-row md:gap-0">
	<aside class="hidden shrink-0 md:block md:w-60 md:border-r md:border-hairline md:pr-6">
		<div class="sticky top-24 flex flex-col gap-6">
			<nav class="flex flex-col gap-0.5">
				{#each itens as item (item.href)}
					{@const estaAtivo = ativo(item.href)}
					<a
						href={item.href}
						aria-current={estaAtivo ? 'page' : undefined}
						class="relative flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition {estaAtivo
							? 'bg-surface-elevated font-medium text-ink'
							: 'text-mute hover:bg-surface-card hover:text-ink'}"
					>
						{#if estaAtivo}
							<span
								class="absolute top-1/2 left-0 h-5 w-0.5 -translate-y-1/2 rounded-r-full bg-accent-blue"
								aria-hidden="true"
							></span>
						{/if}
						<Icone
							nome={item.icone}
							classe={estaAtivo ? 'shrink-0 text-accent-blue' : 'shrink-0'}
						/>
						<span class="truncate">{item.rotulo}</span>
					</a>
				{/each}
			</nav>

			<!-- A saída fica no header, único ponto visível em qualquer largura. -->
			{#if sessao.usuario}
				<div class="flex items-center gap-2.5 border-t border-hairline pt-4">
					<span
						class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-surface-elevated text-[11px] font-semibold text-ink"
						aria-hidden="true"
					>
						{iniciais}
					</span>
					<span class="min-w-0 flex-1">
						<span class="block truncate text-sm text-ink" title={sessao.usuario.nome}>
							{sessao.usuario.nome}
						</span>
						<span class="block text-xs text-mute">Moderação</span>
					</span>
				</div>
			{/if}
		</div>
	</aside>

	<div class="min-w-0 flex-1 md:pl-8">
		{@render children()}
	</div>
</div>
