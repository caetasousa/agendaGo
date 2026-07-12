<script lang="ts">
	import { page } from '$app/state';
	import { ApiError } from '$lib/api/client';
	import { redefinirSenha } from '$lib/api/auth';

	// svelte-ignore state_referenced_locally
	const token = page.url.searchParams.get('token') ?? '';

	let novaSenha = $state('');
	let confirmacao = $state('');

	let enviando = $state(false);
	let erro = $state<string | null>(null);
	let sucesso = $state(false);

	async function enviar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;

		if (novaSenha.length < 8) {
			erro = 'A senha deve ter pelo menos 8 caracteres.';
			return;
		}
		if (novaSenha !== confirmacao) {
			erro = 'As senhas não coincidem.';
			return;
		}

		enviando = true;
		try {
			await redefinirSenha({ token, novaSenha });
			sucesso = true;
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível redefinir a senha.';
		} finally {
			enviando = false;
		}
	}

	const inputClasse =
		'mt-2 h-10 w-full rounded-md border border-hairline-strong bg-surface-card px-3.5 text-sm text-ink outline-none transition placeholder:text-mute focus:border-ink';
</script>

<div class="mx-auto max-w-xl">
	<a href="/login" class="text-sm text-mute transition hover:text-ink">← Voltar</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Redefinir senha</h1>
	<p class="mt-3 text-body">Escolha uma nova senha para sua conta.</p>

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-8">
		{#if !token}
			<p class="text-body">
				Link inválido. <a href="/recuperar-senha" class="font-medium text-ink underline"
					>Solicite uma nova recuperação de senha</a
				>.
			</p>
		{:else if sucesso}
			<p class="text-body">
				Senha redefinida com sucesso. <a href="/login" class="font-medium text-ink underline"
					>Entrar</a
				>
			</p>
		{:else}
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
					<label for="novaSenha" class="block text-sm font-medium text-ink">Nova senha</label>
					<input
						id="novaSenha"
						type="password"
						bind:value={novaSenha}
						required
						minlength="8"
						placeholder="Pelo menos 8 caracteres"
						class={inputClasse}
					/>
				</div>

				<div>
					<label for="confirmacao" class="block text-sm font-medium text-ink"
						>Confirmar nova senha</label
					>
					<input
						id="confirmacao"
						type="password"
						bind:value={confirmacao}
						required
						minlength="8"
						placeholder="Repita a senha"
						class={inputClasse}
					/>
				</div>

				<button
					type="submit"
					disabled={enviando}
					class="flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
				>
					{enviando ? 'Salvando…' : 'Redefinir senha'}
				</button>
			</form>
		{/if}
	</div>
</div>
