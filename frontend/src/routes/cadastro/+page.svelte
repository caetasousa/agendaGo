<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import { cadastrarProvider } from '$lib/api/provider';
	import { cadastrarClient, concluirPreCadastro } from '$lib/api/customer';
	import { login, me, urlLoginGoogle } from '$lib/api/auth';
	import { sessao } from '$lib/stores/session.svelte';
	import GoogleIcon from '$lib/components/GoogleIcon.svelte';
	import AuthLayout from '$lib/components/AuthLayout.svelte';

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

	// Quem chega pelo link de pré-cadastro ou pelo link público de agendamento
	// veio para agendar: o tipo é cliente e não pode ser trocado. Nos demais
	// casos, o tipo pode vir pré-escolhido pela landing (?tipo=) ou ficar em
	// aberto — e aí a página mostra a escolha explícita antes do formulário,
	// para ninguém se cadastrar como prestador achando que era cliente.
	const veioParaAgendar =
		preCadastro != null || (page.url.searchParams.get('voltar')?.startsWith('/agendar') ?? false);

	function tipoInicial(): TipoConta | null {
		if (veioParaAgendar) return 'client';
		const t = page.url.searchParams.get('tipo');
		if (t === 'prestador') return 'provider';
		if (t === 'cliente') return 'client';
		return null;
	}

	// svelte-ignore state_referenced_locally
	let tipo = $state<TipoConta | null>(tipoInicial());

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

	function escolher(t: TipoConta) {
		tipo = t;
		erro = null;
	}

	function voltarParaEscolha() {
		tipo = null;
		erro = null;
	}

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

	const rotuloClasse = 'block text-xs font-semibold tracking-wide text-mute uppercase';

	// A manchete fala com quem está na tela: quem vai atender e quem vai agendar
	// têm motivos diferentes para criar conta, e o texto genérico não servia aos
	// dois.
	const copia = $derived.by(() => {
		if (preCadastro) {
			return {
				titulo: 'Falta só a senha.',
				lede:
					'Já reaproveitamos seus dados do agendamento — escolha uma senha e sua conta está pronta.'
			};
		}
		if (tipo === 'provider') {
			return {
				titulo: 'Comece pela sua agenda.',
				lede:
					'Leva menos de um minuto. Você define os horários; seus clientes escolhem entre o que está livre de verdade.'
			};
		}
		if (tipo === 'client') {
			return {
				titulo: 'Escolha um horário e pronto.',
				lede:
					'Você vê só o que está livre de verdade, solicita em segundos e acompanha tudo por aqui — sem precisar ligar para ninguém.'
			};
		}
		return {
			titulo: 'De que lado da agenda você está?',
			lede:
				'São duas contas diferentes: quem publica horários e quem escolhe entre eles. Diga qual é o seu caso.'
		};
	});
</script>

<AuthLayout titulo={copia.titulo} lede={copia.lede}>
	{#if tipo === null}
		<!-- Escolha explícita do tipo antes do formulário: evita que um cliente
		     se cadastre como prestador sem perceber. -->
		<div class="grid gap-3">
			<button
				type="button"
				data-escolher="cliente"
				onclick={() => escolher('client')}
				class="group rounded-xl border border-hairline-strong bg-surface-card p-5 text-left transition hover:border-ink"
			>
				<span class="flex items-center gap-2 text-base font-semibold text-ink">
					<span class="h-2 w-2 rounded-full bg-accent-blue" aria-hidden="true"></span>
					Quero agendar
				</span>
				<span class="mt-1.5 block text-sm text-body">
					Marco com um profissional e acompanho meus horários em um só lugar.
				</span>
			</button>

			<button
				type="button"
				data-escolher="prestador"
				onclick={() => escolher('provider')}
				class="group rounded-xl border border-hairline-strong bg-surface-card p-5 text-left transition hover:border-ink"
			>
				<span class="flex items-center gap-2 text-base font-semibold text-ink">
					<span class="h-2 w-2 rounded-full bg-accent-green" aria-hidden="true"></span>
					Quero oferecer horários
				</span>
				<span class="mt-1.5 block text-sm text-body">
					Publico minha agenda e recebo pedidos de horário dos meus clientes.
				</span>
			</button>
		</div>

		<p class="mt-7 text-sm text-body">
			Já tem conta?
			<a href="/login" class="font-medium text-ink underline">Entrar</a>
		</p>
	{:else if aguardandoConfirmacao}
		<div class="rounded-xl border border-hairline-strong bg-surface-card p-6">
			<p class="text-body">
				Enviamos um email para <span class="font-medium text-ink">{email}</span>. Abra a mensagem e
				clique no link para confirmar seu cadastro e ativar sua conta.
			</p>
			<p class="mt-4 text-sm text-mute">
				Não recebeu? Verifique a caixa de spam. Se este email já tiver uma conta, você receberá
				instruções para entrar.
			</p>
		</div>
	{:else}
		{#if !preCadastro && !veioParaAgendar}
			<!-- Troca de tipo sem sair da tela: o mesmo formulário serve aos dois. -->
			<div
				class="mb-6 inline-flex gap-1 rounded-full border border-hairline-strong bg-surface-card p-1"
			>
				<button
					type="button"
					aria-pressed={tipo === 'client'}
					onclick={() => escolher('client')}
					class="rounded-full px-3.5 py-1 text-sm transition {tipo === 'client'
						? 'bg-surface-elevated font-semibold text-ink'
						: 'text-mute hover:text-ink'}"
				>
					Quero agendar
				</button>
				<button
					type="button"
					aria-pressed={tipo === 'provider'}
					onclick={() => escolher('provider')}
					class="rounded-full px-3.5 py-1 text-sm transition {tipo === 'provider'
						? 'bg-surface-elevated font-semibold text-ink'
						: 'text-mute hover:text-ink'}"
				>
					Quero atender
				</button>
			</div>
		{/if}

		{#if !preCadastro && tipo}
			<a
				href={urlLoginGoogle(tipo, page.url.searchParams.get('voltar') ?? undefined)}
				class="flex h-11 items-center justify-center gap-2 rounded-lg border border-hairline-strong px-4 text-sm font-medium text-ink transition hover:border-ink"
			>
				<GoogleIcon />
				Criar conta com Google
			</a>

			<div class="my-6 flex items-center gap-3">
				<div class="h-px flex-1 bg-hairline-strong"></div>
				<span class="text-xs tracking-wide text-mute uppercase">ou preencha os dados</span>
				<div class="h-px flex-1 bg-hairline-strong"></div>
			</div>
		{/if}

		<form class="space-y-5" novalidate onsubmit={enviar}>
			{#if erro}
				<div
					class="flex items-start gap-2 rounded-md border border-accent-red/40 bg-accent-red/10 p-3 text-sm"
				>
					<span class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent-red"></span>
					<span class="text-body">{erro}</span>
				</div>
			{/if}

			{#if preCadastro}
				<div class="rounded-lg border border-hairline bg-surface-elevated p-4">
					<p class="text-sm text-body">
						<span class="font-medium text-ink">{nome}</span> · {email} · {telefone}
					</p>
				</div>
			{:else}
				<div>
					<label for="nome" class={rotuloClasse}>Nome</label>
					<input
						id="nome"
						type="text"
						bind:value={nome}
						required
						minlength="2"
						maxlength="100"
						placeholder="Seu nome"
						class="campo-linha mt-1"
					/>
				</div>

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

				<div>
					<label for="telefone" class={rotuloClasse}>Telefone</label>
					<input
						id="telefone"
						type="tel"
						bind:value={telefone}
						required
						minlength="8"
						placeholder="(11) 99999-8888"
						class="campo-linha mt-1"
					/>
				</div>
			{/if}

			<div>
				<label for="senha" class={rotuloClasse}>Senha</label>
				<input
					id="senha"
					type="password"
					bind:value={senha}
					required
					minlength="8"
					placeholder="Mínimo de 8 caracteres"
					class="campo-linha mt-1"
				/>
			</div>

			<div>
				<label for="confirmar-senha" class={rotuloClasse}>Confirmar senha</label>
				<input
					id="confirmar-senha"
					type="password"
					bind:value={confirmarSenha}
					required
					minlength="8"
					placeholder="Repita a senha"
					aria-invalid={senhasDivergentes}
					class="campo-linha mt-1"
				/>
				{#if senhasDivergentes}
					<p class="mt-1.5 text-sm text-accent-red">As senhas não coincidem.</p>
				{/if}
			</div>

			<button
				type="submit"
				disabled={enviando || senhasDivergentes}
				class="flex h-11 w-full items-center justify-center rounded-lg bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
			>
				{enviando ? 'Enviando…' : 'Criar conta'}
			</button>

			{#if tipo === 'client' && !preCadastro}
				<p class="text-xs text-mute">
					Enviaremos um email para confirmar seu cadastro antes de ativar a conta.
				</p>
			{/if}
		</form>

		<p class="mt-7 text-sm text-body">
			Já tem conta?
			<a href="/login" class="font-medium text-ink underline">Entrar</a>
		</p>
	{/if}
</AuthLayout>
