import { redirect } from '@sveltejs/kit';
import { carregarUsuarioDoPainel } from '$lib/auth-guard';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<void> {
	const usuario = await carregarUsuarioDoPainel();

	if (usuario.tipo !== 'provider') {
		throw redirect(302, '/painel');
	}
}
