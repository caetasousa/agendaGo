import { error } from '@sveltejs/kit';
import { ApiError } from '$lib/api/client';
import { buscarPrestador, type PrestadorResumo } from '$lib/api/provider';

// Página pública: é o link que o prestador compartilha (Instagram, WhatsApp…).
// Convidados sem cadastro veem os horários livres; o login só é exigido na
// hora de solicitar. A API vive em outra origem, então roda só no browser.
export const ssr = false;

export async function load({ params }): Promise<{ prestador: PrestadorResumo }> {
	try {
		const prestador = await buscarPrestador(params.id);
		return { prestador };
	} catch (e) {
		if (e instanceof ApiError && e.status === 404) {
			throw error(404, 'Prestador não encontrado');
		}
		throw e;
	}
}
