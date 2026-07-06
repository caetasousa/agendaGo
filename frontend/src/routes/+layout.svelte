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

<div class="min-h-screen bg-canvas text-body">
	<Header />

	<main class="mx-auto max-w-5xl px-6 py-16">
		{@render children()}
	</main>
</div>
