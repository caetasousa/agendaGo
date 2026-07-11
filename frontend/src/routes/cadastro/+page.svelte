<script lang="ts">
	import { goto } from '$app/navigation';
	import { ApiError } from '$lib/api/client';
	import { cadastrarProvider } from '$lib/api/provider';
	import { cadastrarClient } from '$lib/api/customer';
	import { login } from '$lib/api/auth';

	type TipoConta = 'provider' | 'client';

	let tipo = $state<TipoConta>('provider');
	let nome = $state('');
	let email = $state('');
	let senha = $state('');
	let confirmarSenha = $state('');

	let enviando = $state(false);
	let erro = $state<string | null>(null);

	const senhasDivergentes = $derived(confirmarSenha.length > 0 && senha !== confirmarSenha);

	async function enviar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;

		if (senha !== confirmarSenha) {
			erro = 'As senhas não coincidem.';
			return;
		}

		enviando = true;

		try {
			if (tipo === 'provider') {
				await cadastrarProvider({ nome, email, senha });
			} else {
				await cadastrarClient({ nome, email, senha });
			}

			await login({ email, senha });
			goto('/painel');
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

	const opcaoBaseClasse =
		'flex-1 rounded-md border px-4 py-2 text-sm font-medium transition cursor-pointer text-center';
	const opcaoAtivaClasse = 'border-ink bg-surface-elevated text-ink';
	const opcaoInativaClasse = 'border-hairline-strong text-mute hover:text-ink';
</script>

<div class="mx-auto max-w-xl">
	<a href="/" class="text-sm text-mute transition hover:text-ink">← Voltar</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Criar conta</h1>
	<p class="mt-3 text-body">Escolha o tipo de conta para começar.</p>

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-8">
		<form class="space-y-5" novalidate onsubmit={enviar}>
			<div role="radiogroup" aria-label="Tipo de conta" class="flex gap-3">
				<label class="{opcaoBaseClasse} {tipo === 'provider' ? opcaoAtivaClasse : opcaoInativaClasse}">
					<input type="radio" name="tipo" value="provider" bind:group={tipo} class="sr-only" />
					Prestador
				</label>
				<label class="{opcaoBaseClasse} {tipo === 'client' ? opcaoAtivaClasse : opcaoInativaClasse}">
					<input type="radio" name="tipo" value="client" bind:group={tipo} class="sr-only" />
					Cliente
				</label>
			</div>

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
				class="flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{enviando ? 'Enviando…' : 'Criar conta'}
			</button>
		</form>

		<p class="mt-6 text-sm text-body">
			Já tem conta?
			<a href="/login" class="font-medium text-ink underline">Entrar</a>
		</p>
	</div>
</div>
