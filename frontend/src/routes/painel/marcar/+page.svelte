<script lang="ts">
	import { ApiError } from '$lib/api/client';
	import {
		consultarSlotsDoPrestador,
		marcarPeloPrestador,
		type DiaSlots,
		type Slot
	} from '$lib/api/appointments';
	import { minutosParaHHMM, dataLonga } from '$lib/format';
	import { chaveData, feriadosNacionais } from '$lib/holidays';

	// Quantos meses à frente o prestador pode navegar no calendário.
	const MESES_FUTUROS = 2;

	// ---- Calendário de horários livres (da própria agenda) ----

	interface Celula {
		dia: number;
		chave: string;
		slots: Slot[];
		feriado: string | null;
		ehHoje: boolean;
		passado: boolean;
	}

	const hoje = new Date();
	const chaveHoje = chaveData(hoje);

	function chaveMes(mes: Date): string {
		return `${mes.getFullYear()}-${(mes.getMonth() + 1).toString().padStart(2, '0')}`;
	}

	let mesExibido = $state(new Date(hoje.getFullYear(), hoje.getMonth(), 1));
	let slotsPorMes = $state<Record<string, DiaSlots[]>>({});
	let carregandoMes = $state(false);
	let erro = $state<string | null>(null);
	let diaSelecionado = $state<string | null>(null);
	let slotSelecionado = $state<Slot | null>(null);
	let enviando = $state(false);
	let marcado = $state(false);

	// Registro puramente interno: só nome e uma observação livre e opcional.
	let clienteNome = $state('');
	let observacao = $state('');

	const clienteValido = $derived(clienteNome.trim().length >= 2);

	const ehMesAtual = $derived(
		mesExibido.getFullYear() === hoje.getFullYear() && mesExibido.getMonth() === hoje.getMonth()
	);
	const limiteFuturo = new Date(hoje.getFullYear(), hoje.getMonth() + MESES_FUTUROS, 1);
	const noLimiteFuturo = $derived(mesExibido >= limiteFuturo);

	const rotuloMes = $derived.by(() => {
		const rotulo = new Intl.DateTimeFormat('pt-BR', { month: 'long', year: 'numeric' }).format(mesExibido);
		return rotulo.charAt(0).toUpperCase() + rotulo.slice(1);
	});

	async function carregarMes(mes: Date) {
		const chave = chaveMes(mes);
		if (slotsPorMes[chave]) return;

		carregandoMes = true;
		erro = null;
		try {
			const de = chaveData(new Date(mes.getFullYear(), mes.getMonth(), 1));
			const ate = chaveData(new Date(mes.getFullYear(), mes.getMonth() + 1, 0));
			const resposta = await consultarSlotsDoPrestador(de, ate);
			slotsPorMes = { ...slotsPorMes, [chave]: resposta.dias };
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível carregar os horários.';
		} finally {
			carregandoMes = false;
		}
	}

	// primeiro carregamento do mês atual
	$effect(() => {
		carregarMes(mesExibido);
	});

	function mudarMes(delta: number) {
		mesExibido = new Date(mesExibido.getFullYear(), mesExibido.getMonth() + delta, 1);
		diaSelecionado = null;
		slotSelecionado = null;
	}

	const celulas = $derived.by<(Celula | null)[]>(() => {
		const dias = slotsPorMes[chaveMes(mesExibido)];
		if (!dias) return [];

		const ano = mesExibido.getFullYear();
		const mes = mesExibido.getMonth();
		const feriados = feriadosNacionais(ano);
		const porChave = new Map(dias.map((d) => [d.data, d.slots]));
		const resultado: (Celula | null)[] = [];

		for (let i = 0; i < new Date(ano, mes, 1).getDay(); i++) {
			resultado.push(null);
		}

		const totalDias = new Date(ano, mes + 1, 0).getDate();
		for (let dia = 1; dia <= totalDias; dia++) {
			const chave = chaveData(new Date(ano, mes, dia));
			resultado.push({
				dia,
				chave,
				slots: porChave.get(chave) ?? [],
				feriado: feriados.get(chave) ?? null,
				ehHoje: chave === chaveHoje,
				passado: chave < chaveHoje
			});
		}
		return resultado;
	});

	const slotsDoDia = $derived(
		diaSelecionado
			? (slotsPorMes[chaveMes(mesExibido)]?.find((d) => d.data === diaSelecionado)?.slots ?? [])
			: []
	);

	function escolherDia(celula: Celula) {
		if (celula.passado || celula.slots.length === 0) return;
		diaSelecionado = celula.chave;
		slotSelecionado = null;
		marcado = false;
		erro = null;
	}

	// removeSlotOfertado tira da oferta local o horário recém-marcado.
	function removeSlotOfertado(inicioMinutos: number) {
		const chave = chaveMes(mesExibido);
		slotsPorMes = {
			...slotsPorMes,
			[chave]: slotsPorMes[chave].map((d) =>
				d.data === diaSelecionado
					? { ...d, slots: d.slots.filter((s) => s.inicioMinutos !== inicioMinutos) }
					: d
			)
		};
	}

	async function marcar(evento: SubmitEvent) {
		evento.preventDefault();
		if (!diaSelecionado || !slotSelecionado || !clienteValido) return;
		erro = null;
		enviando = true;

		try {
			await marcarPeloPrestador({
				data: diaSelecionado,
				inicioMinutos: slotSelecionado.inicioMinutos,
				nome: clienteNome.trim(),
				observacao: observacao.trim() || undefined
			});
			marcado = true;
			removeSlotOfertado(slotSelecionado.inicioMinutos);
			slotSelecionado = null;
			clienteNome = '';
			observacao = '';
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível marcar o agendamento.';
		} finally {
			enviando = false;
		}
	}

	const inputClasse =
		'h-10 w-full rounded-md border border-hairline-strong bg-surface-elevated px-3 text-sm text-ink outline-none transition focus:border-ink';
</script>

<div class="mx-auto max-w-2xl">
	<a href="/painel" class="text-sm text-mute transition hover:text-ink">← Voltar ao painel</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Marcar para um cliente</h1>
	<p class="mt-3 text-body">
		Cliente ligou? Registre a marcação você mesmo: escolha um horário livre e informe o contato.
		A marcação entra como solicitação — confirme em
		<a href="/painel/agendamentos" class="font-medium text-ink underline">Agendamentos</a>.
	</p>

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-5 sm:p-8">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h2 class="text-lg font-semibold text-ink">Escolha o dia</h2>

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
					disabled={noLimiteFuturo}
					onclick={() => mudarMes(1)}
					class="flex h-8 w-8 items-center justify-center rounded-md border border-hairline-strong text-ink transition hover:bg-surface-elevated disabled:cursor-not-allowed disabled:opacity-40"
				>
					›
				</button>
			</div>
		</div>

		{#if erro}
			<div
				class="mt-4 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
			>
				<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
				<span class="text-body">{erro}</span>
			</div>
		{/if}

		<div class="mt-4 grid grid-cols-7 gap-1 sm:gap-1.5 {carregandoMes ? 'opacity-60' : ''}">
			{#each ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'] as nome, i (nome)}
				<div class="pb-1 text-center text-xs font-medium {i === 0 || i === 6 ? 'text-ash' : 'text-mute'}">
					{nome}
				</div>
			{/each}

			{#each celulas as celula, i (celula?.chave ?? `vazio-${i}`)}
				{#if celula}
					{@const disponivel = celula.slots.length > 0}
					<button
						type="button"
						disabled={celula.passado || !disponivel}
						data-dia={celula.chave}
						data-livres={celula.slots.length}
						title={celula.feriado ? `Feriado: ${celula.feriado}` : undefined}
						onclick={() => escolherDia(celula)}
						class="relative flex aspect-square flex-col items-start gap-0.5 overflow-hidden rounded-md border p-1 text-left transition sm:p-1.5
							{diaSelecionado === celula.chave
							? 'border-ink bg-surface-elevated'
							: disponivel
								? 'border-hairline-strong bg-surface-elevated hover:border-ink'
								: 'border-hairline'}
							{celula.passado ? 'cursor-not-allowed opacity-35' : ''}
							{!disponivel && !celula.passado ? 'cursor-not-allowed opacity-60' : ''}"
					>
						{#if celula.ehHoje}
							<span
								class="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-[11px] font-semibold text-primary-on"
							>
								{celula.dia}
							</span>
						{:else}
							<span class="text-xs font-medium sm:text-sm {disponivel ? 'text-ink' : 'text-mute'}">
								{celula.dia}
							</span>
						{/if}

						{#if celula.feriado}
							<span class="w-full truncate text-[10px] leading-tight text-accent-yellow">
								{celula.feriado}
							</span>
						{/if}

						{#if disponivel}
							<span class="mt-auto flex items-center gap-1 text-[10px] leading-tight text-mute">
								<span class="h-1.5 w-1.5 shrink-0 rounded-full bg-accent-green"></span>
								<span class="hidden sm:block">
									{celula.slots.length}
									{celula.slots.length === 1 ? 'horário' : 'horários'}
								</span>
							</span>
						{/if}
					</button>
				{:else}
					<div aria-hidden="true"></div>
				{/if}
			{/each}
		</div>

		{#if diaSelecionado}
			<h2 class="mt-8 text-lg font-semibold text-ink">Escolha o horário</h2>
			<p class="mt-1 text-sm text-mute">{dataLonga(diaSelecionado)}</p>

			<div class="mt-3 flex flex-wrap gap-2">
				{#each slotsDoDia as slot (slot.inicioMinutos)}
					<button
						type="button"
						data-slot={slot.inicioMinutos}
						onclick={() => (slotSelecionado = slot)}
						class="rounded-md border px-3.5 py-2 text-sm font-medium transition {slotSelecionado?.inicioMinutos ===
						slot.inicioMinutos
							? 'border-ink bg-surface-elevated text-ink'
							: 'border-hairline-strong text-body hover:bg-surface-elevated'}"
					>
						{minutosParaHHMM(slot.inicioMinutos)}
					</button>
				{/each}
			</div>
		{/if}

		{#if marcado}
			<div
				class="mt-6 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
			>
				<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-green"></span>
				<span class="text-body">
					Marcação registrada! Confirme-a em
					<a href="/painel/agendamentos" class="font-medium text-ink underline">Agendamentos</a>.
				</span>
			</div>
		{/if}

		{#if slotSelecionado && diaSelecionado}
			<div class="mt-6 rounded-md border border-hairline bg-surface-elevated p-4">
				<p class="text-sm text-body">
					{dataLonga(diaSelecionado)}, das
					<span class="font-medium text-ink">{minutosParaHHMM(slotSelecionado.inicioMinutos)}</span>
					às
					<span class="font-medium text-ink">{minutosParaHHMM(slotSelecionado.fimMinutos)}</span>
				</p>

				<form class="mt-4 space-y-3" onsubmit={marcar}>
					<p class="text-sm text-body">
						Registro interno: o cliente não recebe nenhuma notificação desta marcação.
					</p>

					<label class="block">
						<span class="mb-1 block text-xs font-medium text-mute">Nome</span>
						<input
							type="text"
							bind:value={clienteNome}
							required
							autocomplete="name"
							placeholder="Nome do cliente"
							class={inputClasse}
						/>
					</label>
					<label class="block">
						<span class="mb-1 block text-xs font-medium text-mute">Observação (opcional)</span>
						<textarea
							bind:value={observacao}
							rows="3"
							placeholder="Ex.: prefere corte curto, já é cliente antigo…"
							class="{inputClasse} h-auto resize-none py-2"
						></textarea>
					</label>

					<button
						type="submit"
						disabled={enviando || !clienteValido}
						class="inline-flex h-10 items-center rounded-md bg-primary px-5 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
					>
						{enviando ? 'Marcando…' : 'Marcar horário'}
					</button>
				</form>
			</div>
		{/if}
	</div>

	<p class="mt-4 text-xs text-mute">
		Seus horários aparecem aqui mesmo com a agenda fechada ao público — dias bloqueados e fora do
		expediente continuam indisponíveis.
	</p>
</div>
