<script lang="ts">
	import { ApiError } from '$lib/api/client';
	import { recuperarSenha } from '$lib/api/auth';
	import AuthLayout from '$lib/components/AuthLayout.svelte';

	let email = $state('');

	let enviando = $state(false);
	let erro = $state<string | null>(null);
	let enviado = $state(false);

	async function enviar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;
		enviando = true;

		try {
			await recuperarSenha({ email });
			// resposta é sempre a mesma, exista ou não a conta com esse email —
			// não dá para diferenciar no frontend, e não deveria
			enviado = true;
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível enviar o email.';
		} finally {
			enviando = false;
		}
	}

	const rotuloClasse = 'block text-xs font-semibold tracking-wide text-mute uppercase';
</script>

<AuthLayout
	titulo="Vamos recuperar seu acesso."
	lede="Informe o email da sua conta e enviamos um link para você definir uma nova senha. O link vale por uma hora."
>
	{#if enviado}
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
			<p class="text-body">
				Se este email estiver cadastrado, você receberá as instruções para redefinir sua senha.
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
				<label for="email" class={rotuloClasse}>E-mail</label>
				<input
					id="email"
					type="email"
					bind:value={email}
					required
					placeholder="voce@exemplo.com"
					class="campo-linha mt-1"
				/>
			</div>

			<button
				type="submit"
				disabled={enviando}
				class="flex h-11 w-full items-center justify-center rounded-lg bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{enviando ? 'Enviando…' : 'Enviar link de recuperação'}
			</button>
		</form>
	{/if}

	<p class="mt-7 text-sm text-body">
		Lembrou a senha?
		<a href="/login" class="font-medium text-ink underline">Entrar</a>
	</p>
</AuthLayout>
