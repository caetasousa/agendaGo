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

	// A equipe é um recurso opcional, ligado em Configurações. Desligada, a API
	// recusa a listagem — voltar ao painel é mais honesto que mostrar um erro
	// sobre algo que a pessoa nem sabe que existe.
	if (!usuario.provider?.permiteEquipe) {
		throw redirect(302, '/painel/configuracoes');
	}

	// Quem opera a agenda vê a equipe; só o dono a modifica. A API impõe isso —
	// aqui a checagem existe para a tela não oferecer botões que dariam 403.
	return {
		equipe: await listarEquipe(),
		ehDono: usuario.provider?.papel === 'dono'
	};
}
