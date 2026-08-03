<!-- Resumo analítico do período na home do prestador: as taxas que resumem a
     agenda e o funil de status por trás delas.

     Os números chegam agregados da API. Contá-los aqui, sobre a lista de
     agendamentos, só enxergaria a primeira página do histórico — e passaria a
     mentir em silêncio no dia em que o prestador cruzasse esse limite. -->
<script lang="ts">
	import type { StatusAgendamento } from '$lib/api/appointments';
	import type { Metricas } from '$lib/api/metricas';
	import { duracaoEmHoras, porcentagem, rotuloStatus } from '$lib/format';
	import Indicadores from './Indicadores.svelte';

	let { metricas }: { metricas: Metricas } = $props();

	// Ordem do ciclo de vida, do pedido aos desfechos — espelha
	// appointment.TodosOsStatus no backend. A cor de cada faixa é a mesma das
	// listas de agendamento (rotuloStatus): o estado se reconhece igual em
	// qualquer tela. Vermelho e laranja ficam perto demais para alguns tipos de
	// daltonismo, e é por isso que toda barra carrega rótulo e número — a cor
	// nunca é a única pista.
	const ordemDoFunil: StatusAgendamento[] = [
		'SOLICITADO',
		'CONFIRMADO',
		'REALIZADO',
		'NAO_COMPARECEU',
		'CANCELADO',
		'RECUSADO',
		'EXPIRADO'
	];

	const dias = $derived(
		Math.round(
			(Date.parse(metricas.ate) - Date.parse(metricas.de)) / (24 * 60 * 60 * 1000)
		) + 1
	);

	const indicadores = $derived([
		{
			rotulo: 'Ocupação da agenda',
			valor: porcentagem(metricas.taxaOcupacao),
			detalhe: `${duracaoEmHoras(metricas.minutosReservados)} de ${duracaoEmHoras(metricas.minutosOfertados)}`
		},
		{
			rotulo: 'Comparecimento',
			valor: porcentagem(metricas.taxaComparecimento),
			detalhe: `${metricas.porStatus.REALIZADO} realizados, ${metricas.porStatus.NAO_COMPARECEU} ausências`
		},
		{ rotulo: 'Agendamentos', valor: metricas.total }
	]);

	// A barra é proporcional ao maior status, não ao total: num funil saudável
	// quase tudo vira uma faixa fina contra o total, e as diferenças entre os
	// desfechos menores — que é o que se quer enxergar — desaparecem.
	const maior = $derived(Math.max(1, ...ordemDoFunil.map((s) => metricas.porStatus[s] ?? 0)));

	const linhas = $derived(
		ordemDoFunil.map((status) => {
			const quantidade = metricas.porStatus[status] ?? 0;
			return {
				status,
				quantidade,
				rotulo: rotuloStatus(status),
				largura: (quantidade / maior) * 100,
				fatia: metricas.total > 0 ? Math.round((quantidade / metricas.total) * 100) : 0
			};
		})
	);
</script>

<section class="mt-6">
	<h2 class="mb-2 text-sm font-semibold text-ink">Últimos {dias} dias</h2>

	<Indicadores itens={indicadores} />

	<ul class="mt-3 space-y-2 rounded-xl border border-hairline-strong bg-surface-card px-5 py-4">
		{#each linhas as linha (linha.status)}
			<li class="flex items-center gap-3" title="{linha.fatia}% do período">
				<span class="w-40 shrink-0 truncate text-xs text-mute">{linha.rotulo.texto}</span>
				<span class="h-2 flex-1 overflow-hidden rounded-full bg-hairline">
					{#if linha.quantidade > 0}
						<span
							class="block h-full rounded-full {linha.rotulo.cor}"
							style="width: {linha.largura}%"
						></span>
					{/if}
				</span>
				<span
					class="w-8 shrink-0 text-right text-xs tabular-nums {linha.quantidade > 0
						? 'text-ink'
						: 'text-mute'}"
				>
					{linha.quantidade}
				</span>
			</li>
		{/each}
	</ul>
</section>
