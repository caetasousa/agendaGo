<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { PageData } from './$types';
	import { ApiError } from '$lib/api/client';
	import { convidarMembro, removerMembro, cancelarConvite } from '$lib/api/membros';
	import PageHeader from '$lib/components/PageHeader.svelte';

	let { data }: { data: PageData } = $props();

	let email = $state('');
	let enviando = $state(false);
	let erro = $state<string | null>(null);
	let sucesso = $state<string | null>(null);
	let removendo = $state<string | null>(null);

	const emailInvalido = $derived(email.trim() !== '' && !email.includes('@'));

	async function convidar(evento: SubmitEvent) {
		evento.preventDefault();
		erro = null;
		sucesso = null;
		enviando = true;
		try {
			const destinatario = email.trim();
			await convidarMembro({ email: destinatario });
			sucesso = `Convite enviado para ${destinatario}.`;
			email = '';
			await invalidateAll();
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível enviar o convite.';
		} finally {
			enviando = false;
		}
	}

	async function remover(id: string, quem: string) {
		if (!confirm(`Remover o acesso de ${quem} a esta agenda?`)) return;
		erro = null;
		sucesso = null;
		removendo = id;
		try {
			await removerMembro(id);
			sucesso = `${quem} não tem mais acesso.`;
			await invalidateAll();
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível remover o acesso.';
		} finally {
			removendo = null;
		}
	}

	async function cancelar(emailConvidado: string) {
		erro = null;
		sucesso = null;
		removendo = emailConvidado;
		try {
			await cancelarConvite(emailConvidado);
			sucesso = `Convite para ${emailConvidado} cancelado.`;
			await invalidateAll();
		} catch (e) {
			erro = e instanceof ApiError ? e.message : 'Não foi possível cancelar o convite.';
		} finally {
			removendo = null;
		}
	}

	function dataCurta(iso: string): string {
		return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' });
	}

	const cartao = 'rounded-xl border border-hairline-strong bg-surface-card p-6';
	const rotulo = 'block text-xs font-semibold tracking-wide text-mute uppercase';
</script>

<PageHeader titulo="Equipe" descricao="Quem pode operar esta agenda além de você." />

{#if erro}
	<p class="mb-4 rounded-lg border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger" role="alert">
		{erro}
	</p>
{/if}
{#if sucesso}
	<p class="mb-4 rounded-lg border border-ok/30 bg-ok/10 px-4 py-3 text-sm text-ok" role="status">
		{sucesso}
	</p>
{/if}

<div class="flex flex-col gap-6">
	{#if data.ehDono}
		<section class={cartao}>
			<h2 class="text-base font-semibold text-ink">Convidar alguém</h2>
			<p class="mt-1 mb-4 text-sm text-mute">
				A pessoa recebe um email, escolhe a própria senha e passa a operar esta agenda. Ela não
				precisa ter conta no agendaGo — e não pode usar um email que já tenha uma.
			</p>
			<form onsubmit={convidar} class="flex flex-col gap-3 sm:flex-row sm:items-end">
				<div class="flex-1">
					<label for="email-convite" class={rotulo}>Email</label>
					<input
						id="email-convite"
						type="email"
						bind:value={email}
						required
						placeholder="recepcao@email.com"
						class="mt-1.5 w-full rounded-lg border border-hairline-strong bg-surface px-3 py-2 text-sm text-ink"
					/>
				</div>
				<button
					type="submit"
					disabled={enviando || email.trim() === '' || emailInvalido}
					class="rounded-lg bg-ink px-4 py-2 text-sm font-semibold text-surface disabled:opacity-50"
				>
					{enviando ? 'Enviando…' : 'Enviar convite'}
				</button>
			</form>
		</section>
	{/if}

	<section class={cartao}>
		<h2 class="text-base font-semibold text-ink">Com acesso</h2>
		<ul class="mt-4 flex flex-col divide-y divide-hairline">
			{#each data.equipe.membros as membro (membro.id)}
				<li class="flex flex-wrap items-center gap-x-3 gap-y-1 py-3" data-membro={membro.id}>
					<span class="min-w-0 flex-1 truncate text-sm text-ink">{membro.email}</span>
					<span class="rounded-full border border-hairline-strong px-2 py-0.5 text-xs text-mute">
						{membro.ehDono ? 'Dona da agenda' : 'Operadora'}
					</span>
					{#if !membro.ativo}
						<span class="text-xs text-danger">conta desativada</span>
					{/if}
					{#if data.ehDono && !membro.ehDono}
						<button
							type="button"
							onclick={() => remover(membro.id, membro.email)}
							disabled={removendo === membro.id}
							class="text-xs font-semibold text-danger underline-offset-2 hover:underline disabled:opacity-50"
						>
							Remover acesso
						</button>
					{/if}
				</li>
			{/each}
		</ul>
	</section>

	{#if data.equipe.pendentes.length > 0}
		<section class={cartao}>
			<h2 class="text-base font-semibold text-ink">Convites enviados</h2>
			<p class="mt-1 text-sm text-mute">Ainda não aceitos.</p>
			<ul class="mt-4 flex flex-col divide-y divide-hairline">
				{#each data.equipe.pendentes as convite (convite.email)}
					<li class="flex flex-wrap items-center gap-x-3 gap-y-1 py-3" data-convite={convite.email}>
						<span class="min-w-0 flex-1 truncate text-sm text-ink">{convite.email}</span>
						<span class="text-xs text-mute">expira {dataCurta(convite.expiraEm)}</span>
						{#if data.ehDono}
							<button
								type="button"
								onclick={() => cancelar(convite.email)}
								disabled={removendo === convite.email}
								class="text-xs font-semibold text-danger underline-offset-2 hover:underline disabled:opacity-50"
							>
								Cancelar
							</button>
						{/if}
					</li>
				{/each}
			</ul>
		</section>
	{/if}
</div>
