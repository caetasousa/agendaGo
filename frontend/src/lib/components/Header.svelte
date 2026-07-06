<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { sessao } from '$lib/stores/session.svelte';

	let tema = $state<'dark' | 'light'>('dark');

	onMount(() => {
		tema = document.documentElement.dataset.theme === 'light' ? 'light' : 'dark';
	});

	function alternarTema() {
		tema = tema === 'dark' ? 'light' : 'dark';
		document.documentElement.dataset.theme = tema;
		try {
			localStorage.setItem('theme', tema);
		} catch {
			// localStorage indisponível: mantém a escolha só nesta sessão
		}
	}

	async function sair() {
		await sessao.sair();
		goto('/');
	}
</script>

<header class="sticky top-0 z-10 border-b border-hairline bg-canvas/80 backdrop-blur">
	<div class="mx-auto flex h-16 max-w-5xl items-center justify-between px-6">
		<a href="/" class="text-lg font-semibold tracking-tight text-ink">agendaGo</a>

		<div class="flex items-center gap-4">
			{#if sessao.carregando}
				<!-- espaço neutro: evita mostrar "Entrar" antes de saber se há sessão -->
			{:else if sessao.usuario}
				<span class="text-sm text-body">{sessao.usuario.nome}</span>
				<a href="/painel" class="text-sm text-mute transition hover:text-ink">Painel</a>
				<button
					type="button"
					onclick={sair}
					class="text-sm text-mute transition hover:text-ink"
				>
					Sair
				</button>
			{:else}
				<a href="/login" class="text-sm text-mute transition hover:text-ink">Entrar</a>
				<a href="/cadastro" class="text-sm text-mute transition hover:text-ink">Criar conta</a>
			{/if}

			<button
				type="button"
				onclick={alternarTema}
				aria-label={tema === 'dark' ? 'Mudar para tema claro' : 'Mudar para tema escuro'}
				class="flex h-9 w-9 items-center justify-center rounded-md border border-hairline-strong bg-surface-elevated text-ink transition hover:border-ink/40"
			>
				{#if tema === 'dark'}
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="18"
						height="18"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"
					>
						<circle cx="12" cy="12" r="4" />
						<path
							d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"
						/>
					</svg>
				{:else}
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="18"
						height="18"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"
					>
						<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z" />
					</svg>
				{/if}
			</button>
		</div>
	</div>
</header>
