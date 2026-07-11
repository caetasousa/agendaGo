<script lang="ts">
	// Seletor de horário em passos de 15 minutos (a granularidade do domínio).
	// Substitui o <input type="time"> nativo, cuja UI varia por navegador e
	// locale (AM/PM) e destoa do design do app. Sempre exibe formato 24h.
	let {
		valor = $bindable(),
		id,
		ariaLabel,
		minimo = 0,
		maximo = 24 * 60 - 15
	}: {
		valor: number;
		id?: string;
		ariaLabel?: string;
		minimo?: number;
		maximo?: number;
	} = $props();

	function hhmm(minutos: number): string {
		const h = Math.floor(minutos / 60)
			.toString()
			.padStart(2, '0');
		const m = (minutos % 60).toString().padStart(2, '0');
		return `${h}:${m}`;
	}

	function paraMinutos(texto: string): number {
		const [h, m] = texto.split(':').map(Number);
		return h * 60 + m;
	}

	const opcoes = $derived.by(() => {
		const lista: number[] = [];
		for (let m = minimo; m <= maximo; m += 15) {
			lista.push(m);
		}
		return lista;
	});
</script>

<div class="relative w-full">
	<select
		{id}
		aria-label={ariaLabel}
		value={hhmm(valor)}
		onchange={(e) => (valor = paraMinutos(e.currentTarget.value))}
		class="h-10 w-full cursor-pointer appearance-none rounded-md border border-hairline-strong bg-surface-card px-3 pr-8 text-sm text-ink outline-none transition focus:border-ink"
	>
		{#each opcoes as minutos (minutos)}
			<option value={hhmm(minutos)}>{hhmm(minutos)}</option>
		{/each}
	</select>
	<span class="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 text-xs text-mute">▾</span>
</div>
