// Tipos e chamadas da API de preferências do prestador.
// Espelham backend/internal/adapter/http/dto/provider.go

import { apiPut } from './client';
import type { Bloco } from './availability';

export interface AtualizarPreferenciasRequest {
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
	horariosPadrao: Bloco[];
}

export interface AtualizarPreferenciasResponse {
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
	horariosPadrao: Bloco[];
}

export function atualizarPreferencias(
	dados: AtualizarPreferenciasRequest
): Promise<AtualizarPreferenciasResponse> {
	return apiPut<AtualizarPreferenciasRequest, AtualizarPreferenciasResponse>(
		'/providers/me/preferencias',
		dados
	);
}
