<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import { cadastrarProvider } from '$lib/api/provider';
	import { cadastrarClient, concluirPreCadastro } from '$lib/api/customer';
	import { login, me } from '$lib/api/auth';
	import { sessao } from '$lib/stores/session.svelte';

	let { data }: { data: PageData } = $props();

	// Veio do link "Criar minha conta" do email: os dados do convidado
	// (nome/email/telefone) já chegam prontos, então o formulário só pede a
	// senha — e a conta nasce direto no submit, sem uma segunda confirmação
	// por email (quem tem esse token já provou posse do email).
	// svelte-ignore state_referenced_locally
	const preCadastro = data.preCadastro;
	const tokenPreCadastro = page.url.searchParams.get('pre');

	// destinoAposCadastro honra ?voltar= (ex: link público de agendamento), mas
	// só para caminhos internos — nunca URLs absolutas, para evitar open redirect.
	function destinoAposCadastro(): string {
		const voltar = page.url.searchParams.get('voltar');
		return voltar && voltar.startsWith('/') && !voltar.startsWith('//') ? voltar : '/painel';
	}

	type TipoConta = 'provider' | 'client';

	// Quem chega pelo link público de agendamento, ou com dados de
	// pré-cadastro, veio para agendar: a conta certa é a de cliente.
	// svelte-ignore state_referenced_locally
	let tipo = $state<TipoConta>(
		preCadastro || page.url.searchParams.get('voltar')?.startsWith('/agendar') ? 'client' : 'provider'
	);
	let nome = $state(preCadastro?.nome ?? '');
	let email = $state(preCadastro?.email ?? '');
	let telefone = $state(preCadastro?.telefone ?? '');
	let senha = $state('');
	let confirmarSenha = $state('');

	let enviando = $state(false);
	let erro = $state<string | null>(null);
	// Cliente: após o cadastro, a conta só nasce quando ele confirma pelo email.
	let aguardandoConfirmacao = $state(false);

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
			if (preCadastro && tokenPreCadastro) {
				// já provou posse do email pelo link recebido: cria a conta direto
				// e loga, sem uma segunda confirmação por email
				await concluirPreCadastro(tokenPreCadastro, senha);
				await login({ email, senha });
				sessao.definir(await me());
				goto(destinoAposCadastro());
			} else if (tipo === 'provider') {
				// prestador entra logado direto (sem verificação por email)
				await cadastrarProvider({ nome, email, telefone, senha });
				await login({ email, senha });
				sessao.definir(await me());
				goto(destinoAposCadastro());
			} else {
				// cliente: o backend envia um email de confirmação e responde sempre
				// igual (exista ou não o email). Só entra logado ao confirmar pelo link.
				await cadastrarClient({ nome, email, telefone, senha });
				aguardandoConfirmacao = true;
			}
		} catch (e) {
			// A API é a fonte da verdade da validação: mostramos a mensagem que ela devolve.
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
	<p class="mt-3 text-body">
		{preCadastro
			? 'Falta só a senha — já reaproveitamos seus dados do agendamento.'
			: 'Escolha o tipo de conta para começar.'}
	</p>

	<div class="mt-8 rounded-xl border border-hairline-strong bg-surface-card p-8">
		{#if aguardandoConfirmacao}
			<p class="text-body">
				Enviamos um email para <span class="font-medium text-ink">{email}</span>. Abra a mensagem e
				clique no link para confirmar seu cadastro e ativar sua conta.
			</p>
			<p class="mt-4 text-sm text-mute">
				Não recebeu? Verifique a caixa de spam. Se este email já tiver uma conta, você receberá
				instruções para entrar.
			</p>
		{:else}
		<form class="space-y-5" novalidate onsubmit={enviar}>
			{#if !preCadastro}
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
			{/if}

			{#if erro}
				<div
					class="flex items-start gap-2 rounded-md border border-hairline-strong bg-surface-elevated p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
					<span class="text-body">{erro}</span>
				</div>
			{/if}

			{#if preCadastro}
				<div class="rounded-md border border-hairline bg-surface-elevated p-4">
					<p class="text-sm text-body">
						<span class="font-medium text-ink">{nome}</span> · {email} · {telefone}
					</p>
				</div>
			{:else}
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
					<label for="telefone" class="block text-sm font-medium text-ink">Telefone</label>
					<input
						id="telefone"
						type="tel"
						bind:value={telefone}
						required
						minlength="8"
						placeholder="(11) 99999-8888"
						class={inputClasse}
					/>
				</div>
			{/if}

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
		{/if}
	</div>
</div>
