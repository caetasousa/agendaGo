import { redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api/client';
import { me } from '$lib/api/auth';
import {
	listarAgendamentosDoCliente,
	listarAgendamentosDoPrestador,
	type Agendamento
} from '$lib/api/appointments';
import { sessao } from '$lib/stores/session.svelte';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{ tipo: 'provider' | 'client'; agendamentos: Agendamento[] }> {
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

	const tipo = usuario.tipo === 'provider' ? 'provider' : 'client';
	const resposta =
		tipo === 'provider' ? await listarAgendamentosDoPrestador() : await listarAgendamentosDoCliente();

	return { tipo, agendamentos: resposta.agendamentos };
}
