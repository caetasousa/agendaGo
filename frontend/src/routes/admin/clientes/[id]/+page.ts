import { error, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api/client';
import { me } from '$lib/api/auth';
import { detalharCliente, type DetalheCliente } from '$lib/api/admin';
import { sessao } from '$lib/stores/session.svelte';
import type { PageLoad } from './$types';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export const load: PageLoad = async ({ params }): Promise<{ cliente: DetalheCliente }> => {
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
		const cliente = await detalharCliente(params.id);
		return { cliente };
	} catch (e) {
		if (e instanceof ApiError && e.status === 404) {
			throw error(404, 'Cliente não encontrado');
		}
		throw e;
	}
};
