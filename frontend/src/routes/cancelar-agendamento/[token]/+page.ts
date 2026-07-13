// Mesmo motivo do /login: sem SSR, a página só renderiza após o JS hidratar.
// O load busca os detalhes do agendamento pelo token e trata token inválido
// como 404, para a página mostrar o estado de "link inválido".
import { error } from '@sveltejs/kit';
import { ApiError } from '$lib/api/client';
import { buscarCancelamento, type DetalheCancelamento } from '$lib/api/appointments';

export const ssr = false;

export async function load({
	params
}): Promise<{ token: string; detalhe: DetalheCancelamento }> {
	try {
		const detalhe = await buscarCancelamento(params.token);
		return { token: params.token, detalhe };
	} catch (e) {
		if (e instanceof ApiError && e.status === 404) {
			throw error(404, 'Link de cancelamento inválido');
		}
		throw e;
	}
}
