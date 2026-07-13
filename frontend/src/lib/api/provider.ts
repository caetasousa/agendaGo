// Tipos e chamadas da API de prestadores.
// Espelham backend/internal/adapter/http/dto/provider.go

import { apiGet, apiPost } from './client';

export interface CadastrarProviderRequest {
	nome: string;
	email: string;
	telefone: string;
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

export interface PrestadorResumo {
	id: string;
	nome: string;
	duracaoAtendimentoMinutos: number;
	aceitaAgendamentos: boolean;
}

export interface ListarPrestadoresResponse {
	prestadores: PrestadorResumo[];
}

// listarPrestadores devolve todos os prestadores da vitrine — rota pública.
// Quem está com a agenda desativada aparece sem horários.
export function listarPrestadores(): Promise<ListarPrestadoresResponse> {
	return apiGet<ListarPrestadoresResponse>('/providers');
}

// buscarPrestador devolve a identificação pública de um prestador — usada
// pela página de agendamento acessada via link direto.
export function buscarPrestador(id: string): Promise<PrestadorResumo> {
	return apiGet<PrestadorResumo>(`/providers/${id}`);
}
