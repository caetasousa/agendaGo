<script lang="ts">
	import { page } from '$app/state';
	import { ApiError } from '$lib/api/client';
	import { confirmarCadastro } from '$lib/api/customer';
	import AuthLayout from '$lib/components/AuthLayout.svelte';

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

<AuthLayout
	titulo={estado === 'ok' ? 'Sua conta está pronta.' : 'Confirmação de cadastro.'}
	lede={estado === 'ok'
		? 'Foi só isso. Entre para ver seus agendamentos — inclusive os que você já tinha feito sem conta com este email.'
		: 'Estamos verificando o link que você recebeu por email.'}
>
	<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
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
</AuthLayout>
