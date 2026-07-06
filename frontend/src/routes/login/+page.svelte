<script lang="ts">
	import { goto } from '$app/navigation';
	import { ApiError } from '$lib/api/client';
	import { login } from '$lib/api/auth';

	let email = $state('');
	let senha = $state('');

	let enviando = $state(false);
	let erro = $state<string | null>(null);

	async function enviar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;
		enviando = true;

		try {
			await login({ email, senha });
			goto('/painel');
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível entrar.';
		} finally {
			enviando = false;
		}
	}

	const inputClasse =
		'mt-2 h-10 w-full rounded-md border border-hairline-strong bg-surface-card px-3.5 text-sm text-ink outline-none transition placeholder:text-mute focus:border-ink';
</script>

<div class="mx-auto max-w-xl">
	<a href="/" class="text-sm text-mute transition hover:text-ink">← Voltar</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Entrar</h1>
	<p class="mt-3 text-body">Acesse sua conta de prestador ou cliente.</p>

	<div class="mt-8 rounded-lg border border-hairline-strong bg-surface-card p-8">
		<form class="space-y-5" novalidate onsubmit={enviar}>
			{#if erro}
				<div
					class="flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
					<span class="text-body">{erro}</span>
				</div>
			{/if}

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
					placeholder="Sua senha"
					class={inputClasse}
				/>
			</div>

			<button
				type="submit"
				disabled={enviando}
				class="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{enviando ? 'Entrando…' : 'Entrar'}
			</button>
		</form>

		<p class="mt-6 text-sm text-body">
			Ainda não tem conta?
			<a href="/cadastro" class="font-medium text-ink underline">Cadastre-se</a>
		</p>
	</div>
</div>
