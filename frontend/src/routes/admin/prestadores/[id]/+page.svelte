<script lang="ts">
	import type { PageData } from './$types';
	import type { AgendamentoAdmin } from '$lib/api/admin';
	import { dataLonga, minutosParaHHMM, rotuloStatus } from '$lib/format';

	let { data }: { data: PageData } = $props();
	// A página recarrega a cada navegação, então capturar o valor inicial é intencional.
	// svelte-ignore state_referenced_locally
	const p = data.prestador;
</script>

<div class="mx-auto max-w-2xl">
	<a href="/admin" class="text-sm text-mute transition hover:text-ink">← Voltar à moderação</a>

	<div class="mt-4 flex flex-wrap items-center gap-3">
		<h1 class="display text-4xl text-ink sm:text-5xl">{p.nome}</h1>
		{#if !p.ativo}
			<span
				class="inline-flex items-center rounded-full border border-accent-red/40 bg-accent-red/10 px-2.5 py-0.5 text-xs font-medium text-accent-red"
			>
				Banido
			</span>
		{/if}
	</div>
	<p class="mt-2 text-sm text-mute">Prestador · visão em leitura</p>

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-5 sm:p-8">
		<h2 class="text-lg font-semibold text-ink">Dados cadastrais</h2>
		<dl class="mt-4 grid gap-4 sm:grid-cols-2">
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

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-5 sm:p-8">
		<div class="flex items-center justify-between">
			<h2 class="text-lg font-semibold text-ink">Agendamentos recebidos</h2>
			<span class="text-xs text-mute">{p.agendamentos.length}</span>
		</div>

		{#if p.agendamentos.length === 0}
			<p class="mt-4 text-sm text-body">Este prestador ainda não recebeu agendamentos.</p>
		{:else}
			<ul class="mt-4 space-y-2">
				{#each p.agendamentos as a (a.id)}
					{@const rotulo = rotuloStatus(a.status)}
					<li
						data-agendamento={a.id}
						class="rounded-md border border-hairline-strong p-4"
					>
						<div class="flex flex-wrap items-center justify-between gap-2">
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
								class="inline-flex items-center gap-1.5 rounded-full border border-hairline-strong bg-surface-elevated px-2.5 py-0.5 text-xs text-body"
							>
								<span class="h-1.5 w-1.5 rounded-full {rotulo.cor}"></span>
								{rotulo.texto}
							</span>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</div>
