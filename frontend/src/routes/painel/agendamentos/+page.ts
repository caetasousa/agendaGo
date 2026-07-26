import {
	listarAgendamentosDoCliente,
	listarAgendamentosDoPrestador,
	type Agendamento
} from '$lib/api/appointments';
import { carregarUsuarioDoPainel } from '$lib/auth-guard';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{
	tipo: 'provider' | 'client';
	agendamentos: Agendamento[];
	total: number;
}> {
	const usuario = await carregarUsuarioDoPainel();

	const tipo = usuario.tipo === 'provider' ? 'provider' : 'client';
	// primeira página (limite padrão da API); a tela pede as seguintes sob demanda
	const resposta =
		tipo === 'provider' ? await listarAgendamentosDoPrestador() : await listarAgendamentosDoCliente();

	return { tipo, agendamentos: resposta.agendamentos, total: resposta.total };
}
