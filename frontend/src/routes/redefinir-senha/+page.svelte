<script lang="ts">
	import { page } from '$app/state';
	import { ApiError } from '$lib/api/client';
	import { redefinirSenha } from '$lib/api/auth';
	import AuthLayout from '$lib/components/AuthLayout.svelte';

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

	const rotuloClasse = 'block text-xs font-semibold tracking-wide text-mute uppercase';
</script>

<AuthLayout
	titulo="Escolha uma nova senha."
	lede="Ao salvar, todas as sessões abertas são encerradas — você precisará entrar de novo nos seus outros dispositivos."
>
	{#if !token}
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
			<p class="text-body">
				Link inválido. <a href="/recuperar-senha" class="font-medium text-ink underline"
					>Solicite uma nova recuperação de senha</a
				>.
			</p>
		</div>
	{:else if sucesso}
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
			<p class="text-body">
				Senha redefinida com sucesso. <a href="/login" class="font-medium text-ink underline"
					>Entrar</a
				>
			</p>
		</div>
	{:else}
		<form class="space-y-6" novalidate onsubmit={enviar}>
			{#if erro}
				<div
					class="flex items-start gap-2 rounded-md border border-accent-red/40 bg-accent-red/10 p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
					<span class="text-body">{erro}</span>
				</div>
			{/if}

			<div>
				<label for="novaSenha" class={rotuloClasse}>Nova senha</label>
				<input
					id="novaSenha"
					type="password"
					bind:value={novaSenha}
					required
					minlength="8"
					placeholder="Pelo menos 8 caracteres"
					class="campo-linha mt-1"
				/>
			</div>

			<div>
				<label for="confirmacao" class={rotuloClasse}>Confirmar nova senha</label>
				<input
					id="confirmacao"
					type="password"
					bind:value={confirmacao}
					required
					minlength="8"
					placeholder="Repita a senha"
					class="campo-linha mt-1"
				/>
			</div>

			<button
				type="submit"
				disabled={enviando}
				class="flex h-11 w-full items-center justify-center rounded-lg bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{enviando ? 'Salvando…' : 'Redefinir senha'}
			</button>
		</form>
	{/if}
</AuthLayout>
