import { redirect } from '@sveltejs/kit';
import type { MeResponse } from '$lib/api/auth';
import { carregarUsuarioDoPainel } from '$lib/auth-guard';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{ usuario: MeResponse }> {
	const usuario = await carregarUsuarioDoPainel();
	// O admin não tem painel de cliente/prestador: vai direto à moderação.
	if (usuario.tipo === 'admin') {
		throw redirect(302, '/admin');
	}
	return { usuario };
}
