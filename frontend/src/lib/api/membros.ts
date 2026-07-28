// Tipos e chamadas da API de equipe: quem opera a agenda e os convites.
// Espelham backend/internal/adapter/http/dto/membro.go

import { apiGet, apiPost, apiPostSemResposta, apiDelete } from './client';

export interface Membro {
	id: string;
	email: string;
	papel: string;
	ativo: boolean;
	ehDono: boolean;
	criadoEm: string;
}

export interface ConvitePendente {
	email: string;
	papel: string;
	expiraEm: string;
}

export interface Equipe {
	membros: Membro[];
	pendentes: ConvitePendente[];
}

export interface Convite {
	email: string;
	nomeAgenda: string;
	papel: string;
}

export interface ConvidarRequest {
	email: string;
	papel?: string;
}

export interface AceitarConviteRequest {
	token: string;
	telefone: string;
	senha: string;
}

export function listarEquipe(): Promise<Equipe> {
	return apiGet<Equipe>('/providers/me/membros');
}

export function convidarMembro(dados: ConvidarRequest): Promise<void> {
	return apiPostSemResposta<ConvidarRequest>('/providers/me/membros', dados);
}

export function removerMembro(id: string): Promise<void> {
	return apiDelete(`/providers/me/membros/${id}`);
}

export function cancelarConvite(email: string): Promise<void> {
	return apiDelete(`/providers/me/convites/${encodeURIComponent(email)}`);
}

// Rotas públicas: quem foi convidado ainda não tem conta, e é o aceite que a cria.
export function consultarConvite(token: string): Promise<Convite> {
	return apiGet<Convite>(`/membros/convite?token=${encodeURIComponent(token)}`);
}

export function aceitarConvite(dados: AceitarConviteRequest): Promise<Convite> {
	return apiPost<AceitarConviteRequest, Convite>('/membros/aceitar-convite', dados);
}
