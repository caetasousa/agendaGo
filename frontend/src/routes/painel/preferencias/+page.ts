import { redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api/client';
import { me } from '$lib/api/auth';
import type { Bloco } from '$lib/api/availability';
import { sessao } from '$lib/stores/session.svelte';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
	horariosPadrao: Bloco[];
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

	if (usuario.tipo !== 'provider') {
		throw redirect(302, '/painel');
	}

	return {
		aceitaAgendamentos: usuario.aceitaAgendamentos ?? false,
		descansoMinutos: usuario.descansoMinutos ?? 0,
		horariosPadrao: usuario.horariosPadrao ?? []
	};
}
