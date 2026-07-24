import { redirect } from '@sveltejs/kit';
import { listarPrestadores, type PrestadorResumo } from '$lib/api/provider';
import { carregarUsuarioDoPainel } from '$lib/auth-guard';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{ prestadores: PrestadorResumo[] }> {
	const usuario = await carregarUsuarioDoPainel();

	if (usuario.tipo !== 'client') {
		throw redirect(302, '/painel');
	}

	const resposta = await listarPrestadores();
	return { prestadores: resposta.prestadores };
}
