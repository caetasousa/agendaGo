<script lang="ts">
	import { ApiError } from '$lib/api/client';
	import { cadastrarProvider, type CadastrarProviderResponse } from '$lib/api/provider';

	let nome = $state('');
	let email = $state('');
	let senha = $state('');

	let enviando = $state(false);
	let erro = $state<string | null>(null);
	let sucesso = $state<CadastrarProviderResponse | null>(null);

	async function enviar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;
		sucesso = null;
		enviando = true;

		try {
			sucesso = await cadastrarProvider({ nome, email, senha });
			nome = '';
			email = '';
			senha = '';
		} catch (e) {
			// A API é a fonte da verdade da validação: mostramos a mensagem que ela devolve
			// (400 = dado inválido, 409 = e-mail já cadastrado).
			erro = e instanceof ApiError ? e.message : 'Não foi possível concluir o cadastro.';
		} finally {
			enviando = false;
		}
	}
</script>

<h1 class="text-2xl font-bold">Cadastrar prestador</h1>

{#if sucesso}
	<div class="mt-6 rounded-md border border-green-200 bg-green-50 p-4 text-green-800">
		<p class="font-medium">Prestador cadastrado com sucesso!</p>
		<p class="mt-1 text-sm">
			{sucesso.nome} ({sucesso.email})
		</p>
	</div>
{/if}

<form class="mt-6 space-y-4" onsubmit={enviar}>
	{#if erro}
		<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
			{erro}
		</div>
	{/if}

	<div>
		<label for="nome" class="block text-sm font-medium">Nome</label>
		<input
			id="nome"
			type="text"
			bind:value={nome}
			required
			minlength="2"
			maxlength="100"
			class="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
		/>
	</div>

	<div>
		<label for="email" class="block text-sm font-medium">E-mail</label>
		<input
			id="email"
			type="email"
			bind:value={email}
			required
			class="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
		/>
	</div>

	<div>
		<label for="senha" class="block text-sm font-medium">Senha</label>
		<input
			id="senha"
			type="password"
			bind:value={senha}
			required
			minlength="8"
			class="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
		/>
		<p class="mt-1 text-xs text-gray-500">Mínimo de 8 caracteres.</p>
	</div>

	<button
		type="submit"
		disabled={enviando}
		class="rounded-md bg-indigo-600 px-4 py-2 font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-60"
	>
		{enviando ? 'Enviando...' : 'Cadastrar'}
	</button>
</form>
