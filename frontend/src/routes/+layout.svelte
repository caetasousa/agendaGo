<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="min-h-screen bg-gray-50 text-gray-900">
	<header class="border-b border-gray-200 bg-white">
		<div class="mx-auto max-w-3xl px-4 py-4">
			<a href="/" class="text-xl font-semibold tracking-tight">agendaGo</a>
		</div>
	</header>

	<main class="mx-auto max-w-3xl px-4 py-8">
		{@render children()}
	</main>
</div>
