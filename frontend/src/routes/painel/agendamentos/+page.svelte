<script lang="ts">
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import {
		cancelarAgendamento,
		confirmarAgendamento,
		listarAgendamentosDoCliente,
		listarAgendamentosDoPrestador,
		marcarNaoCompareceu,
		marcarRealizado,
		recusarAgendamento,
		type Agendamento
	} from '$lib/api/appointments';
	import { chaveData } from '$lib/holidays';
	import { dataLonga, minutosParaHHMM, rotuloStatus } from '$lib/format';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import LinhaAgendamento from '$lib/components/LinhaAgendamento.svelte';

	let { data }: { data: PageData } = $props();

	// svelte-ignore state_referenced_locally
	let agendamentos = $state<Agendamento[]>(data.agendamentos);
	// svelte-ignore state_referenced_locally
	let total = $state(data.total);
	let agindo = $state<string | null>(null);
	let carregandoMais = $state(false);
	let erro = $state<string | null>(null);

	// O tipo do usuário não muda dentro da sessão da página — capturar o valor
	// inicial é intencional.
	// svelte-ignore state_referenced_locally
	const ehPrestador = data.tipo === 'provider';
	const chaveHoje = chaveData(new Date());

	const pendentes = $derived(agendamentos.filter((a) => a.status === 'SOLICITADO'));
	const confirmados = $derived(agendamentos.filter((a) => a.status === 'CONFIRMADO'));
	const historico = $derived(
		agendamentos.filter((a) => a.status !== 'SOLICITADO' && a.status !== 'CONFIRMADO')
	);

	// jaComecou marca confirmados cujo horário já chegou — habilita o
	// desfecho (realizado / não compareceu) para o prestador.
	function jaComecou(a: Agendamento): boolean {
		if (a.data < chaveHoje) return true;
		if (a.data > chaveHoje) return false;
		const agora = new Date();
		return a.inicioMinutos <= agora.getHours() * 60 + agora.getMinutes();
	}

	async function executar(id: string, acao: (id: string) => Promise<void>) {
		if (agindo) return;
		erro = null;
		agindo = id;
		try {
			await acao(id);
			// recarrega mantendo a mesma quantidade já visível — sem isso, uma
			// ação sobre um item além da primeira página encolheria a lista de
			// volta ao padrão da API
			const pagina = { limite: agendamentos.length };
			const resposta = ehPrestador
				? await listarAgendamentosDoPrestador(pagina)
				: await listarAgendamentosDoCliente(pagina);
			agendamentos = resposta.agendamentos;
			total = resposta.total;
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível concluir a ação.';
		} finally {
			agindo = null;
		}
	}

	// O offset é o que já está carregado; o limite fica a cargo do padrão da API.
	async function carregarMais() {
		carregandoMais = true;
		try {
			const resposta = ehPrestador
				? await listarAgendamentosDoPrestador({ offset: agendamentos.length })
				: await listarAgendamentosDoCliente({ offset: agendamentos.length });
			agendamentos = [...agendamentos, ...resposta.agendamentos];
			total = resposta.total;
		} finally {
			carregandoMais = false;
		}
	}

	let copiadoId = $state<string | null>(null);

	// mensagemParaCliente monta um texto pronto com data, hora e observação —
	// só faz sentido para marcações do prestador, que não disparam email.
	function mensagemParaCliente(a: Agendamento): string {
		const linhas = [
			`Agendamento confirmado para ${a.nomeCliente ?? 'você'}:`,
			`${dataLonga(a.data)}, das ${minutosParaHHMM(a.inicioMinutos)} às ${minutosParaHHMM(a.fimMinutos)}.`
		];
		if (a.observacao) {
			linhas.push(`Observação: ${a.observacao}`);
		}
		return linhas.join('\n');
	}

	async function copiarMensagem(a: Agendamento) {
		try {
			await navigator.clipboard.writeText(mensagemParaCliente(a));
			copiadoId = a.id;
			setTimeout(() => {
				if (copiadoId === a.id) copiadoId = null;
			}, 2000);
		} catch {
			erro = 'Não foi possível copiar — copie manualmente pela seleção do texto.';
		}
	}

	// O preenchimento branco é a ação primária da página; repetido em cada linha
	// da lista ele competiria com o conteúdo. Ação de linha usa contorno, e a cor
	// carrega o significado.
	const botaoBase =
		'inline-flex h-8 items-center rounded-md border px-3 text-xs font-medium transition disabled:cursor-not-allowed disabled:opacity-60';
	const botaoPrimario = `${botaoBase} border-accent-green/45 text-accent-green hover:bg-accent-green/10`;
	const botaoContorno = `${botaoBase} border-hairline-strong text-ink hover:bg-surface-elevated`;
	const botaoPerigo = `${botaoBase} border-hairline-strong text-mute hover:border-accent-red/45 hover:text-accent-red`;

	// ---- Agrupamento por dia ----

	type Grupo = { chave: string; rotulo: string; ehHoje: boolean; itens: Agendamento[] };

	type Filtro = 'tudo' | 'aguardando' | 'confirmados' | 'historico';
	let filtro = $state<Filtro>('tudo');

	const visiveis = $derived.by(() => {
		switch (filtro) {
			case 'aguardando':
				return pendentes;
			case 'confirmados':
				return confirmados;
			case 'historico':
				return historico;
			default:
				return agendamentos;
		}
	});

	function rotuloDia(data: string): string {
		const [ano, mes, dia] = data.split('-').map(Number);
		const texto = new Intl.DateTimeFormat('pt-BR', {
			weekday: 'long',
			day: 'numeric',
			month: 'long'
		}).format(new Date(ano, mes - 1, dia));
		return texto.charAt(0).toUpperCase() + texto.slice(1);
	}

	// Dias futuros em ordem crescente (o mais próximo primeiro) e, depois deles,
	// os dias passados em ordem decrescente — é como se lê uma agenda: o que vem
	// primeiro, e só então o que já aconteceu.
	const grupos = $derived.by<Grupo[]>(() => {
		const porDia = new Map<string, Agendamento[]>();
		for (const a of visiveis) {
			const lista = porDia.get(a.data) ?? [];
			lista.push(a);
			porDia.set(a.data, lista);
		}

		const futuros: string[] = [];
		const passados: string[] = [];
		for (const chave of porDia.keys()) {
			(chave >= chaveHoje ? futuros : passados).push(chave);
		}
		futuros.sort();
		passados.sort().reverse();

		return [...futuros, ...passados].map((chave) => ({
			chave,
			rotulo: rotuloDia(chave),
			ehHoje: chave === chaveHoje,
			itens: (porDia.get(chave) ?? []).sort((x, y) => x.inicioMinutos - y.inicioMinutos)
		}));
	});

</script>

{#snippet acoesDe(a: Agendamento)}
	<div class="mt-2.5 flex flex-wrap gap-2 empty:hidden">
		{#if ehPrestador && a.marcadoPeloPrestador && a.status === 'CONFIRMADO'}
			<button type="button" onclick={() => copiarMensagem(a)} class={botaoContorno}>
				{copiadoId === a.id ? 'Copiado!' : 'Copiar para o cliente'}
			</button>
		{/if}

		{#if ehPrestador && a.status === 'SOLICITADO'}
			<button
				type="button"
				disabled={agindo !== null}
				onclick={() => executar(a.id, confirmarAgendamento)}
				class={botaoPrimario}
			>
				Confirmar
			</button>
			<button
				type="button"
				disabled={agindo !== null}
				onclick={() => executar(a.id, recusarAgendamento)}
				class={botaoPerigo}
			>
				Cancelar
			</button>
		{/if}

		{#if !ehPrestador && a.status === 'SOLICITADO'}
			<button
				type="button"
				disabled={agindo !== null}
				onclick={() => executar(a.id, cancelarAgendamento)}
				class={botaoPerigo}
			>
				Cancelar solicitação
			</button>
		{/if}

		{#if a.status === 'CONFIRMADO' && (a.marcadoPeloPrestador || !jaComecou(a))}
			<button
				type="button"
				disabled={agindo !== null}
				onclick={() => executar(a.id, cancelarAgendamento)}
				class={botaoPerigo}
			>
				Cancelar
			</button>
		{/if}

		{#if ehPrestador && a.status === 'CONFIRMADO' && jaComecou(a)}
			<button
				type="button"
				disabled={agindo !== null}
				onclick={() => executar(a.id, marcarRealizado)}
				class={botaoPrimario}
			>
				Realizado
			</button>
			<button
				type="button"
				disabled={agindo !== null}
				onclick={() => executar(a.id, marcarNaoCompareceu)}
				class={botaoContorno}
			>
				Não compareceu
			</button>
		{/if}
	</div>
{/snippet}

{#snippet chip(valor: Filtro, texto: string, total: number, cor?: string)}
	<button
		type="button"
		onclick={() => (filtro = valor)}
		aria-pressed={filtro === valor}
		class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs transition {filtro ===
		valor
			? 'border-hairline-strong bg-surface-elevated font-medium text-ink'
			: 'border-hairline text-mute hover:text-ink'}"
	>
		{#if cor}
			<span class="h-1.5 w-1.5 rounded-full {cor}"></span>
		{/if}
		{texto}
		<span class="tabular-nums">{total}</span>
	</button>
{/snippet}

<div>
	<PageHeader
		titulo="Agendamentos"
		descricao={ehPrestador
			? 'Sua agenda em ordem cronológica — a faixa colorida indica o estado de cada horário.'
			: 'Seus atendimentos em ordem cronológica, do próximo ao mais antigo.'}
	/>

	{#if erro}
		<div
			class="mb-6 flex items-start gap-2 rounded-md border border-accent-red/40 bg-accent-red/10 p-3 text-sm"
		>
			<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
			<span class="text-body">{erro}</span>
		</div>
	{/if}

	{#if agendamentos.length === 0}
		<div class="rounded-xl border border-hairline-strong bg-surface-card px-5 py-8 text-center">
			<p class="text-sm text-body">
				{ehPrestador
					? 'Nenhum agendamento recebido ainda. Compartilhe seu link de agendamento para começar.'
					: 'Você ainda não tem agendamentos.'}
			</p>
			<a
				href={ehPrestador ? '/painel' : '/painel/agendar'}
				class="mt-4 inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90"
			>
				{ehPrestador ? 'Ver meu link' : 'Agendar agora'}
			</a>
		</div>
	{:else}
		<div class="mb-5 flex flex-wrap gap-2">
			{@render chip('tudo', 'Tudo', agendamentos.length)}
			{@render chip('aguardando', 'Aguardando', pendentes.length, 'bg-accent-yellow')}
			{@render chip('confirmados', 'Confirmados', confirmados.length, 'bg-accent-green')}
			{@render chip('historico', 'Histórico', historico.length)}
		</div>

		{#if grupos.length === 0}
			<p
				class="rounded-xl border border-hairline-strong bg-surface-card px-4 py-8 text-center text-sm text-mute"
			>
				Nada neste filtro.
			</p>
		{/if}

		{#each grupos as grupo (grupo.chave)}
			<section class="mb-5">
				<div class="mb-2 flex flex-wrap items-baseline gap-2">
					<h2 class="text-sm font-semibold text-ink">{grupo.rotulo}</h2>
					{#if grupo.ehHoje}
						<span
							class="rounded-full border border-accent-green/40 bg-accent-green/10 px-2 py-0.5 text-[11px] tracking-wide text-accent-green uppercase"
						>
							Hoje
						</span>
					{/if}
					<span class="text-xs text-mute">
						{grupo.itens.length}
						{grupo.itens.length === 1 ? 'atendimento' : 'atendimentos'}
					</span>
				</div>

				<ul
					class="divide-y divide-hairline overflow-hidden rounded-xl border border-hairline-strong bg-surface-card"
				>
					{#each grupo.itens as a (a.id)}
						<LinhaAgendamento agendamento={a} {ehPrestador}>
							{#snippet extra()}
								{#if ehPrestador && a.marcadoPeloPrestador && a.status === 'CONFIRMADO'}
									<!-- Marcação feita pelo prestador não dispara email: a mensagem
									     pronta fica à vista para ele mandar pelo canal que quiser. -->
									<p
										class="mt-2 border-l-2 border-hairline-strong pl-3 text-xs whitespace-pre-wrap text-mute"
									>
										{mensagemParaCliente(a)}
									</p>
								{/if}
							{/snippet}
							{#snippet acoes()}
								{@render acoesDe(a)}
							{/snippet}
						</LinhaAgendamento>
					{/each}
				</ul>
			</section>
		{/each}

		{#if agendamentos.length < total}
			<div class="mt-2 flex justify-center">
				<button
					type="button"
					onclick={carregarMais}
					disabled={carregandoMais}
					class="rounded-full border border-hairline-strong bg-surface-card px-4 py-1.5 text-sm text-body transition hover:bg-surface-elevated/40 disabled:opacity-60"
				>
					{carregandoMais ? 'Carregando…' : 'Carregar mais'}
				</button>
			</div>
		{/if}
	{/if}
</div>
