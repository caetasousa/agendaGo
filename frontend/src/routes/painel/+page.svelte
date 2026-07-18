<script lang="ts">
	import { page } from '$app/state';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const ehPrestador = $derived(data.usuario.tipo === 'provider');
	const linkAgendamento = $derived(`${page.url.origin}/agendar/${data.usuario.id}`);

	let copiado = $state(false);

	async function copiarLink() {
		try {
			await navigator.clipboard.writeText(linkAgendamento);
			copiado = true;
			setTimeout(() => (copiado = false), 2000);
		} catch {
			// clipboard indisponível (ex: contexto sem HTTPS): o link continua visível para cópia manual
		}
	}
</script>

<div class="mx-auto max-w-2xl">
	<h1 class="display mt-4 text-4xl text-ink sm:text-5xl">Olá, {data.usuario.nome}</h1>
	<p class="mt-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-body">
		<span
			class="inline-flex items-center gap-1.5 rounded-full border border-hairline-strong bg-surface-elevated px-2.5 py-0.5 text-xs font-medium text-ink"
		>
			<span class="h-1.5 w-1.5 rounded-full bg-accent-green"></span>
			{ehPrestador ? 'Conta de prestador' : 'Conta de cliente'}
		</span>
		<span class="text-sm text-mute">{data.usuario.email}</span>
	</p>

	{#if ehPrestador}
		<div
			class="mt-8 flex flex-wrap items-center gap-3 rounded-xl border border-hairline-strong bg-surface-card p-4"
		>
			<div class="min-w-0 flex-1">
				<p class="text-xs font-medium tracking-wide text-mute uppercase">Seu link de agendamento</p>
				<p class="mt-1 truncate text-sm text-ink" data-link-agendamento>{linkAgendamento}</p>
				<p class="mt-1 text-xs text-mute">
					Compartilhe no Instagram ou WhatsApp — quem abrir vê seus horários livres e agenda.
				</p>
			</div>
			<button
				type="button"
				onclick={copiarLink}
				class="inline-flex h-9 items-center rounded-md border border-hairline-strong px-4 text-sm font-medium text-ink transition hover:bg-surface-elevated"
			>
				{copiado ? 'Copiado!' : 'Copiar'}
			</button>
		</div>

		<div class="mt-4 grid gap-4 sm:grid-cols-2">
			<a
				href="/painel/agendamentos"
				class="group rounded-xl border border-hairline-strong bg-surface-card p-6 transition hover:border-ink/40"
			>
				<span class="block h-2 w-2 rounded-full bg-accent-orange" aria-hidden="true"></span>
				<h2 class="mt-4 flex items-center justify-between text-base font-semibold text-ink">
					Agendamentos
					<span class="text-mute transition group-hover:translate-x-0.5 group-hover:text-ink">→</span>
				</h2>
				<p class="mt-2 text-sm text-body">
					Confirme ou recuse solicitações e acompanhe os atendimentos confirmados.
				</p>
			</a>

			{#if data.usuario.permiteMarcacaoPeloPrestador}
				<a
					href="/painel/marcar"
					class="group rounded-xl border border-hairline-strong bg-surface-card p-6 transition hover:border-ink/40"
				>
					<span class="block h-2 w-2 rounded-full bg-accent-red" aria-hidden="true"></span>
					<h2 class="mt-4 flex items-center justify-between text-base font-semibold text-ink">
						Marcar para um cliente
						<span class="text-mute transition group-hover:translate-x-0.5 group-hover:text-ink"
							>→</span
						>
					</h2>
					<p class="mt-2 text-sm text-body">
						Cliente ligou? Registre você mesmo a marcação num horário livre da sua agenda.
					</p>
				</a>
			{/if}

			<a
				href="/painel/disponibilidade"
				class="group rounded-xl border border-hairline-strong bg-surface-card p-6 transition hover:border-ink/40"
			>
				<span class="block h-2 w-2 rounded-full bg-accent-green" aria-hidden="true"></span>
				<h2 class="mt-4 flex items-center justify-between text-base font-semibold text-ink">
					Disponibilidade
					<span class="text-mute transition group-hover:translate-x-0.5 group-hover:text-ink">→</span>
				</h2>
				<p class="mt-2 text-sm text-body">
					Veja o mês no calendário, bloqueie dias ou defina horários próprios para datas
					específicas.
				</p>
			</a>

			<a
				href="/painel/preferencias"
				class="group rounded-xl border border-hairline-strong bg-surface-card p-6 transition hover:border-ink/40"
			>
				<span class="block h-2 w-2 rounded-full bg-accent-blue" aria-hidden="true"></span>
				<h2 class="mt-4 flex items-center justify-between text-base font-semibold text-ink">
					Preferências
					<span class="text-mute transition group-hover:translate-x-0.5 group-hover:text-ink">→</span>
				</h2>
				<p class="mt-2 text-sm text-body">
					Ative a agenda, ajuste a duração dos atendimentos e monte seu expediente padrão.
				</p>
			</a>
		</div>
	{:else}
		<div class="mt-10 grid gap-4 sm:grid-cols-2">
			<a
				href="/painel/agendar"
				class="group rounded-xl border border-hairline-strong bg-surface-card p-6 transition hover:border-ink/40"
			>
				<span class="block h-2 w-2 rounded-full bg-accent-green" aria-hidden="true"></span>
				<h2 class="mt-4 flex items-center justify-between text-base font-semibold text-ink">
					Agendar
					<span class="text-mute transition group-hover:translate-x-0.5 group-hover:text-ink">→</span>
				</h2>
				<p class="mt-2 text-sm text-body">
					Escolha um prestador, veja os horários livres e solicite seu atendimento.
				</p>
			</a>

			<a
				href="/painel/agendamentos"
				class="group rounded-xl border border-hairline-strong bg-surface-card p-6 transition hover:border-ink/40"
			>
				<span class="block h-2 w-2 rounded-full bg-accent-blue" aria-hidden="true"></span>
				<h2 class="mt-4 flex items-center justify-between text-base font-semibold text-ink">
					Meus agendamentos
					<span class="text-mute transition group-hover:translate-x-0.5 group-hover:text-ink">→</span>
				</h2>
				<p class="mt-2 text-sm text-body">
					Acompanhe suas solicitações, veja o que foi confirmado e cancele quando precisar.
				</p>
			</a>
		</div>
	{/if}
</div>
