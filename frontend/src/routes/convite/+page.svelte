<script lang="ts">
	import { goto } from '$app/navigation';
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import { aceitarConvite } from '$lib/api/membros';
	import AuthLayout from '$lib/components/AuthLayout.svelte';

	let { data }: { data: PageData } = $props();

	let telefone = $state('');
	let senha = $state('');
	let confirmarSenha = $state('');
	let enviando = $state(false);
	let erro = $state<string | null>(null);
	let pronto = $state(false);

	const senhaCurta = $derived(senha !== '' && senha.length < 8);
	const senhasDiferentes = $derived(confirmarSenha !== '' && senha !== confirmarSenha);
	const podeEnviar = $derived(
		telefone.trim() !== '' && senha.length >= 8 && senha === confirmarSenha && !enviando
	);

	async function enviar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;
		enviando = true;
		try {
			await aceitarConvite({ token: data.token, telefone: telefone.trim(), senha });
			pronto = true;
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível criar seu acesso.';
		} finally {
			enviando = false;
		}
	}

	const rotulo = 'block text-xs font-semibold tracking-wide text-mute uppercase';
	const campo =
		'mt-1.5 w-full rounded-lg border border-hairline-strong bg-surface px-3 py-2 text-sm text-ink';
</script>

{#if data.erro || !data.convite}
	<AuthLayout titulo="Convite inválido" lede="Este link não vale mais.">
		<p class="text-sm text-mute">{data.erro ?? 'Convite não encontrado.'}</p>
		<a href="/" class="mt-6 inline-block text-sm font-semibold text-ink underline-offset-2 hover:underline">
			Voltar ao início
		</a>
	</AuthLayout>
{:else if pronto}
	<AuthLayout
		titulo="Acesso criado"
		lede="Agora é só entrar com o email do convite e a senha que você escolheu."
	>
		<p class="text-sm text-mute">
			Você passou a operar a agenda de <strong class="text-ink">{data.convite.nomeAgenda}</strong>.
		</p>
		<button
			type="button"
			onclick={() => goto('/login')}
			class="mt-6 w-full rounded-lg bg-ink px-4 py-2.5 text-sm font-semibold text-surface"
		>
			Ir para o login
		</button>
	</AuthLayout>
{:else}
	<AuthLayout
		titulo="Você foi convidada"
		lede="Crie seu acesso para ajudar a operar esta agenda."
	>
		<p class="mb-6 text-sm text-mute">
			<strong class="text-ink">{data.convite.nomeAgenda}</strong> convidou
			<strong class="text-ink">{data.convite.email}</strong> para operar a agenda.
		</p>

		{#if erro}
			<p class="mb-4 rounded-lg border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger" role="alert">
				{erro}
			</p>
		{/if}

		<form onsubmit={enviar} class="flex flex-col gap-4">
			<div>
				<label for="telefone" class={rotulo}>Telefone</label>
				<input id="telefone" type="tel" bind:value={telefone} required placeholder="(11) 99999-8888" class={campo} />
			</div>
			<div>
				<label for="senha" class={rotulo}>Senha</label>
				<input id="senha" type="password" bind:value={senha} required minlength="8" class={campo} />
				{#if senhaCurta}
					<p class="mt-1 text-xs text-danger">A senha precisa de ao menos 8 caracteres.</p>
				{/if}
			</div>
			<div>
				<label for="confirmar-senha" class={rotulo}>Confirmar senha</label>
				<input id="confirmar-senha" type="password" bind:value={confirmarSenha} required class={campo} />
				{#if senhasDiferentes}
					<p class="mt-1 text-xs text-danger">As senhas não coincidem.</p>
				{/if}
			</div>
			<button
				type="submit"
				disabled={!podeEnviar}
				class="mt-2 w-full rounded-lg bg-ink px-4 py-2.5 text-sm font-semibold text-surface disabled:opacity-50"
			>
				{enviando ? 'Criando…' : 'Criar meu acesso'}
			</button>
		</form>
	</AuthLayout>
{/if}
