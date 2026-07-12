import { redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api/client';
import { me } from '$lib/api/auth';
import { listarClientes, listarPrestadores, type UsuarioModeracao } from '$lib/api/admin';
import { sessao } from '$lib/stores/session.svelte';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{
	prestadores: UsuarioModeracao[];
	clientes: UsuarioModeracao[];
}> {
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

	const [prestadores, clientes] = await Promise.all([listarPrestadores(), listarClientes()]);
	return { prestadores: prestadores.usuarios, clientes: clientes.usuarios };
}
