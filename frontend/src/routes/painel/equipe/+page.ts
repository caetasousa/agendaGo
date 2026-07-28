import { redirect } from '@sveltejs/kit';
import { carregarUsuarioDoPainel } from '$lib/auth-guard';
import { listarEquipe, type Equipe } from '$lib/api/membros';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{ equipe: Equipe; ehDono: boolean }> {
	const usuario = await carregarUsuarioDoPainel();

	if (usuario.tipo !== 'provider') {
		throw redirect(302, '/painel');
	}

	// Quem opera a agenda vê a equipe; só o dono a modifica. A API impõe isso —
	// aqui a checagem existe para a tela não oferecer botões que dariam 403.
	return {
		equipe: await listarEquipe(),
		ehDono: usuario.provider?.papel === 'dono'
	};
}
