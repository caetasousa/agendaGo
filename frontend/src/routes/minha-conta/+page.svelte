<script lang="ts">
	import { goto } from '$app/navigation';
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import { baixarComoJson, exportarMeusDados, removerMinhaConta } from '$lib/api/lgpd';
	import { sessao } from '$lib/stores/session.svelte';

	let { data }: { data: PageData } = $props();

	let baixando = $state(false);
	let removendo = $state(false);
	let erro = $state<string | null>(null);
	let confirmandoRemocao = $state(false);

	async function baixarDados() {
		erro = null;
		baixando = true;
		try {
			const dados = await exportarMeusDados();
			baixarComoJson(dados, 'meus-dados-agendago.json');
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível exportar seus dados.';
		} finally {
			baixando = false;
		}
	}

	async function removerConta() {
		erro = null;
		removendo = true;
		try {
			await removerMinhaConta();
			// A sessão já morreu no servidor; limpar aqui evita a próxima página
			// tentar carregar com um estado que não existe mais.
			sessao.limpar();
			await goto('/');
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível remover sua conta.';
			removendo = false;
		}
	}
</script>

<svelte:head><title>Minha conta · agendaGo</title></svelte:head>

<div class="mx-auto w-full max-w-2xl px-4 py-10">
	<h1 class="text-xl font-semibold text-ink">Minha conta</h1>
	<p class="mt-1 text-sm text-mute">{data.nome} · {data.email}</p>

	{#if erro}
		<div
			class="mt-6 flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
		>
			<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
			<span class="text-body">{erro}</span>
		</div>
	{/if}

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-6">
		<h2 class="text-sm font-semibold text-ink">Seus dados</h2>
		<p class="mt-1 text-sm text-mute">
			Baixe um arquivo com seu cadastro e seu histórico de agendamentos.
		</p>
		<button
			type="button"
			disabled={baixando}
			onclick={baixarDados}
			class="mt-4 inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
		>
			{baixando ? 'Preparando…' : 'Baixar meus dados'}
		</button>
	</div>

	<div class="mt-4 rounded-xl border border-hairline-strong bg-surface-card p-6">
		<h2 class="text-sm font-semibold text-ink">Remover minha conta</h2>
		<p class="mt-1 text-sm text-mute">
			Seu nome, email e telefone são apagados, e você deixa de conseguir entrar.
		</p>
		<p class="mt-2 text-sm text-mute">
			Os horários que você já agendou continuam na agenda de quem atendeu, mas
			<strong class="text-body">sem identificar você</strong> — é o registro de trabalho dessa
			pessoa, e ela precisa mantê-lo.
		</p>

		{#if confirmandoRemocao}
			<div class="mt-4 rounded-md border border-hairline-strong bg-surface-elevated p-4">
				<p class="text-sm text-body">Esta ação não pode ser desfeita. Confirma?</p>
				<div class="mt-3 flex flex-wrap gap-2">
					<button
						type="button"
						disabled={removendo}
						onclick={removerConta}
						class="inline-flex h-9 items-center rounded-md border border-hairline-strong px-4 text-sm font-medium text-accent-red transition hover:bg-surface-card disabled:cursor-not-allowed disabled:opacity-60"
					>
						{removendo ? 'Removendo…' : 'Sim, remover minha conta'}
					</button>
					<button
						type="button"
						disabled={removendo}
						onclick={() => (confirmandoRemocao = false)}
						class="text-sm text-mute transition hover:text-ink"
					>
						Cancelar
					</button>
				</div>
			</div>
		{:else}
			<button
				type="button"
				onclick={() => (confirmandoRemocao = true)}
				class="mt-4 inline-flex h-9 items-center rounded-md border border-hairline-strong px-4 text-sm font-medium text-accent-red transition hover:bg-surface-elevated"
			>
				Remover minha conta
			</button>
		{/if}
	</div>
</div>
