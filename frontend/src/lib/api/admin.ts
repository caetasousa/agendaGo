// Tipos e chamadas da API de moderação (admin).
// Espelham backend/internal/adapter/http/dto/admin.go

import { apiGet, apiPostVazio } from './client';

export interface UsuarioModeracao {
	id: string;
	nome: string;
	email: string;
	ativo: boolean;
	aceitaAgendamentos: boolean;
}

export interface ListarUsuariosResponse {
	usuarios: UsuarioModeracao[];
}

// AgendamentoAdmin é um agendamento na visão de detalhe do admin. Na visão do
// prestador vem o contato do cliente; na visão do cliente, o nome do prestador.
export interface AgendamentoAdmin {
	id: string;
	data: string;
	inicioMinutos: number;
	fimMinutos: number;
	status: string;
	nomeCliente?: string;
	emailCliente?: string;
	telefoneCliente?: string;
	nomePrestador?: string;
}

export interface DetalhePrestador {
	id: string;
	nome: string;
	email: string;
	ativo: boolean;
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
	duracaoAtendimentoMinutos: number;
	agendamentos: AgendamentoAdmin[];
}

export interface DetalheCliente {
	id: string;
	nome: string;
	email: string;
	telefone?: string;
	ativo: boolean;
	temConta: boolean;
	agendamentos: AgendamentoAdmin[];
}

// listarPrestadores devolve todos os prestadores com o status de moderação.
export function listarPrestadores(): Promise<ListarUsuariosResponse> {
	return apiGet<ListarUsuariosResponse>('/admin/prestadores');
}

// listarClientes devolve os clientes com conta e o status de moderação.
export function listarClientes(): Promise<ListarUsuariosResponse> {
	return apiGet<ListarUsuariosResponse>('/admin/clientes');
}

// detalharPrestador devolve os dados cadastrais do prestador e os agendamentos
// que ele recebeu (leitura).
export function detalharPrestador(id: string): Promise<DetalhePrestador> {
	return apiGet<DetalhePrestador>(`/admin/prestadores/${id}`);
}

// detalharCliente devolve os dados cadastrais do cliente e os agendamentos que
// ele fez (leitura).
export function detalharCliente(id: string): Promise<DetalheCliente> {
	return apiGet<DetalheCliente>(`/admin/clientes/${id}`);
}

export function banirPrestador(id: string): Promise<void> {
	return apiPostVazio(`/admin/prestadores/${id}/banir`);
}

export function reativarPrestador(id: string): Promise<void> {
	return apiPostVazio(`/admin/prestadores/${id}/reativar`);
}

export function banirCliente(id: string): Promise<void> {
	return apiPostVazio(`/admin/clientes/${id}/banir`);
}

export function reativarCliente(id: string): Promise<void> {
	return apiPostVazio(`/admin/clientes/${id}/reativar`);
}
