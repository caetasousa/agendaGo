<script lang="ts">
	import { ApiError } from '$lib/api/client';
	import { cadastrarProvider, type CadastrarProviderResponse } from '$lib/api/provider';

	let nome = $state('');
	let email = $state('');
	let senha = $state('');
	let confirmarSenha = $state('');

	let enviando = $state(false);
	let erro = $state<string | null>(null);
	let sucesso = $state<CadastrarProviderResponse | null>(null);

	const senhasDivergentes = $derived(confirmarSenha.length > 0 && senha !== confirmarSenha);

	async function enviar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;
		sucesso = null;

		if (senha !== confirmarSenha) {
			erro = 'As senhas não coincidem.';
			return;
		}

		enviando = true;

		try {
			sucesso = await cadastrarProvider({ nome, email, senha });
			nome = '';
			email = '';
			senha = '';
			confirmarSenha = '';
		} catch (e) {
			// A API é a fonte da verdade da validação: mostramos a mensagem que ela devolve
			// (400 = dado inválido, 409 = e-mail já cadastrado).
			erro = e instanceof ApiError ? e.message : 'Não foi possível concluir o cadastro.';
		} finally {
			enviando = false;
		}
	}

	const inputClasse =
		'mt-2 h-10 w-full rounded-md border border-hairline-strong bg-surface-card px-3.5 text-sm text-ink outline-none transition placeholder:text-mute focus:border-ink';
</script>

<div class="mx-auto max-w-xl">
	<a href="/" class="text-sm text-mute transition hover:text-ink">← Voltar</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Cadastrar prestador</h1>
	<p class="mt-3 text-body">Crie sua conta de prestador para começar a receber agendamentos.</p>

	<div class="mt-8 rounded-lg border border-hairline-strong bg-surface-card p-8">
		{#if sucesso}
			<div class="mb-6 rounded-md border border-hairline-strong bg-surface-elevated p-4">
				<div class="flex items-center gap-2">
					<span class="h-2 w-2 rounded-full bg-accent-green"></span>
					<p class="font-medium text-ink">Prestador cadastrado!</p>
				</div>
				<p class="mt-1 text-sm text-body">{sucesso.nome} ({sucesso.email})</p>
			</div>
		{/if}

		<form class="space-y-5" onsubmit={enviar}>
			{#if erro}
				<div
					class="flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
					<span class="text-body">{erro}</span>
				</div>
			{/if}

			<div>
				<label for="nome" class="block text-sm font-medium text-ink">Nome</label>
				<input
					id="nome"
					type="text"
					bind:value={nome}
					required
					minlength="2"
					maxlength="100"
					placeholder="Seu nome"
					class={inputClasse}
				/>
			</div>

			<div>
				<label for="email" class="block text-sm font-medium text-ink">E-mail</label>
				<input
					id="email"
					type="email"
					bind:value={email}
					required
					placeholder="voce@exemplo.com"
					class={inputClasse}
				/>
			</div>

			<div>
				<label for="senha" class="block text-sm font-medium text-ink">Senha</label>
				<input
					id="senha"
					type="password"
					bind:value={senha}
					required
					minlength="8"
					placeholder="Mínimo de 8 caracteres"
					class={inputClasse}
				/>
			</div>

			<div>
				<label for="confirmar-senha" class="block text-sm font-medium text-ink"
					>Confirmar senha</label
				>
				<input
					id="confirmar-senha"
					type="password"
					bind:value={confirmarSenha}
					required
					minlength="8"
					placeholder="Repita a senha"
					aria-invalid={senhasDivergentes}
					class={inputClasse}
				/>
				{#if senhasDivergentes}
					<p class="mt-1.5 text-sm text-accent-red">As senhas não coincidem.</p>
				{/if}
			</div>

			<button
				type="submit"
				disabled={enviando || senhasDivergentes}
				class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{enviando ? 'Enviando…' : 'Cadastrar'}
			</button>
		</form>
	</div>
</div>
