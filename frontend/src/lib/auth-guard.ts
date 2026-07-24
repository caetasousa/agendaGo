// Guard de autenticação compartilhado pelos load() das páginas do painel:
// consulta /auth/me, trata 401 (desloga e redireciona ao login), popula a
// store de sessão, e trava o prestador com telefone pendente (login social
// que ainda não confirmou um telefone real) em /painel/preferencias — só essa
// página pode carregar até o telefone ser salvo.
import { redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api/client';
import { me, type MeResponse } from '$lib/api/auth';
import { sessao } from '$lib/stores/session.svelte';

// permitirTelefonePendente é true só na própria página de Preferências —
// nas demais, o prestador é redirecionado antes de carregar qualquer dado.
export async function carregarUsuarioDoPainel(permitirTelefonePendente = false): Promise<MeResponse> {
	let usuario: MeResponse;
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

	if (usuario.tipo === 'provider' && usuario.telefonePendente && !permitirTelefonePendente) {
		throw redirect(302, '/painel/preferencias');
	}

	return usuario;
}
