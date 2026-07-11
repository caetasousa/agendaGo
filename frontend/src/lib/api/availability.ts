// Tipos e chamadas da API de disponibilidade do prestador.
// Espelham backend/internal/adapter/http/dto/availability.go

import { apiDelete, apiGet, apiPut } from './client';

export interface Bloco {
	inicioMinutos: number;
	fimMinutos: number;
}

// OrigemDia indica de onde veio a disponibilidade resolvida de uma data:
// expediente padrão, dia bloqueado ou horários personalizados.
export type OrigemDia = 'padrao' | 'bloqueio' | 'extra';

export interface DiaAgenda {
	data: string;
	origem: OrigemDia;
	blocos: Bloco[];
}

export interface AgendaResponse {
	aceitaAgendamentos: boolean;
	dias: DiaAgenda[];
}

export interface DefinirDiaRequest {
	tipo: 'bloqueio' | 'extra';
	blocos: Bloco[];
}

// consultarAgenda resolve a disponibilidade de cada dia do período (inclusivo,
// datas YYYY-MM-DD): definição própria da data ou expediente padrão.
export function consultarAgenda(de: string, ate: string): Promise<AgendaResponse> {
	return apiGet<AgendaResponse>(`/providers/me/agenda?de=${de}&ate=${ate}`);
}

// definirDia cria ou substitui a definição própria da data: bloqueio (dia
// indisponível) ou extra (horários personalizados).
export function definirDia(data: string, corpo: DefinirDiaRequest): Promise<DiaAgenda> {
	return apiPut<DefinirDiaRequest, DiaAgenda>(`/providers/me/dias/${data}`, corpo);
}

// removerDia apaga a definição própria da data; o dia volta ao expediente padrão.
export function removerDia(data: string): Promise<void> {
	return apiDelete(`/providers/me/dias/${data}`);
}
