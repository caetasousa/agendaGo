// Tipos e chamadas da API de moderação (admin).
// Espelham backend/internal/adapter/http/dto/admin.go

import { apiGet, apiPostVazio } from './client';
import { queryDaPagina, type Pagina, type Paginacao } from './paginacao';

export interface UsuarioModeracao {
	id: string;
	nome: string;
	email: string;
	ativo: boolean;
	aceitaAgendamentos: boolean;
}

export interface ListarUsuariosResponse extends Paginacao {
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

export interface DetalhePrestador extends Paginacao {
	id: string;
	nome: string;
	email: string;
	ativo: boolean;
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
	duracaoAtendimentoMinutos: number;
	agendamentos: AgendamentoAdmin[];
}

export interface DetalheCliente extends Paginacao {
	id: string;
	nome: string;
	email: string;
	telefone?: string;
	ativo: boolean;
	temConta: boolean;
	agendamentos: AgendamentoAdmin[];
}

// listarPrestadores devolve uma página de prestadores com o status de moderação.
export function listarPrestadores(pagina?: Pagina): Promise<ListarUsuariosResponse> {
	return apiGet<ListarUsuariosResponse>(`/admin/prestadores${queryDaPagina(pagina)}`);
}

// listarClientes devolve uma página de clientes com conta e o status de moderação.
export function listarClientes(pagina?: Pagina): Promise<ListarUsuariosResponse> {
	return apiGet<ListarUsuariosResponse>(`/admin/clientes${queryDaPagina(pagina)}`);
}

// detalharPrestador devolve os dados cadastrais do prestador e uma página dos
// agendamentos que ele recebeu (leitura).
export function detalharPrestador(id: string, pagina?: Pagina): Promise<DetalhePrestador> {
	return apiGet<DetalhePrestador>(`/admin/prestadores/${id}${queryDaPagina(pagina)}`);
}

// detalharCliente devolve os dados cadastrais do cliente e uma página dos
// agendamentos que ele fez (leitura).
export function detalharCliente(id: string, pagina?: Pagina): Promise<DetalheCliente> {
	return apiGet<DetalheCliente>(`/admin/clientes/${id}${queryDaPagina(pagina)}`);
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
