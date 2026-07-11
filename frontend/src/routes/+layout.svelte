<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import favicon from '$lib/assets/favicon.svg';
	import Header from '$lib/components/Header.svelte';
	import { sessao } from '$lib/stores/session.svelte';

	let { children } = $props();

	onMount(() => {
		sessao.carregar();
	});
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="flex min-h-screen flex-col bg-canvas text-body">
	<Header />

	<main class="mx-auto w-full max-w-5xl flex-1 px-6 py-16">
		{@render children()}
	</main>

	<footer class="border-t border-hairline">
		<div class="mx-auto flex max-w-5xl items-center justify-between px-6 py-8 text-xs text-mute">
			<span>agendaGo — agendamento entre clientes e prestadores.</span>
			<span>Projeto de estudo · Go + SvelteKit</span>
		</div>
	</footer>
</div>
