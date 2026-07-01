// Tipos e chamadas da API de prestadores.
// Espelham backend/internal/adapter/http/dto/provider.go

import { apiPost } from './client';

export interface CadastrarProviderRequest {
	nome: string;
	email: string;
	senha: string;
}

export interface CadastrarProviderResponse {
	id: string;
	nome: string;
	email: string;
}

export function cadastrarProvider(
	dados: CadastrarProviderRequest
): Promise<CadastrarProviderResponse> {
	return apiPost<CadastrarProviderRequest, CadastrarProviderResponse>('/providers', dados);
}
