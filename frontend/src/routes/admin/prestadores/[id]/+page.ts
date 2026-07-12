import { error, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api/client';
import { me } from '$lib/api/auth';
import { detalharPrestador, type DetalhePrestador } from '$lib/api/admin';
import { sessao } from '$lib/stores/session.svelte';
import type { PageLoad } from './$types';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export const load: PageLoad = async ({ params }): Promise<{ prestador: DetalhePrestador }> => {
	let usuario;
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

	if (usuario.tipo !== 'admin') {
		throw redirect(302, '/painel');
	}

	try {
		const prestador = await detalharPrestador(params.id);
		return { prestador };
	} catch (e) {
		if (e instanceof ApiError && e.status === 404) {
			throw error(404, 'Prestador não encontrado');
		}
		throw e;
	}
};
