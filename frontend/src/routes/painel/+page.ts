import { redirect } from '@sveltejs/kit';
import type { MeResponse } from '$lib/api/auth';
import {
	listarAgendamentosDoCliente,
	listarAgendamentosDoPrestador,
	type Agendamento
} from '$lib/api/appointments';
import { buscarMetricas, type Metricas } from '$lib/api/metricas';
import { carregarUsuarioDoPainel } from '$lib/auth-guard';
import { chaveData } from '$lib/holidays';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

// diasDoResumo é a janela do resumo analítico do prestador: um mês de agenda,
// longo o bastante para a ocupação significar alguma coisa e curto o bastante
// para refletir como ele trabalha hoje.
const diasDoResumo = 30;

export async function load(): Promise<{
	usuario: MeResponse;
	agendamentos: Agendamento[];
	metricas: Metricas | null;
}> {
	const usuario = await carregarUsuarioDoPainel();
	// O admin não tem painel de cliente/prestador: vai direto à moderação.
	if (usuario.tipo === 'admin') {
		throw redirect(302, '/admin');
	}

	if (usuario.tipo !== 'provider') {
		const resposta = await listarAgendamentosDoCliente();
		return { usuario, agendamentos: resposta.agendamentos, metricas: null };
	}

	const hoje = new Date();
	const inicio = new Date(hoje);
	inicio.setDate(hoje.getDate() - (diasDoResumo - 1));

	const [resposta, metricas] = await Promise.all([
		listarAgendamentosDoPrestador(),
		buscarMetricas(chaveData(inicio), chaveData(hoje))
	]);

	return { usuario, agendamentos: resposta.agendamentos, metricas };
}
