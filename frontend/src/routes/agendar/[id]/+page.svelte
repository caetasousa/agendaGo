<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import {
		consultarSlots,
		solicitarAgendamento,
		solicitarConvidado,
		type DiaSlots,
		type Slot
	} from '$lib/api/appointments';
	import { chaveData, feriadosNacionais } from '$lib/holidays';
	import { sessao } from '$lib/stores/session.svelte';

	let { data }: { data: PageData } = $props();

	// Quantos meses à frente o cliente pode navegar no calendário.
	const MESES_FUTUROS = 2;

	function minutosParaHHMM(minutos: number): string {
		const h = Math.floor(minutos / 60)
			.toString()
			.padStart(2, '0');
		const m = (minutos % 60).toString().padStart(2, '0');
		return `${h}:${m}`;
	}

	// ---- Calendário de horários livres ----

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
	let solicitado = $state(false);

	// Agendamento sem cadastro: o visitante informa nome, email e telefone.
	let convidadoNome = $state('');
	let convidadoEmail = $state('');
	let convidadoTelefone = $state('');

	// Nota livre e opcional, visível ao prestador — usada tanto pelo cliente
	// autenticado quanto pelo convidado.
	let observacao = $state('');

	// Conta os dígitos do telefone para a mesma validação leve do backend (>= 8).
	const telefoneValido = $derived((convidadoTelefone.match(/\d/g) ?? []).length >= 8);
	const convidadoValido = $derived(
		convidadoNome.trim().length >= 2 && /\S+@\S+\.\S+/.test(convidadoEmail) && telefoneValido
	);

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
			const resposta = await consultarSlots(data.prestador.id, de, ate);
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
		solicitado = false;
		erro = null;
	}

	// removeSlotOfertado tira da oferta local o horário recém-reservado.
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

	async function solicitar() {
		if (!diaSelecionado || !slotSelecionado) return;
		erro = null;
		enviando = true;

		try {
			await solicitarAgendamento({
				providerId: data.prestador.id,
				data: diaSelecionado,
				inicioMinutos: slotSelecionado.inicioMinutos,
				observacao: observacao.trim() || undefined
			});
			solicitado = true;
			removeSlotOfertado(slotSelecionado.inicioMinutos);
			slotSelecionado = null;
			observacao = '';
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível solicitar o agendamento.';
		} finally {
			enviando = false;
		}
	}

	async function solicitarSemCadastro() {
		if (!diaSelecionado || !slotSelecionado || !convidadoValido) return;
		erro = null;
		enviando = true;

		try {
			await solicitarConvidado({
				providerId: data.prestador.id,
				data: diaSelecionado,
				inicioMinutos: slotSelecionado.inicioMinutos,
				nome: convidadoNome.trim(),
				email: convidadoEmail.trim(),
				telefone: convidadoTelefone.trim(),
				observacao: observacao.trim() || undefined
			});
			solicitado = true;
			removeSlotOfertado(slotSelecionado.inicioMinutos);
			slotSelecionado = null;
			observacao = '';
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível solicitar o agendamento.';
		} finally {
			enviando = false;
		}
	}

	function irParaLogin() {
		goto(`/login?voltar=${encodeURIComponent(page.url.pathname)}`);
	}

	function dataLonga(data: string): string {
		const [ano, mes, dia] = data.split('-').map(Number);
		const rotulo = new Intl.DateTimeFormat('pt-BR', {
			weekday: 'long',
			day: 'numeric',
			month: 'long'
		}).format(new Date(ano, mes - 1, dia));
		return rotulo.charAt(0).toUpperCase() + rotulo.slice(1);
	}
</script>

<div class="mx-auto max-w-2xl">
	<span
		class="inline-flex items-center gap-2 rounded-full border border-hairline-strong bg-surface-elevated px-3 py-1 text-xs text-body"
	>
		<span class="h-2 w-2 rounded-full bg-accent-green"></span>
		Agendamento online
	</span>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">{data.prestador.nome}</h1>
	<p class="mt-3 text-body">
		Atendimento de {data.prestador.duracaoAtendimentoMinutos} minutos. Escolha um dia com horários
		livres e solicite — a confirmação chega em até 24h.
	</p>

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-5 sm:p-8">
		{#if !data.prestador.aceitaAgendamentos}
			<div
				class="flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
			>
				<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-yellow"></span>
				<span class="text-body">Este prestador não está aceitando agendamentos no momento.</span>
			</div>
		{:else}
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

			{#if solicitado}
				<div
					class="mt-6 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-green"></span>
					<span class="text-body">
						Solicitação enviada! {data.prestador.nome} tem 24h para confirmar{#if sessao.usuario?.tipo === 'client'} — acompanhe
							em
							<a href="/painel/agendamentos" class="font-medium text-ink underline">Meus agendamentos</a
							>{:else}. Você receberá o retorno pelo contato informado{/if}.
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
						com <span class="font-medium text-ink">{data.prestador.nome}</span>
					</p>

					{#if sessao.usuario?.tipo === 'client'}
						<label class="mt-4 block">
							<span class="mb-1 block text-xs font-medium text-mute">Observação (opcional)</span>
							<textarea
								bind:value={observacao}
								rows="3"
								placeholder="Alguma informação para o prestador?"
								class="h-auto w-full resize-none rounded-md border border-hairline-strong bg-surface-elevated px-3 py-2 text-sm text-ink outline-none transition focus:border-ink"
							></textarea>
						</label>
						<button
							type="button"
							disabled={enviando}
							onclick={solicitar}
							class="mt-3 inline-flex h-10 items-center rounded-md bg-primary px-5 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
						>
							{enviando ? 'Enviando…' : 'Solicitar agendamento'}
						</button>
					{:else if sessao.usuario?.tipo === 'provider'}
						<p class="mt-3 text-sm text-mute">
							Você está logado como prestador — entre com uma conta de cliente para agendar.
						</p>
					{:else if !sessao.carregando}
						<form
							class="mt-4 space-y-3"
							onsubmit={(e) => {
								e.preventDefault();
								solicitarSemCadastro();
							}}
						>
							<p class="text-sm text-body">
								Agende sem criar conta — só precisamos do seu contato para o prestador confirmar.
							</p>

							<div class="grid gap-3 sm:grid-cols-3">
								<label class="block">
									<span class="mb-1 block text-xs font-medium text-mute">Nome</span>
									<input
										type="text"
										bind:value={convidadoNome}
										required
										autocomplete="name"
										placeholder="Seu nome"
										class="h-10 w-full rounded-md border border-hairline-strong bg-surface-elevated px-3 text-sm text-ink outline-none transition focus:border-ink"
									/>
								</label>
								<label class="block">
									<span class="mb-1 block text-xs font-medium text-mute">E-mail</span>
									<input
										type="email"
										bind:value={convidadoEmail}
										required
										autocomplete="email"
										placeholder="voce@email.com"
										class="h-10 w-full rounded-md border border-hairline-strong bg-surface-elevated px-3 text-sm text-ink outline-none transition focus:border-ink"
									/>
								</label>
								<label class="block">
									<span class="mb-1 block text-xs font-medium text-mute">Telefone</span>
									<input
										type="tel"
										bind:value={convidadoTelefone}
										required
										autocomplete="tel"
										placeholder="(11) 99999-8888"
										class="h-10 w-full rounded-md border border-hairline-strong bg-surface-elevated px-3 text-sm text-ink outline-none transition focus:border-ink"
									/>
								</label>
							</div>

							<label class="block">
								<span class="mb-1 block text-xs font-medium text-mute">Observação (opcional)</span>
								<textarea
									bind:value={observacao}
									rows="3"
									placeholder="Alguma informação para o prestador?"
									class="h-auto w-full resize-none rounded-md border border-hairline-strong bg-surface-elevated px-3 py-2 text-sm text-ink outline-none transition focus:border-ink"
								></textarea>
							</label>

							<div class="flex flex-wrap items-center gap-3">
								<button
									type="submit"
									disabled={enviando || !convidadoValido}
									class="inline-flex h-10 items-center rounded-md bg-primary px-5 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
								>
									{enviando ? 'Enviando…' : 'Solicitar agendamento'}
								</button>
								<button type="button" onclick={irParaLogin} class="text-sm text-body">
									Já tem conta? <span class="font-medium text-ink underline">Entrar</span>
								</button>
							</div>
						</form>
					{/if}
				</div>
			{/if}
		{/if}
	</div>

	<p class="mt-4 text-xs text-mute">
		Horários já consideram a duração do atendimento e o intervalo de preparação do prestador.
	</p>
</div>
