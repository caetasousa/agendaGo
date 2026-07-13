<script lang="ts">
	import { page } from '$app/state';
	import { ApiError } from '$lib/api/client';
	import { confirmarCadastro } from '$lib/api/customer';

	const token = page.url.searchParams.get('token') ?? '';

	// Estados: 'confirmando' enquanto chama a API, 'ok' no sucesso, 'erro' quando
	// o token é inválido/expirado ou não veio na URL.
	let estado = $state<'confirmando' | 'ok' | 'erro'>(token ? 'confirmando' : 'erro');

	$effect(() => {
		if (!token) return;
		confirmarCadastro(token)
			.then(() => (estado = 'ok'))
			.catch((e) => {
				estado = 'erro';
				void e;
			});
	});
</script>

<div class="mx-auto max-w-xl">
	<a href="/" class="text-sm text-mute transition hover:text-ink">← Início</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Confirmação de cadastro</h1>

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-8">
		{#if estado === 'confirmando'}
			<p class="text-body">Confirmando seu cadastro…</p>
		{:else if estado === 'ok'}
			<p class="text-body">
				Cadastro confirmado! Sua conta está pronta.
				<a href="/login" class="font-medium text-ink underline">Entrar</a>
			</p>
		{:else}
			<p class="text-body">
				Este link de confirmação é inválido ou expirou.
				<a href="/cadastro" class="font-medium text-ink underline">Faça o cadastro novamente</a>.
			</p>
		{/if}
	</div>
</div>
