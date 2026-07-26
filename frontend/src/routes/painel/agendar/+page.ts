import { redirect } from '@sveltejs/kit';
import { listarPrestadores, type PrestadorResumo } from '$lib/api/provider';
import { carregarUsuarioDoPainel } from '$lib/auth-guard';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{ prestadores: PrestadorResumo[]; total: number }> {
	const usuario = await carregarUsuarioDoPainel();

	if (usuario.tipo !== 'client') {
		throw redirect(302, '/painel');
	}

	// primeira página da vitrine (limite padrão da API); a tela pede as
	// seguintes sob demanda
	const resposta = await listarPrestadores();
	return { prestadores: resposta.prestadores, total: resposta.total };
}
