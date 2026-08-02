// Tipos e chamadas da API de preferências do prestador.
// Espelham backend/internal/adapter/http/dto/provider.go

import { apiPut } from './client';
import type { Bloco } from './availability';

export interface AtualizarPreferenciasRequest {
	// Opcional: vazio ou ausente mantém o endereço atual. Trocá-lo quebra os
	// links já compartilhados — a tela avisa antes de salvar.
	slug?: string;
	telefone: string;
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
	duracaoAtendimentoMinutos: number;
	horariosPadrao: Bloco[];
	permiteMarcacaoPeloPrestador: boolean;
}

export interface AtualizarPreferenciasResponse {
	slug: string;
	telefone: string;
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
	duracaoAtendimentoMinutos: number;
	horariosPadrao: Bloco[];
	permiteMarcacaoPeloPrestador: boolean;
}

export function atualizarPreferencias(
	dados: AtualizarPreferenciasRequest
): Promise<AtualizarPreferenciasResponse> {
	return apiPut<AtualizarPreferenciasRequest, AtualizarPreferenciasResponse>(
		'/providers/me/preferencias',
		dados
	);
}
