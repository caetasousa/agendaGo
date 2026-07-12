<script lang="ts">
	import { ApiError } from '$lib/api/client';
	import { recuperarSenha } from '$lib/api/auth';

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

	const inputClasse =
		'mt-2 h-10 w-full rounded-md border border-hairline-strong bg-surface-card px-3.5 text-sm text-ink outline-none transition placeholder:text-mute focus:border-ink';
</script>

<div class="mx-auto max-w-xl">
	<a href="/login" class="text-sm text-mute transition hover:text-ink">← Voltar</a>

	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Recuperar senha</h1>
	<p class="mt-3 text-body">Informe seu email para receber um link de redefinição de senha.</p>

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-8">
		{#if enviado}
			<p class="text-body">
				Se este email estiver cadastrado, você receberá as instruções para redefinir sua senha.
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

				<button
					type="submit"
					disabled={enviando}
					class="flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
				>
					{enviando ? 'Enviando…' : 'Enviar link de recuperação'}
				</button>
			</form>
		{/if}
	</div>
</div>
