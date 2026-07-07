// Tipos e chamadas da API de disponibilidade do prestador.
// Espelham backend/internal/adapter/http/dto/availability.go

import { apiDelete, apiGet, apiPost, apiPut } from './client';

export interface Bloco {
	inicioMinutos: number;
	fimMinutos: number;
}

export interface DiaGrade {
	diaSemana: number;
	blocos: Bloco[];
}

export interface GradeSemanalResponse {
	dias: DiaGrade[];
}

export interface Excecao {
	id: string;
	data: string;
	tipo: 'bloqueio' | 'extra';
	blocos: Bloco[];
}

export interface ListarExcecoesResponse {
	excecoes: Excecao[];
}

export interface CriarExcecaoRequest {
	data: string;
	tipo: 'bloqueio' | 'extra';
	blocos: Bloco[];
}

export function consultarGradeSemanal(): Promise<GradeSemanalResponse> {
	return apiGet<GradeSemanalResponse>('/providers/me/disponibilidade');
}

export function definirGradeSemanal(dias: DiaGrade[]): Promise<GradeSemanalResponse> {
	return apiPut<{ dias: DiaGrade[] }, GradeSemanalResponse>('/providers/me/disponibilidade', { dias });
}

export function listarExcecoes(): Promise<ListarExcecoesResponse> {
	return apiGet<ListarExcecoesResponse>('/providers/me/excecoes');
}

export function criarExcecao(dados: CriarExcecaoRequest): Promise<Excecao> {
	return apiPost<CriarExcecaoRequest, Excecao>('/providers/me/excecoes', dados);
}

export function removerExcecao(id: string): Promise<void> {
	return apiDelete(`/providers/me/excecoes/${id}`);
}
