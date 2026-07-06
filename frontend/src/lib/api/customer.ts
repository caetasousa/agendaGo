// Tipos e chamadas da API de clientes.
// Espelham backend/internal/adapter/http/dto/client.go

import { apiPost } from './client';

export interface CadastrarClientRequest {
	nome: string;
	email: string;
	senha: string;
}

export interface CadastrarClientResponse {
	id: string;
	nome: string;
	email: string;
}

export function cadastrarClient(
	dados: CadastrarClientRequest
): Promise<CadastrarClientResponse> {
	return apiPost<CadastrarClientRequest, CadastrarClientResponse>('/clients', dados);
}
