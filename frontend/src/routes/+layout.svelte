<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import favicon from '$lib/assets/favicon.svg';
	import Header from '$lib/components/Header.svelte';
	import { sessao } from '$lib/stores/session.svelte';

	let { children } = $props();

	onMount(() => {
		sessao.carregar();
	});

	// Painel e moderação são aplicação, não página de marketing: usam container
	// mais largo, respiro vertical menor e dispensam o rodapé institucional.
	const noApp = $derived(
		page.url.pathname.startsWith('/painel') || page.url.pathname.startsWith('/admin')
	);
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="flex min-h-screen flex-col bg-canvas text-body">
	<Header />

	<main
		class="mx-auto flex w-full flex-1 flex-col {noApp
			? 'max-w-6xl px-4 py-8 sm:px-6'
			: 'max-w-5xl px-6 py-16'}"
	>
		{@render children()}
	</main>

	{#if !noApp}
		<footer class="border-t border-hairline">
			<div class="mx-auto flex max-w-5xl items-center justify-between px-6 py-8 text-xs text-mute">
				<span>agendaGo — agendamento entre clientes e prestadores.</span>
				<span>Projeto de estudo · Go + SvelteKit</span>
			</div>
		</footer>
	{/if}
</div>
