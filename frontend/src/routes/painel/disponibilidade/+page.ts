import { redirect } from '@sveltejs/kit';
import { consultarAgenda, type AgendaResponse } from '$lib/api/availability';
import { chaveData } from '$lib/holidays';
import { carregarUsuarioDoPainel } from '$lib/auth-guard';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{ agenda: AgendaResponse }> {
	const usuario = await carregarUsuarioDoPainel();

	if (usuario.tipo !== 'provider') {
		throw redirect(302, '/painel');
	}

	const hoje = new Date();
	const de = chaveData(new Date(hoje.getFullYear(), hoje.getMonth(), 1));
	const ate = chaveData(new Date(hoje.getFullYear(), hoje.getMonth() + 1, 0));
	const agenda = await consultarAgenda(de, ate);

	return { agenda };
}
