// Tipos e chamadas da API de preferências do prestador.
// Espelham backend/internal/adapter/http/dto/provider.go

import { apiPut } from './client';

export interface AtualizarPreferenciasRequest {
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
}

export interface AtualizarPreferenciasResponse {
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
}

export function atualizarPreferencias(
	dados: AtualizarPreferenciasRequest
): Promise<AtualizarPreferenciasResponse> {
	return apiPut<AtualizarPreferenciasRequest, AtualizarPreferenciasResponse>(
		'/providers/me/preferencias',
		dados
	);
}
