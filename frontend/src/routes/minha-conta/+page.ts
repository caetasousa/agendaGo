import { redirect } from '@sveltejs/kit';
import { carregarUsuarioDoPainel } from '$lib/auth-guard';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{ nome: string; email: string }> {
	const usuario = await carregarUsuarioDoPainel();

	// Esta página trata dos dados de CLIENTE. O prestador tem os dele em
	// Preferências, e a exclusão do lado prestador envolve a agenda inteira —
	// outro fluxo, não este.
	if (usuario.tipo !== 'client') {
		throw redirect(302, '/painel');
	}

	return { nome: usuario.nome, email: usuario.email };
}
