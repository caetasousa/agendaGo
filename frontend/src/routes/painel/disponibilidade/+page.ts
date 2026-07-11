import { redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api/client';
import { me } from '$lib/api/auth';
import { consultarAgenda, type AgendaResponse } from '$lib/api/availability';
import { chaveData } from '$lib/holidays';
import { sessao } from '$lib/stores/session.svelte';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{ agenda: AgendaResponse }> {
	let usuario;
	try {
		usuario = await me();
	} catch (e) {
		if (e instanceof ApiError && e.status === 401) {
			sessao.limpar();
			throw redirect(302, '/login');
		}
		throw e;
	}

	sessao.definir(usuario);

	if (usuario.tipo !== 'provider') {
		throw redirect(302, '/painel');
	}

	const hoje = new Date();
	const de = chaveData(new Date(hoje.getFullYear(), hoje.getMonth(), 1));
	const ate = chaveData(new Date(hoje.getFullYear(), hoje.getMonth() + 1, 0));
	const agenda = await consultarAgenda(de, ate);

	return { agenda };
}
