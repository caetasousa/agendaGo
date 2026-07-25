<script lang="ts">
	import type { PageData } from './$types';
	import type { AgendamentoAdmin } from '$lib/api/admin';
	import { dataLonga, minutosParaHHMM, rotuloStatus } from '$lib/format';
	import Indicadores from '$lib/components/Indicadores.svelte';

	let { data }: { data: PageData } = $props();
	// A página recarrega a cada navegação, então capturar o valor inicial é intencional.
	// svelte-ignore state_referenced_locally
	const p = data.prestador;

	const totalAgendamentos = p.agendamentos.length;
	const confirmados = p.agendamentos.filter((a) => a.status === 'CONFIRMADO').length;
	const semDesfechoPositivo = p.agendamentos.filter(
		(a) => a.status === 'CANCELADO' || a.status === 'NAO_COMPARECEU'
	).length;
</script>

<div>
	<a
		href="/admin"
		class="mb-4 inline-block text-sm text-mute transition hover:text-ink"
	>
		← Voltar à moderação
	</a>

	<div class="grid gap-3 lg:grid-cols-3">
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-5 sm:p-6 lg:col-span-1">
			<div class="flex flex-wrap items-center gap-2">
				<h1 class="min-w-0 text-xl font-semibold tracking-tight text-balance text-ink">{p.nome}</h1>
				{#if !p.ativo}
					<span
						class="inline-flex shrink-0 items-center rounded-full border border-accent-red/40 bg-accent-red/10 px-2.5 py-0.5 text-xs font-medium text-accent-red"
					>
						Banido
					</span>
				{/if}
			</div>
			<p class="mt-1.5 text-sm text-mute">Prestador</p>

			<dl class="mt-6 space-y-4 border-t border-hairline pt-4">
				<div>
					<dt class="text-xs text-mute">E-mail</dt>
					<dd class="text-sm text-body">{p.email}</dd>
				</div>
				<div>
					<dt class="text-xs text-mute">Aceita agendamentos</dt>
					<dd class="text-sm text-body">{p.aceitaAgendamentos ? 'Sim' : 'Não'}</dd>
				</div>
				<div>
					<dt class="text-xs text-mute">Duração do atendimento</dt>
					<dd class="text-sm text-body">{p.duracaoAtendimentoMinutos} min</dd>
				</div>
				<div>
					<dt class="text-xs text-mute">Intervalo de preparação</dt>
					<dd class="text-sm text-body">{p.descansoMinutos} min</dd>
				</div>
			</dl>
		</div>

		<div class="lg:col-span-2 lg:content-start">
			<Indicadores
				itens={[
					{ rotulo: 'Recebidos', valor: totalAgendamentos },
					{ rotulo: 'Confirmados', valor: confirmados },
					{ rotulo: 'Sem desfecho', valor: semDesfechoPositivo }
				]}
			/>
		</div>
	</div>

	<section class="mt-6">
		<h2 class="mb-2 text-sm font-semibold text-ink">Agendamentos recebidos</h2>

		{#if p.agendamentos.length === 0}
			<p
				class="rounded-xl border border-hairline-strong bg-surface-card px-4 py-6 text-center text-sm text-body"
			>
				Este prestador ainda não recebeu agendamentos.
			</p>
		{:else}
			<ul
				class="divide-y divide-hairline overflow-hidden rounded-xl border border-hairline-strong bg-surface-card md:hidden"
			>
				{#each p.agendamentos as a (a.id)}
					{@const rotulo = rotuloStatus(a.status)}
					<li data-agendamento={a.id} class="px-4 py-3.5">
						<div class="flex flex-wrap items-start justify-between gap-2">
							<div class="min-w-0">
								<p class="text-sm font-medium text-ink">
									{dataLonga(a.data)} · {minutosParaHHMM(a.inicioMinutos)}–{minutosParaHHMM(a.fimMinutos)}
								</p>
								<p class="mt-0.5 text-sm text-body">{a.nomeCliente}</p>
								{#if a.telefoneCliente || a.emailCliente}
									<p class="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-mute">
										{#if a.telefoneCliente}<span>{a.telefoneCliente}</span>{/if}
										{#if a.emailCliente}<span>{a.emailCliente}</span>{/if}
									</p>
								{/if}
							</div>
							<span
								class="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-hairline-strong bg-surface-elevated px-2.5 py-0.5 text-xs text-body"
							>
								<span class="h-1.5 w-1.5 rounded-full {rotulo.cor}"></span>
								{rotulo.texto}
							</span>
						</div>
					</li>
				{/each}
			</ul>

			<div
				class="hidden overflow-x-auto rounded-xl border border-hairline-strong bg-surface-card md:block"
			>
				<table class="w-full text-left text-sm">
					<thead>
						<tr class="text-xs text-mute uppercase">
							<th class="border-b border-hairline px-4 py-2.5 font-medium">Data</th>
							<th class="border-b border-hairline px-4 py-2.5 font-medium">Horário</th>
							<th class="border-b border-hairline px-4 py-2.5 font-medium">Cliente</th>
							<th class="border-b border-hairline px-4 py-2.5 font-medium">Status</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-hairline">
						{#each p.agendamentos as a (a.id)}
							{@const rotulo = rotuloStatus(a.status)}
							<tr data-agendamento={a.id}>
								<td class="px-4 py-3 whitespace-nowrap text-ink">{dataLonga(a.data)}</td>
								<td class="px-4 py-3 whitespace-nowrap text-body tabular-nums">
									{minutosParaHHMM(a.inicioMinutos)}–{minutosParaHHMM(a.fimMinutos)}
								</td>
								<td class="px-4 py-3 text-body">
									<div>{a.nomeCliente}</div>
									{#if a.telefoneCliente || a.emailCliente}
										<div class="mt-0.5 flex flex-wrap gap-x-3 text-xs text-mute">
											{#if a.telefoneCliente}<span>{a.telefoneCliente}</span>{/if}
											{#if a.emailCliente}<span>{a.emailCliente}</span>{/if}
										</div>
									{/if}
								</td>
								<td class="px-4 py-3">
									<span
										class="inline-flex items-center gap-1.5 rounded-full border border-hairline-strong bg-surface-elevated px-2.5 py-0.5 text-xs text-body"
									>
										<span class="h-1.5 w-1.5 rounded-full {rotulo.cor}"></span>
										{rotulo.texto}
									</span>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
</div>
