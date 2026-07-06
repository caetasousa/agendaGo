import { redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api/client';
import { me, type MeResponse } from '$lib/api/auth';
import { sessao } from '$lib/stores/session.svelte';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{ usuario: MeResponse }> {
	try {
		const usuario = await me();
		sessao.definir(usuario);
		return { usuario };
	} catch (e) {
		if (e instanceof ApiError && e.status === 401) {
			sessao.limpar();
			throw redirect(302, '/login');
		}
		throw e;
	}
}
