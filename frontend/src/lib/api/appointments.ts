// Tipos e chamadas da API de agendamentos.
// Espelham backend/internal/adapter/http/dto/appointment.go

import { apiGet, apiPost, apiPostVazio } from './client';

// StatusAgendamento é o ciclo de vida da reserva:
// SOLICITADO → CONFIRMADO → REALIZADO, com saídas para
// RECUSADO, EXPIRADO, CANCELADO e NAO_COMPARECEU.
export type StatusAgendamento =
	| 'SOLICITADO'
	| 'CONFIRMADO'
	| 'REALIZADO'
	| 'RECUSADO'
	| 'EXPIRADO'
	| 'CANCELADO'
	| 'NAO_COMPARECEU';

export interface Agendamento {
	id: string;
	data: string;
	inicioMinutos: number;
	fimMinutos: number;
	status: StatusAgendamento;
	expiraEm: string;
	nomeCliente?: string;
	// Contato do cliente — só vem preenchido na visão do prestador.
	emailCliente?: string;
	telefoneCliente?: string;
	nomePrestador?: string;
}

export interface ListarAgendamentosResponse {
	agendamentos: Agendamento[];
}

export interface SolicitarAgendamentoRequest {
	providerId: string;
	data: string;
	inicioMinutos: number;
}

export interface SolicitarConvidadoRequest {
	providerId: string;
	data: string;
	inicioMinutos: number;
	nome: string;
	email: string;
	telefone: string;
}

export interface Slot {
	inicioMinutos: number;
	fimMinutos: number;
}

export interface DiaSlots {
	data: string;
	slots: Slot[];
}

export interface SlotsResponse {
	dias: DiaSlots[];
}

// consultarSlots devolve os horários livres ofertáveis do prestador no
// período (inclusivo, datas YYYY-MM-DD) — rota pública.
export function consultarSlots(providerId: string, de: string, ate: string): Promise<SlotsResponse> {
	return apiGet<SlotsResponse>(`/providers/${providerId}/slots?de=${de}&ate=${ate}`);
}

// solicitarAgendamento reserva um slot livre para o cliente autenticado; a
// solicitação já ocupa o intervalo até o prestador responder ou expirar.
export function solicitarAgendamento(dados: SolicitarAgendamentoRequest): Promise<Agendamento> {
	return apiPost<SolicitarAgendamentoRequest, Agendamento>('/agendamentos', dados);
}

// solicitarConvidado reserva um slot sem cadastro: cria (ou reusa) um cliente
// convidado a partir do nome/email/telefone informados — rota pública.
export function solicitarConvidado(dados: SolicitarConvidadoRequest): Promise<Agendamento> {
	return apiPost<SolicitarConvidadoRequest, Agendamento>('/agendamentos/convidado', dados);
}

// listarAgendamentosDoCliente lista os agendamentos feitos pelo cliente autenticado.
export function listarAgendamentosDoCliente(): Promise<ListarAgendamentosResponse> {
	return apiGet<ListarAgendamentosResponse>('/clients/me/agendamentos');
}

// listarAgendamentosDoPrestador lista os agendamentos recebidos pelo prestador autenticado.
export function listarAgendamentosDoPrestador(): Promise<ListarAgendamentosResponse> {
	return apiGet<ListarAgendamentosResponse>('/providers/me/agendamentos');
}

// confirmarAgendamento aceita uma solicitação pendente (prestador).
export function confirmarAgendamento(id: string): Promise<void> {
	return apiPostVazio(`/agendamentos/${id}/confirmar`);
}

// recusarAgendamento nega uma solicitação pendente (prestador).
export function recusarAgendamento(id: string): Promise<void> {
	return apiPostVazio(`/agendamentos/${id}/recusar`);
}

// cancelarAgendamento encerra um agendamento (cliente ou prestador; confirmado
// exige antecedência mínima de 24h).
export function cancelarAgendamento(id: string): Promise<void> {
	return apiPostVazio(`/agendamentos/${id}/cancelar`);
}

// marcarRealizado conclui um agendamento confirmado cujo horário já passou (prestador).
export function marcarRealizado(id: string): Promise<void> {
	return apiPostVazio(`/agendamentos/${id}/realizado`);
}

// marcarNaoCompareceu registra a ausência do cliente (prestador).
export function marcarNaoCompareceu(id: string): Promise<void> {
	return apiPostVazio(`/agendamentos/${id}/nao-compareceu`);
}
