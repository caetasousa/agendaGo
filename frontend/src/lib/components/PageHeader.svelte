<!-- Cabeçalho padrão das telas de aplicação (painel e moderação). Título em
     escala de ferramenta — a serifa de display fica reservada às páginas
     públicas, onde é manchete, não rótulo de tela.
     A descrição vem como texto (`descricao`) ou, quando precisa de links, como
     conteúdo filho. -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		titulo,
		descricao,
		acoes,
		children
	}: {
		titulo: string;
		descricao?: string;
		acoes?: Snippet;
		children?: Snippet;
	} = $props();
</script>

<header
	class="mb-6 flex flex-wrap items-start justify-between gap-x-6 gap-y-3 border-b border-hairline pb-5"
>
	<div class="min-w-0">
		<h1 class="text-2xl font-semibold tracking-tight text-balance text-ink">{titulo}</h1>
		{#if children}
			<div class="mt-1.5 max-w-2xl text-sm text-body">{@render children()}</div>
		{:else if descricao}
			<p class="mt-1.5 max-w-2xl text-sm text-body">{descricao}</p>
		{/if}
	</div>

	{#if acoes}
		<div class="flex shrink-0 items-center gap-2">
			{@render acoes()}
		</div>
	{/if}
</header>
