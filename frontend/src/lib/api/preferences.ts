// Tipos e chamadas da API de preferências do prestador.
// Espelham backend/internal/adapter/http/dto/provider.go

import { apiPut } from './client';
import type { Bloco } from './availability';

export interface AtualizarPreferenciasRequest {
	telefone: string;
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
	duracaoAtendimentoMinutos: number;
	horariosPadrao: Bloco[];
	permiteMarcacaoPeloPrestador: boolean;
}

export interface AtualizarPreferenciasResponse {
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
