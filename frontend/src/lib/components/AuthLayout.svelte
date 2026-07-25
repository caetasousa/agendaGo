<!-- Moldura das telas de autenticação: manchete editorial à esquerda e o
     formulário solto sobre o fundo à direita, sem cartão. A manchete muda
     conforme o contexto (entrar, criar conta de cliente, de prestador…), por
     isso vem como propriedade em vez de texto fixo. -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		titulo,
		lede,
		children
	}: { titulo: string; lede: string; children: Snippet } = $props();
</script>

<!-- Colunas ancoradas no topo: com `items-center`, a manchete descia para o meio
     quando o formulário era alto (cadastro) e parecia solta dele. O recuo no
     topo alinha a manchete opticamente com o primeiro campo. -->
<div class="grid items-start gap-10 lg:grid-cols-2 lg:gap-16">
	<div class="lg:pt-1.5">
		<h1 class="display text-4xl text-balance text-ink sm:text-5xl">{titulo}</h1>
		<p class="mt-4 max-w-[34ch] text-base leading-relaxed text-body">{lede}</p>
	</div>

	<div class="w-full max-w-md lg:justify-self-end">
		{@render children()}
	</div>
</div>
