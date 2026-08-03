<script lang="ts">
	import { page } from '$app/state';
	import { sessao } from '$lib/stores/session.svelte';
	import Icone from '$lib/components/Icone.svelte';

	let { children } = $props();

	const ehPrestador = $derived(sessao.usuario?.tipo === 'provider');
	const permiteMarcacao = $derived(sessao.usuario?.provider?.permiteMarcacaoPeloPrestador ?? false);

	type Item = { href: string; rotulo: string; icone: 'inicio' | 'agendamentos' | 'disponibilidade' | 'configuracoes' | 'marcar' | 'agendar' | 'equipe' };
	type Secao = { titulo: string | null; itens: Item[] };

	// Agrupamento por intenção: o que é operação do dia a dia fica separado do
	// que é configuração da agenda — a sidebar precisa dizer isso, não só listar.
	const secoes = $derived.by<Secao[]>(() => {
		if (!ehPrestador) {
			return [
				{ titulo: null, itens: [{ href: '/painel', rotulo: 'Início', icone: 'inicio' }] },
				{
					titulo: 'Agendamentos',
					itens: [
						{ href: '/painel/agendar', rotulo: 'Agendar', icone: 'agendar' },
						{ href: '/painel/agendamentos', rotulo: 'Meus agendamentos', icone: 'agendamentos' }
					]
				}
			];
		}

		const atendimento: Item[] = [
			{ href: '/painel/agendamentos', rotulo: 'Agendamentos', icone: 'agendamentos' }
		];
		if (permiteMarcacao) {
			atendimento.push({ href: '/painel/marcar', rotulo: 'Marcar para cliente', icone: 'marcar' });
		}

		return [
			{ titulo: null, itens: [{ href: '/painel', rotulo: 'Início', icone: 'inicio' }] },
			{ titulo: 'Atendimento', itens: atendimento },
			{
				titulo: 'Sua agenda',
				itens: [
					{ href: '/painel/disponibilidade', rotulo: 'Disponibilidade', icone: 'disponibilidade' },
					{ href: '/painel/configuracoes', rotulo: 'Configurações', icone: 'configuracoes' },
					{ href: '/painel/equipe', rotulo: 'Equipe', icone: 'equipe' }
				]
			}
		];
	});

	const todosItens = $derived(secoes.flatMap((s) => s.itens));

	function ativo(href: string): boolean {
		return href === '/painel' ? page.url.pathname === href : page.url.pathname.startsWith(href);
	}

	// Iniciais para o avatar: primeiro e último nome dão uma marca estável mesmo
	// quando o nome completo é longo demais para caber na sidebar.
	const iniciais = $derived.by(() => {
		const partes = (sessao.usuario?.nome ?? '').trim().split(/\s+/).filter(Boolean);
		if (partes.length === 0) return '?';
		const primeira = partes[0][0];
		const ultima = partes.length > 1 ? partes[partes.length - 1][0] : '';
		return (primeira + ultima).toUpperCase();
	});
</script>

<div class="flex w-full flex-1 flex-col gap-6 md:flex-row md:gap-0">
	<!-- Sidebar (desktop) -->
	<aside class="hidden shrink-0 md:block md:w-60 md:border-r md:border-hairline md:pr-6">
		<div class="sticky top-24 flex flex-col gap-6">
			<nav class="flex flex-col gap-6">
				{#each secoes as secao (secao.titulo ?? 'principal')}
					<div class="flex flex-col gap-0.5">
						{#if secao.titulo}
							<p class="mb-1.5 px-3 text-[11px] font-semibold tracking-widest text-mute uppercase">
								{secao.titulo}
							</p>
						{/if}
						{#each secao.itens as item (item.href)}
							{@const estaAtivo = ativo(item.href)}
							<a
								href={item.href}
								aria-current={estaAtivo ? 'page' : undefined}
								class="relative flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition {estaAtivo
									? 'bg-surface-elevated font-medium text-ink'
									: 'text-mute hover:bg-surface-card hover:text-ink'}"
							>
								{#if estaAtivo}
									<span
										class="absolute top-1/2 left-0 h-5 w-0.5 -translate-y-1/2 rounded-r-full bg-accent-green"
										aria-hidden="true"
									></span>
								{/if}
								<Icone
									nome={item.icone}
									classe={estaAtivo ? 'shrink-0 text-accent-green' : 'shrink-0'}
								/>
								<span class="truncate">{item.rotulo}</span>
							</a>
						{/each}
					</div>
				{/each}
			</nav>

			<!-- Identidade da sessão: o nome vive aqui, truncado, em vez de vazar
			     cru no header e empurrar a navegação. A saída fica no header, que é
			     o único lugar visível em qualquer largura. -->
			{#if sessao.usuario}
				<div class="flex items-center gap-2.5 border-t border-hairline pt-4">
					<span
						class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-surface-elevated text-[11px] font-semibold text-ink"
						aria-hidden="true"
					>
						{iniciais}
					</span>
					<span class="min-w-0 flex-1">
						<span class="block truncate text-sm text-ink" title={sessao.usuario.nome}>
							{sessao.usuario.nome}
						</span>
						<span class="block text-xs text-mute">
							{ehPrestador ? 'Prestador' : 'Cliente'}
						</span>
					</span>
				</div>
			{/if}
		</div>
	</aside>

	<!-- Navegação (mobile) -->
	<nav class="-mx-4 flex gap-2 overflow-x-auto px-4 pb-1 md:hidden">
		{#each todosItens as item (item.href)}
			{@const estaAtivo = ativo(item.href)}
			<a
				href={item.href}
				aria-current={estaAtivo ? 'page' : undefined}
				class="flex shrink-0 items-center gap-2 rounded-full border px-3.5 py-1.5 text-sm transition {estaAtivo
					? 'border-hairline-strong bg-surface-elevated font-medium text-ink'
					: 'border-hairline text-mute'}"
			>
				<Icone nome={item.icone} tamanho={15} classe="shrink-0" />
				{item.rotulo}
			</a>
		{/each}
	</nav>

	<div class="min-w-0 flex-1 md:pl-8">
		{@render children()}
	</div>
</div>
