<!-- Faixa de indicadores das telas de aplicação. É uma superfície única dividida
     em colunas, e não um bloco por número: contadores soltos com borda própria
     competiam em peso com o conteúdo que eles apenas resumem. -->
<script lang="ts">
	type Indicador = { rotulo: string; valor: number; destaque?: boolean };

	let { itens }: { itens: Indicador[] } = $props();
</script>

<dl
	class="grid divide-y divide-hairline rounded-xl border border-hairline-strong bg-surface-card sm:auto-cols-fr sm:grid-flow-col sm:divide-x sm:divide-y-0"
>
	{#each itens as item (item.rotulo)}
		<div class="px-5 py-3.5">
			<dt class="truncate text-xs font-medium tracking-wide text-mute uppercase" title={item.rotulo}>
				{item.rotulo}
			</dt>
			<dd
				class="mt-0.5 text-2xl font-semibold tabular-nums {item.destaque
					? 'text-accent-yellow'
					: 'text-ink'}"
			>
				{item.valor}
			</dd>
		</div>
	{/each}
</dl>
