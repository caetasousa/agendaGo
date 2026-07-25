<script lang="ts">
	import type { PageData } from './$types';
	import { dataLonga, minutosParaHHMM, rotuloStatus } from '$lib/format';
	import Indicadores from '$lib/components/Indicadores.svelte';

	let { data }: { data: PageData } = $props();
	// A página recarrega a cada navegação, então capturar o valor inicial é intencional.
	// svelte-ignore state_referenced_locally
	const c = data.cliente;

	const totalAgendamentos = c.agendamentos.length;
	const confirmados = c.agendamentos.filter((a) => a.status === 'CONFIRMADO').length;
	const prestadoresDistintos = new Set(c.agendamentos.map((a) => a.nomePrestador)).size;
</script>

<div>
	<a href="/admin" class="mb-4 inline-block text-sm text-mute transition hover:text-ink">
		← Voltar à moderação
	</a>

	<div class="grid gap-3 lg:grid-cols-3">
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-5 sm:p-6 lg:col-span-1">
			<div class="flex flex-wrap items-center gap-2">
				<h1 class="min-w-0 text-xl font-semibold tracking-tight text-balance text-ink">{c.nome}</h1>
				{#if !c.ativo}
					<span
						class="inline-flex shrink-0 items-center rounded-full border border-accent-red/40 bg-accent-red/10 px-2.5 py-0.5 text-xs font-medium text-accent-red"
					>
						Banido
					</span>
				{/if}
			</div>
			<p class="mt-1.5 text-sm text-mute">
				Cliente · {c.temConta ? 'com conta' : 'convidado (sem cadastro)'}
			</p>

			<dl class="mt-6 space-y-4 border-t border-hairline pt-4">
				<div>
					<dt class="text-xs text-mute">E-mail</dt>
					<dd class="text-sm text-body">{c.email}</dd>
				</div>
				<div>
					<dt class="text-xs text-mute">Telefone</dt>
					<dd class="text-sm text-body">{c.telefone || '—'}</dd>
				</div>
			</dl>
		</div>

		<div class="lg:col-span-2 lg:content-start">
			<Indicadores
				itens={[
					{ rotulo: 'Agendamentos', valor: totalAgendamentos },
					{ rotulo: 'Confirmados', valor: confirmados },
					{ rotulo: 'Prestadores', valor: prestadoresDistintos }
				]}
			/>
		</div>
	</div>

	<section class="mt-6">
		<h2 class="mb-2 text-sm font-semibold text-ink">Agendamentos feitos</h2>

		{#if c.agendamentos.length === 0}
			<p
				class="rounded-xl border border-hairline-strong bg-surface-card px-4 py-6 text-center text-sm text-body"
			>
				Este cliente ainda não fez agendamentos.
			</p>
		{:else}
			<ul
				class="divide-y divide-hairline overflow-hidden rounded-xl border border-hairline-strong bg-surface-card md:hidden"
			>
				{#each c.agendamentos as a (a.id)}
					{@const rotulo = rotuloStatus(a.status)}
					<li data-agendamento={a.id} class="px-4 py-3.5">
						<div class="flex flex-wrap items-start justify-between gap-2">
							<div class="min-w-0">
								<p class="text-sm font-medium text-ink">
									{dataLonga(a.data)} · <span class="tabular-nums"
										>{minutosParaHHMM(a.inicioMinutos)}–{minutosParaHHMM(a.fimMinutos)}</span
									>
								</p>
								<p class="mt-0.5 text-sm text-body">{a.nomePrestador}</p>
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
							<th class="border-b border-hairline px-4 py-2.5 font-medium">Prestador</th>
							<th class="border-b border-hairline px-4 py-2.5 font-medium">Status</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-hairline">
						{#each c.agendamentos as a (a.id)}
							{@const rotulo = rotuloStatus(a.status)}
							<tr data-agendamento={a.id}>
								<td class="px-4 py-3 whitespace-nowrap text-ink">{dataLonga(a.data)}</td>
								<td class="px-4 py-3 whitespace-nowrap text-body tabular-nums">
									{minutosParaHHMM(a.inicioMinutos)}–{minutosParaHHMM(a.fimMinutos)}
								</td>
								<td class="px-4 py-3 text-body">{a.nomePrestador}</td>
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
