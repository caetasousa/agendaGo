<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '$lib/api/client';
	import { login, me, urlLoginGoogle } from '$lib/api/auth';
	import { sessao } from '$lib/stores/session.svelte';
	import GoogleIcon from '$lib/components/GoogleIcon.svelte';
	import AuthLayout from '$lib/components/AuthLayout.svelte';

	// destinoAposLogin honra ?voltar= (ex: link público de agendamento), mas só
	// para caminhos internos — nunca URLs absolutas, para evitar open redirect.
	// Admin vai sempre ao seu painel de moderação, ignorando o ?voltar= (que é
	// do fluxo de agendamento de cliente).
	function destinoAposLogin(tipo: string): string {
		if (tipo === 'admin') return '/admin';
		const voltar = page.url.searchParams.get('voltar');
		return voltar && voltar.startsWith('/') && !voltar.startsWith('//') ? voltar : '/painel';
	}

	let email = $state('');
	let senha = $state('');

	let enviando = $state(false);
	// ?erro=social vem do backend quando o callback do Google falha (state
	// inválido, email não verificado, conta banida) — mensagem genérica de
	// propósito, para não vazar qual dessas condições ocorreu. ?erro=social_outro_tipo
	// é o único caso com mensagem específica: o email já é conta do outro tipo
	// (prestador tentando entrar como cliente, ou vice-versa), então orientar
	// a pessoa a usar o botão certo é uma informação legítima de UX (o mesmo
	// que o cadastro por senha já expõe).
	function mensagemErroSocial(codigo: string | null): string | null {
		if (codigo === 'social_outro_tipo') {
			return 'Esse email já tem conta como outro tipo de usuário (cliente/prestador). Entre pela opção correta.';
		}
		if (codigo === 'social') {
			return 'Não foi possível entrar com o Google.';
		}
		return null;
	}

	let erro = $state<string | null>(mensagemErroSocial(page.url.searchParams.get('erro')));

	const voltar = page.url.searchParams.get('voltar') ?? undefined;

	async function enviar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;
		enviando = true;

		try {
			const usuario = await login({ email, senha });
			// popula a sessão antes de navegar: o destino pode ser uma página
			// pública (ex: link de agendamento) que não tem guard para fazê-lo
			sessao.definir(await me());
			goto(destinoAposLogin(usuario.tipo));
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível entrar.';
		} finally {
			enviando = false;
		}
	}

	const rotuloClasse = 'block text-xs font-semibold tracking-wide text-mute uppercase';
</script>

<AuthLayout
	titulo="Bem-vindo de volta."
	lede="Sua agenda continua exatamente onde você parou — com os horários que você definiu e os pedidos que chegaram enquanto isso."
>
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

		<div>
			<div class="flex items-baseline justify-between gap-3">
				<label for="senha" class={rotuloClasse}>Senha</label>
				<a href="/recuperar-senha" class="text-xs text-mute transition hover:text-ink">
					Esqueci minha senha
				</a>
			</div>
			<input
				id="senha"
				type="password"
				bind:value={senha}
				required
				placeholder="Sua senha"
				class="campo-linha mt-1"
			/>
		</div>

		<button
			type="submit"
			disabled={enviando}
			class="flex h-11 w-full items-center justify-center rounded-lg bg-primary px-4 text-sm font-medium text-primary-on transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
		>
			{enviando ? 'Entrando…' : 'Entrar'}
		</button>
	</form>

	<div class="mt-7 flex items-center gap-3">
		<div class="h-px flex-1 bg-hairline-strong"></div>
		<span class="text-xs tracking-wide text-mute uppercase">ou</span>
		<div class="h-px flex-1 bg-hairline-strong"></div>
	</div>

	<!-- O Google exige saber o tipo de conta antes de iniciar o fluxo: o backend
	     usa caminhos distintos para cliente e prestador. -->
	<div class="mt-4 grid gap-3 sm:grid-cols-2">
		<a
			href={urlLoginGoogle('client', voltar)}
			class="flex h-11 items-center justify-center gap-2 rounded-lg border border-hairline-strong px-4 text-sm font-medium text-ink transition hover:border-ink"
		>
			<GoogleIcon />
			Sou cliente
		</a>
		<a
			href={urlLoginGoogle('provider', voltar)}
			class="flex h-11 items-center justify-center gap-2 rounded-lg border border-hairline-strong px-4 text-sm font-medium text-ink transition hover:border-ink"
		>
			<GoogleIcon />
			Sou prestador
		</a>
	</div>

	<p class="mt-7 text-sm text-body">
		Ainda não tem conta?
		<a
			href="/cadastro{voltar ? `?voltar=${encodeURIComponent(voltar)}` : ''}"
			class="font-medium text-ink underline">Cadastre-se</a
		>
	</p>
</AuthLayout>
