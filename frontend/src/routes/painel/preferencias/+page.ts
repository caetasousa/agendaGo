import { redirect } from '@sveltejs/kit';
import type { Bloco } from '$lib/api/availability';
import { carregarUsuarioDoPainel } from '$lib/auth-guard';

// O cookie de sessão é HttpOnly e a API vive em outra origem, então o SSR
// nunca teria acesso a ele — a checagem de autenticação só pode rodar no browser.
export const ssr = false;

export async function load(): Promise<{
	telefone: string;
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
	duracaoAtendimentoMinutos: number;
	horariosPadrao: Bloco[];
	permiteMarcacaoPeloPrestador: boolean;
	telefonePendente: boolean;
}> {
	// permite carregar mesmo com telefone pendente — é aqui que ele é resolvido
	const usuario = await carregarUsuarioDoPainel(true);

	if (usuario.tipo !== 'provider') {
		throw redirect(302, '/painel');
	}

	return {
		// telefone pendente é um placeholder técnico, não um valor real — o
		// campo começa vazio para o prestador digitar o telefone de verdade
		telefone: usuario.telefonePendente ? '' : (usuario.telefone ?? ''),
		aceitaAgendamentos: usuario.aceitaAgendamentos ?? false,
		descansoMinutos: usuario.descansoMinutos ?? 0,
		duracaoAtendimentoMinutos: usuario.duracaoAtendimentoMinutos ?? 60,
		horariosPadrao: usuario.horariosPadrao ?? [],
		permiteMarcacaoPeloPrestador: usuario.permiteMarcacaoPeloPrestador ?? true,
		telefonePendente: usuario.telefonePendente ?? false
	};
}
