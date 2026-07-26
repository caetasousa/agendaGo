<script lang="ts">
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import {
		banirCliente,
		banirPrestador,
		listarClientes,
		listarPrestadores,
		reativarCliente,
		reativarPrestador,
		type UsuarioModeracao
	} from '$lib/api/admin';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Indicadores from '$lib/components/Indicadores.svelte';

	let { data }: { data: PageData } = $props();

	// svelte-ignore state_referenced_locally
	let prestadores = $state<UsuarioModeracao[]>(data.prestadores);
	// svelte-ignore state_referenced_locally
	let clientes = $state<UsuarioModeracao[]>(data.clientes);
	// svelte-ignore state_referenced_locally
	let totalPrestadores = $state(data.totalPrestadores);
	// svelte-ignore state_referenced_locally
	let totalClientes = $state(data.totalClientes);
	let agindo = $state<string | null>(null);
	let carregandoMais = $state<'prestadores' | 'clientes' | null>(null);
	let erro = $state<string | null>(null);

	const temMaisPrestadores = $derived(prestadores.length < totalPrestadores);
	const temMaisClientes = $derived(clientes.length < totalClientes);

	async function recarregar() {
		// mantém a mesma quantidade já visível de cada lista — sem isso, uma
		// ação sobre um item além da primeira página encolheria a lista de volta
		// ao padrão da API
		const [ps, cs] = await Promise.all([
			listarPrestadores({ limite: prestadores.length || undefined }),
			listarClientes({ limite: clientes.length || undefined })
		]);
		prestadores = ps.usuarios;
		clientes = cs.usuarios;
		totalPrestadores = ps.total;
		totalClientes = cs.total;
	}

	async function executar(id: string, acao: (id: string) => Promise<void>) {
		if (agindo) return;
		erro = null;
		agindo = id;
		try {
			await acao(id);
			await recarregar();
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível concluir a ação.';
		} finally {
			agindo = null;
		}
	}

	// O offset é o que já está carregado; o limite fica a cargo do padrão da API.
	async function carregarMaisPrestadores() {
		carregandoMais = 'prestadores';
		try {
			const resposta = await listarPrestadores({ offset: prestadores.length });
			prestadores = [...prestadores, ...resposta.usuarios];
			totalPrestadores = resposta.total;
		} finally {
			carregandoMais = null;
		}
	}

	async function carregarMaisClientes() {
		carregandoMais = 'clientes';
		try {
			const resposta = await listarClientes({ offset: clientes.length });
			clientes = [...clientes, ...resposta.usuarios];
			totalClientes = resposta.total;
		} finally {
			carregandoMais = null;
		}
	}

	const totalPrestadoresAtivos = $derived(prestadores.filter((p) => p.ativo).length);
	const totalClientesAtivos = $derived(clientes.filter((c) => c.ativo).length);
</script>

{#snippet linha(
	u: UsuarioModeracao,
	base: string,
	banir: (id: string) => Promise<void>,
	reativar: (id: string) => Promise<void>
)}
	<li
		data-usuario={u.id}
		data-ativo={u.ativo}
		class="flex flex-wrap items-center justify-between gap-3 px-4 py-3 {u.ativo ? '' : 'opacity-70'}"
	>
		<a href="{base}/{u.id}" class="group min-w-0" data-detalhe={u.id}>
			<p class="flex items-center gap-2 text-sm font-medium text-ink group-hover:underline">
				{u.nome}
				{#if !u.ativo}
					<span
						class="inline-flex items-center gap-1 rounded-full border border-accent-red/40 bg-accent-red/10 px-2 py-0.5 text-[11px] font-medium text-accent-red"
					>
						Banido
					</span>
				{/if}
			</p>
			<p class="truncate text-xs text-mute">{u.email}</p>
		</a>

		{#if u.ativo}
			<button
				type="button"
				disabled={agindo !== null}
				onclick={() => executar(u.id, banir)}
				class="inline-flex h-8 items-center rounded-md border border-hairline-strong px-3 text-xs font-medium text-accent-red transition hover:bg-surface-elevated disabled:cursor-not-allowed disabled:opacity-60"
			>
				Banir
			</button>
		{:else}
			<button
				type="button"
				disabled={agindo !== null}
				onclick={() => executar(u.id, reativar)}
				class="inline-flex h-8 items-center rounded-md bg-primary px-3 text-xs font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				Reativar
			</button>
		{/if}
	</li>
{/snippet}

{#snippet carregarMais(temMais: boolean, carregando: boolean, acao: () => Promise<void>)}
	{#if temMais}
		<div class="mt-3 flex justify-center">
			<button
				type="button"
				onclick={acao}
				disabled={carregando}
				class="rounded-full border border-hairline-strong bg-surface-card px-4 py-1.5 text-sm text-body transition hover:bg-surface-elevated/40 disabled:opacity-60"
			>
				{carregando ? 'Carregando…' : 'Carregar mais'}
			</button>
		</div>
	{/if}
{/snippet}

<div>
	<PageHeader
		titulo="Moderação"
		descricao="Banir remove o acesso do usuário e o tira da vitrine; o histórico de agendamentos é preservado. Reativar devolve o acesso."
	/>

	{#if erro}
		<div
			class="mb-6 flex items-start gap-2 rounded-md border border-accent-red/40 bg-accent-red/10 p-3 text-sm"
		>
			<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
			<span class="text-body">{erro}</span>
		</div>
	{/if}

	<Indicadores
		itens={[
			{ rotulo: 'Prestadores', valor: totalPrestadores },
			{ rotulo: 'Prestadores ativos', valor: totalPrestadoresAtivos },
			{ rotulo: 'Clientes', valor: totalClientes },
			{ rotulo: 'Clientes ativos', valor: totalClientesAtivos }
		]}
	/>

	<section class="mt-6">
		<div class="mb-2 flex items-baseline justify-between gap-3">
			<h2 class="text-sm font-semibold text-ink">Prestadores</h2>
			<span class="text-xs text-mute tabular-nums">
				{prestadores.length}/{totalPrestadores}
			</span>
		</div>

		{#if prestadores.length === 0}
			<p
				class="rounded-xl border border-hairline-strong bg-surface-card px-4 py-6 text-center text-sm text-body"
			>
				Nenhum prestador cadastrado.
			</p>
		{:else}
			<ul
				class="divide-y divide-hairline overflow-hidden rounded-xl border border-hairline-strong bg-surface-card"
			>
				{#each prestadores as p (p.id)}
					{@render linha(p, '/admin/prestadores', banirPrestador, reativarPrestador)}
				{/each}
			</ul>
			{@render carregarMais(temMaisPrestadores, carregandoMais === 'prestadores', carregarMaisPrestadores)}
		{/if}
	</section>

	<section class="mt-6">
		<div class="mb-2 flex items-baseline justify-between gap-3">
			<h2 class="text-sm font-semibold text-ink">Clientes</h2>
			<span class="text-xs text-mute tabular-nums">
				{clientes.length}/{totalClientes}
			</span>
		</div>

		{#if clientes.length === 0}
			<p
				class="rounded-xl border border-hairline-strong bg-surface-card px-4 py-6 text-center text-sm text-body"
			>
				Nenhum cliente cadastrado.
			</p>
		{:else}
			<ul
				class="divide-y divide-hairline overflow-hidden rounded-xl border border-hairline-strong bg-surface-card"
			>
				{#each clientes as c (c.id)}
					{@render linha(c, '/admin/clientes', banirCliente, reativarCliente)}
				{/each}
			</ul>
			{@render carregarMais(temMaisClientes, carregandoMais === 'clientes', carregarMaisClientes)}
		{/if}
	</section>
</div>
