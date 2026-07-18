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
	// Nota livre escrita por quem criou o agendamento, visível às duas partes.
	observacao?: string;
	// Registro que o próprio prestador criou (cliente que ligou, por exemplo):
	// nasce CONFIRMADO, sem pedido para aceitar/recusar, e ele cancela a
	// qualquer momento, sem antecedência mínima.
	marcadoPeloPrestador?: boolean;
}

export interface ListarAgendamentosResponse {
	agendamentos: Agendamento[];
}

export interface DetalheCancelamento {
	nomePrestador: string;
	data: string;
	inicioMinutos: number;
	fimMinutos: number;
	status: StatusAgendamento;
	podeCancelar: boolean;
}

export interface SolicitarAgendamentoRequest {
	providerId: string;
	data: string;
	inicioMinutos: number;
	observacao?: string;
}

export interface SolicitarConvidadoRequest {
	providerId: string;
	data: string;
	inicioMinutos: number;
	nome: string;
	email: string;
	telefone: string;
	observacao?: string;
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

export interface MarcarPeloPrestadorRequest {
	data: string;
	inicioMinutos: number;
	nome: string;
	// Registro puramente interno: sem telefone, sem email, sem notificação.
	observacao?: string;
}

// consultarSlotsDoPrestador devolve os horários livres da agenda do prestador
// autenticado — inclusive com a agenda fechada ao público, já que é o dono
// consultando para marcar um cliente que ligou.
export function consultarSlotsDoPrestador(de: string, ate: string): Promise<SlotsResponse> {
	return apiGet<SlotsResponse>(`/providers/me/slots?de=${de}&ate=${ate}`);
}

// marcarPeloPrestador registra na agenda do prestador autenticado um cliente
// que o contatou por fora (ex.: telefone), com só nome e observação — registro
// puramente interno, sem notificação. Nasce SOLICITADO, como os demais — o
// prestador confirma em seguida na lista de agendamentos.
export function marcarPeloPrestador(dados: MarcarPeloPrestadorRequest): Promise<Agendamento> {
	return apiPost<MarcarPeloPrestadorRequest, Agendamento>('/providers/me/agendamentos', dados);
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

// buscarCancelamento devolve os detalhes do agendamento apontado por um token
// de cancelamento, para a página pública do convidado — rota pública.
export function buscarCancelamento(token: string): Promise<DetalheCancelamento> {
	return apiGet<DetalheCancelamento>(`/agendamentos/cancelar/${token}`);
}

// cancelarPorToken cancela o agendamento pelo token do email, sem login —
// respeita a antecedência mínima de 24h (rota pública).
export function cancelarPorToken(token: string): Promise<void> {
	return apiPostVazio(`/agendamentos/cancelar/${token}`);
}
