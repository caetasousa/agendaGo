<script lang="ts">
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import {
		consultarAgenda,
		definirDia,
		removerDia,
		type Bloco,
		type DiaAgenda
	} from '$lib/api/availability';
	import TimeSelect from '$lib/components/TimeSelect.svelte';
	import { chaveData, feriadosNacionais } from '$lib/holidays';

	let { data }: { data: PageData } = $props();

	const nomesDiasCurtos = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'];

	// Espelho de blocosComerciaisPadrao do backend — usado só como ponto de
	// partida do editor quando o dia ainda não tem horários.
	const BLOCOS_COMERCIAIS: Bloco[] = [
		{ inicioMinutos: 8 * 60, fimMinutos: 12 * 60 },
		{ inicioMinutos: 14 * 60, fimMinutos: 18 * 60 }
	];

	const AVISO_ANTECEDENCIA = 'Alterar a disponibilidade do dia de hoje exige ao menos 24h de antecedência.';

	function minutosParaHHMM(minutos: number): string {
		const h = Math.floor(minutos / 60)
			.toString()
			.padStart(2, '0');
		const m = (minutos % 60).toString().padStart(2, '0');
		return `${h}:${m}`;
	}

	function horaCurta(minutos: number): string {
		const h = Math.floor(minutos / 60);
		const m = minutos % 60;
		return m === 0 ? `${h}h` : `${h}h${m.toString().padStart(2, '0')}`;
	}

	// ---- Calendário ----

	type EstadoDia = 'disponivel' | 'indisponivel' | 'bloqueado' | 'personalizado';

	interface Celula {
		dia: number;
		chave: string;
		estado: EstadoDia;
		origem: DiaAgenda['origem'];
		blocos: Bloco[];
		feriado: string | null;
		fimDeSemana: boolean;
		ehHoje: boolean;
		passado: boolean;
	}

	const hoje = new Date();
	const chaveHoje = chaveData(hoje);

	function chaveMes(mes: Date): string {
		return `${mes.getFullYear()}-${(mes.getMonth() + 1).toString().padStart(2, '0')}`;
	}

	let mesExibido = $state(new Date(hoje.getFullYear(), hoje.getMonth(), 1));
	// svelte-ignore state_referenced_locally
	let aceitaAgendamentos = $state(data.agenda.aceitaAgendamentos);
	// Cache por mês da agenda resolvida pelo backend — recarregado após cada mutação.
	// svelte-ignore state_referenced_locally
	let agendaPorMes = $state<Record<string, DiaAgenda[]>>({
		[chaveMes(new Date(hoje.getFullYear(), hoje.getMonth(), 1))]: data.agenda.dias
	});
	let carregandoMes = $state(false);
	let erroCalendario = $state<string | null>(null);

	const ehMesAtual = $derived(
		mesExibido.getFullYear() === hoje.getFullYear() && mesExibido.getMonth() === hoje.getMonth()
	);

	const rotuloMes = $derived.by(() => {
		const rotulo = new Intl.DateTimeFormat('pt-BR', { month: 'long', year: 'numeric' }).format(mesExibido);
		return rotulo.charAt(0).toUpperCase() + rotulo.slice(1);
	});

	async function carregarMes(mes: Date, forcar = false) {
		const chave = chaveMes(mes);
		if (!forcar && agendaPorMes[chave]) return;

		carregandoMes = true;
		erroCalendario = null;
		try {
			const de = chaveData(new Date(mes.getFullYear(), mes.getMonth(), 1));
			const ate = chaveData(new Date(mes.getFullYear(), mes.getMonth() + 1, 0));
			const resp = await consultarAgenda(de, ate);
			agendaPorMes = { ...agendaPorMes, [chave]: resp.dias };
			aceitaAgendamentos = resp.aceitaAgendamentos;
		} catch (e) {
			erroCalendario = e instanceof ApiError ? e.message : 'Não foi possível carregar a agenda.';
		} finally {
			carregandoMes = false;
		}
	}

	function mudarMes(delta: number) {
		mesExibido = new Date(mesExibido.getFullYear(), mesExibido.getMonth() + delta, 1);
		carregarMes(mesExibido);
	}

	// celulas monta a grade do mês a partir da agenda resolvida pelo backend
	// (origem padrão/bloqueio/extra), acrescentando o contexto visual local
	// (feriados, fim de semana, hoje/passado).
	const celulas = $derived.by<(Celula | null)[]>(() => {
		const dias = agendaPorMes[chaveMes(mesExibido)];
		if (!dias) return [];

		const ano = mesExibido.getFullYear();
		const mes = mesExibido.getMonth();
		const feriados = feriadosNacionais(ano);
		const porChave = new Map(dias.map((d) => [d.data, d]));
		const resultado: (Celula | null)[] = [];

		for (let i = 0; i < new Date(ano, mes, 1).getDay(); i++) {
			resultado.push(null);
		}

		const totalDias = new Date(ano, mes + 1, 0).getDate();
		for (let dia = 1; dia <= totalDias; dia++) {
			const dataDia = new Date(ano, mes, dia);
			const chave = chaveData(dataDia);
			const diaSemana = dataDia.getDay();
			const resolvido = porChave.get(chave);
			const origem = resolvido?.origem ?? 'padrao';
			const blocos = resolvido?.blocos ?? [];

			let estado: EstadoDia;
			if (origem === 'bloqueio') estado = 'bloqueado';
			else if (origem === 'extra') estado = 'personalizado';
			else estado = blocos.length > 0 ? 'disponivel' : 'indisponivel';

			resultado.push({
				dia,
				chave,
				estado,
				origem,
				blocos,
				feriado: feriados.get(chave) ?? null,
				fimDeSemana: diaSemana === 0 || diaSemana === 6,
				ehHoje: chave === chaveHoje,
				passado: chave < chaveHoje
			});
		}

		return resultado;
	});

	function rotuloEstado(celula: Celula): string {
		const horarios = celula.blocos
			.map((b) => `${minutosParaHHMM(b.inicioMinutos)}–${minutosParaHHMM(b.fimMinutos)}`)
			.join(', ');
		switch (celula.estado) {
			case 'disponivel':
				return `Disponível (${horarios})`;
			case 'personalizado':
				return `Horários próprios (${horarios})`;
			case 'bloqueado':
				return 'Indisponível (bloqueado)';
			default:
				return celula.fimDeSemana ? 'Indisponível — fim de semana' : 'Indisponível';
		}
	}

	function tituloCelula(celula: Celula): string {
		const rotulo = rotuloEstado(celula);
		return celula.feriado ? `Feriado: ${celula.feriado} — ${rotulo}` : rotulo;
	}

	function classesCelula(celula: Celula): string {
		let classes: string;
		if (celula.estado === 'disponivel' || celula.estado === 'personalizado') {
			classes = 'border-accent-green/40 bg-accent-green/10';
		} else if (celula.estado === 'bloqueado') {
			classes = 'border-accent-red/40 bg-accent-red/10';
		} else {
			classes = 'border-hairline';
		}

		if (celula.passado) {
			return `${classes} cursor-not-allowed opacity-35`;
		}
		if (celula.estado === 'indisponivel') {
			classes += ' hover:border-hairline-strong hover:bg-surface-elevated';
		} else if (celula.estado === 'bloqueado') {
			classes += ' hover:border-accent-red/80';
		} else {
			classes += ' hover:border-accent-green/80';
		}
		return classes;
	}

	// ---- Modal de edição do dia ----

	let diaSelecionado = $state<Celula | null>(null);
	let blocosEdicao = $state<Bloco[]>([]);
	let salvandoModal = $state(false);
	let erroModal = $state<string | null>(null);

	const tituloDia = $derived.by(() => {
		if (!diaSelecionado) return '';
		const [ano, mes, dia] = diaSelecionado.chave.split('-').map(Number);
		const rotulo = new Intl.DateTimeFormat('pt-BR', {
			weekday: 'long',
			day: 'numeric',
			month: 'long'
		}).format(new Date(ano, mes - 1, dia));
		return rotulo.charAt(0).toUpperCase() + rotulo.slice(1);
	});

	// Qualquer alteração no dia de hoje exige 24h de antecedência — não se
	// oferta nem se cancela oferta em cima da hora. A única exceção é desfazer
	// um bloqueio (restaurar padrão a partir de "bloqueado"): isso só reduz a
	// oferta, nunca a amplia, então nunca surpreende um cliente.
	const podeEditarHoje = $derived(diaSelecionado !== null && diaSelecionado.chave > chaveHoje);
	const podeRestaurarPadrao = $derived(
		diaSelecionado !== null && (podeEditarHoje || diaSelecionado.estado === 'bloqueado')
	);

	function abrirModal(celula: Celula) {
		if (celula.passado) return;
		diaSelecionado = celula;
		blocosEdicao = celula.blocos.length > 0 ? celula.blocos.map((b) => ({ ...b })) : BLOCOS_COMERCIAIS.map((b) => ({ ...b }));
		erroModal = null;
	}

	function fecharModal() {
		if (salvandoModal) return;
		diaSelecionado = null;
	}

	function adicionarBlocoEdicao() {
		blocosEdicao = [...blocosEdicao, { inicioMinutos: 480, fimMinutos: 720 }];
	}

	function removerBlocoEdicao(index: number) {
		blocosEdicao = blocosEdicao.filter((_, i) => i !== index);
	}

	async function executarMutacao(acao: () => Promise<unknown>) {
		erroModal = null;
		salvandoModal = true;
		try {
			await acao();
			await carregarMes(mesExibido, true);
			diaSelecionado = null;
		} catch (e) {
			erroModal = e instanceof ApiError ? e.message : 'Não foi possível salvar o dia.';
		} finally {
			salvandoModal = false;
		}
	}

	function salvarHorarios() {
		if (!diaSelecionado) return;
		const chave = diaSelecionado.chave;
		executarMutacao(() => definirDia(chave, { tipo: 'extra', blocos: blocosEdicao }));
	}

	function marcarIndisponivel() {
		if (!diaSelecionado) return;
		const chave = diaSelecionado.chave;
		executarMutacao(() => definirDia(chave, { tipo: 'bloqueio', blocos: [] }));
	}

	function restaurarPadrao() {
		if (!diaSelecionado) return;
		const chave = diaSelecionado.chave;
		executarMutacao(() => removerDia(chave));
	}

</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && diaSelecionado) fecharModal();
	}}
/>

<div class="mx-auto max-w-2xl">
	<a href="/painel" class="text-sm text-mute transition hover:text-ink">← Voltar ao painel</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Disponibilidade</h1>
	<p class="mt-3 text-body">
		Seus dias úteis seguem o expediente padrão definido em
		<a href="/painel/preferencias" class="font-medium text-ink underline">Preferências</a>. Clique
		em um dia do calendário para bloqueá-lo ou definir horários próprios.
	</p>

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-5 sm:p-8">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h2 class="text-lg font-semibold text-ink">Calendário</h2>

			<div class="flex items-center gap-1">
				<button
					type="button"
					aria-label="Mês anterior"
					disabled={ehMesAtual}
					onclick={() => mudarMes(-1)}
					class="flex h-8 w-8 items-center justify-center rounded-md border border-hairline-strong text-ink transition hover:bg-surface-elevated disabled:cursor-not-allowed disabled:opacity-40"
				>
					‹
				</button>
				<span class="w-36 text-center text-sm font-medium text-ink">{rotuloMes}</span>
				<button
					type="button"
					aria-label="Próximo mês"
					onclick={() => mudarMes(1)}
					class="flex h-8 w-8 items-center justify-center rounded-md border border-hairline-strong text-ink transition hover:bg-surface-elevated"
				>
					›
				</button>
			</div>
		</div>

		<div class="mt-4 flex flex-wrap gap-x-4 gap-y-1.5 text-xs text-mute">
			<span class="flex items-center gap-1.5">
				<span class="h-2 w-2 rounded-full bg-accent-green"></span>Disponível
			</span>
			<span class="flex items-center gap-1.5">
				<span class="h-2 w-2 rounded-full border border-hairline-strong"></span>Indisponível
			</span>
			<span class="flex items-center gap-1.5">
				<span class="h-2 w-2 rounded-full bg-accent-red"></span>Bloqueado
			</span>
			<span class="flex items-center gap-1.5">
				<span class="h-2 w-2 rounded-full bg-accent-blue"></span>Horários próprios
			</span>
			<span class="flex items-center gap-1.5">
				<span class="h-2 w-2 rounded-full bg-accent-yellow"></span>Feriado
			</span>
		</div>

		{#if !aceitaAgendamentos}
			<div
				class="mt-4 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
			>
				<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-yellow"></span>
				<span class="text-body">
					Sua agenda está desativada, então os dias úteis não oferecem o expediente padrão. Ative
					em <a href="/painel/preferencias" class="font-medium text-ink underline">Preferências</a>.
				</span>
			</div>
		{/if}

		{#if erroCalendario}
			<div
				class="mt-4 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
			>
				<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
				<span class="text-body">{erroCalendario}</span>
			</div>
		{/if}

		<div class="mt-4 grid grid-cols-7 gap-1 sm:gap-1.5 {carregandoMes ? 'opacity-60' : ''}">
			{#each nomesDiasCurtos as nome, i (nome)}
				<div class="pb-1 text-center text-xs font-medium {i === 0 || i === 6 ? 'text-ash' : 'text-mute'}">
					{nome}
				</div>
			{/each}

			{#each celulas as celula, i (celula?.chave ?? `vazio-${i}`)}
				{#if celula}
					<button
						type="button"
						disabled={celula.passado}
						data-data={celula.chave}
						data-estado={celula.estado}
						title={tituloCelula(celula)}
						aria-label="{celula.chave}: {rotuloEstado(celula)}"
						onclick={() => abrirModal(celula)}
						class="relative flex aspect-square flex-col items-start gap-0.5 overflow-hidden rounded-md border p-1 text-left transition sm:p-1.5 {classesCelula(celula)}"
					>
						{#if celula.ehHoje}
							<span
								class="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-[11px] font-semibold text-primary-on"
							>
								{celula.dia}
							</span>
						{:else}
							<span
								class="text-xs font-medium sm:text-sm {celula.estado === 'indisponivel' || celula.estado === 'bloqueado'
									? 'text-mute'
									: 'text-ink'} {celula.estado === 'bloqueado' ? 'line-through' : ''}"
							>
								{celula.dia}
							</span>
						{/if}

						{#if celula.feriado}
							<span class="w-full truncate text-[10px] leading-tight text-accent-yellow">
								{celula.feriado}
							</span>
						{/if}

						<span class="mt-auto flex w-full flex-col">
							{#if celula.estado === 'personalizado'}
								<span class="truncate text-[10px] font-medium leading-tight text-accent-blue">próprio</span>
							{:else if celula.estado === 'bloqueado'}
								<span class="truncate text-[10px] font-medium leading-tight text-accent-red">bloqueado</span>
							{/if}
							{#each celula.blocos.slice(0, 3) as bloco (bloco.inicioMinutos)}
								<span class="hidden text-[10px] leading-tight text-mute sm:block">
									{horaCurta(bloco.inicioMinutos)}–{horaCurta(bloco.fimMinutos)}
								</span>
							{/each}
							{#if celula.blocos.length > 3}
								<span class="hidden text-[10px] leading-tight text-mute sm:block">
									+{celula.blocos.length - 3}
								</span>
							{/if}
						</span>
					</button>
				{:else}
					<div aria-hidden="true"></div>
				{/if}
			{/each}
		</div>

		<p class="mt-4 text-xs text-mute">
			Sábados, domingos e feriados são apenas destaque visual — você decide se atende neles. Dias
			passados não podem ser alterados, e o dia de hoje exige 24h de antecedência para qualquer
			mudança de horário.
		</p>
	</div>
</div>

{#if diaSelecionado}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
		role="presentation"
		onmousedown={(e) => {
			if (e.target === e.currentTarget) fecharModal();
		}}
	>
		<div
			role="dialog"
			aria-modal="true"
			aria-label="Editar {diaSelecionado.chave}"
			class="w-full max-w-md rounded-xl border border-hairline-strong bg-surface-card p-6"
		>
			<h3 class="text-lg font-semibold text-ink">{tituloDia}</h3>
			<p class="mt-1 text-sm text-mute">{rotuloEstado(diaSelecionado)}</p>

			{#if diaSelecionado.feriado}
				<p class="mt-2 flex items-center gap-1.5 text-sm text-body">
					<span class="h-2 w-2 rounded-full bg-accent-yellow"></span>
					Feriado: {diaSelecionado.feriado}
				</p>
			{/if}

			{#if erroModal}
				<div
					class="mt-4 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
					<span class="text-body">{erroModal}</span>
				</div>
			{/if}

			{#if !podeEditarHoje}
				<div
					class="mt-4 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-yellow"></span>
					<span class="text-body">{AVISO_ANTECEDENCIA}</span>
				</div>
			{/if}

			<div class="mt-5">
				<p class="text-sm font-medium text-ink">Horários do dia</p>

				<div class="mt-3 space-y-2">
					{#each blocosEdicao as bloco, index (index)}
						<div class="flex items-center gap-2">
							<TimeSelect
								ariaLabel="Início do bloco {index + 1}"
								bind:valor={blocosEdicao[index].inicioMinutos}
							/>
							<span class="text-mute">–</span>
							<TimeSelect
								ariaLabel="Fim do bloco {index + 1}"
								bind:valor={blocosEdicao[index].fimMinutos}
								minimo={15}
								maximo={24 * 60}
							/>
							<button
								type="button"
								onclick={() => removerBlocoEdicao(index)}
								class="text-sm text-mute transition hover:text-accent-red"
							>
								Remover
							</button>
						</div>
					{/each}
				</div>

				<button
					type="button"
					onclick={adicionarBlocoEdicao}
					class="mt-3 flex h-10 w-full items-center justify-center rounded-md border border-dashed border-hairline-strong text-sm font-medium text-ink transition hover:bg-surface-elevated"
				>
					+ Adicionar período
				</button>
			</div>

			<div class="mt-6 flex flex-wrap items-center gap-2">
				<button
					type="button"
					disabled={salvandoModal || !podeEditarHoje || blocosEdicao.length === 0}
					onclick={salvarHorarios}
					class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
				>
					{salvandoModal ? 'Salvando…' : 'Salvar horários'}
				</button>

				{#if diaSelecionado.estado !== 'bloqueado'}
					<button
						type="button"
						disabled={salvandoModal || !podeEditarHoje}
						onclick={marcarIndisponivel}
						class="inline-flex h-9 items-center rounded-md border border-hairline-strong px-4 text-sm font-medium text-accent-red transition hover:bg-surface-elevated disabled:cursor-not-allowed disabled:opacity-60"
					>
						Marcar indisponível
					</button>
				{/if}

				{#if diaSelecionado.origem !== 'padrao'}
					<button
						type="button"
						disabled={salvandoModal || !podeRestaurarPadrao}
						onclick={restaurarPadrao}
						class="inline-flex h-9 items-center rounded-md border border-hairline-strong px-4 text-sm font-medium text-ink transition hover:bg-surface-elevated disabled:cursor-not-allowed disabled:opacity-60"
					>
						Restaurar padrão
					</button>
				{/if}

				<button
					type="button"
					disabled={salvandoModal}
					onclick={fecharModal}
					class="ml-auto text-sm text-mute transition hover:text-ink"
				>
					Cancelar
				</button>
			</div>
		</div>
	</div>
{/if}
