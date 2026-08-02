// Tipos e chamadas da API de compromissos pessoais do prestador.
// Espelham backend/internal/adapter/http/dto/ocupacao.go

import { apiDelete, apiGet, apiPost } from './client';

// Origem indica de onde veio o compromisso. Hoje só existe o criado à mão,
// mas o campo vem do backend porque é o que permitirá distinguir compromisso
// importado de outro calendário sem mudar este contrato.
export type OrigemOcupacao = 'manual';

export interface Ocupacao {
	id: string;
	data: string;
	inicioMinutos: number;
	fimMinutos: number;
	titulo?: string;
	origem: OrigemOcupacao;
}

export interface OcupacoesResponse {
	ocupacoes: Ocupacao[];
}

export interface CriarOcupacaoRequest {
	data: string;
	inicioMinutos: number;
	fimMinutos: number;
	titulo: string;
}

export function listarOcupacoes(de: string, ate: string): Promise<OcupacoesResponse> {
	return apiGet<OcupacoesResponse>(
		`/providers/me/ocupacoes?de=${encodeURIComponent(de)}&ate=${encodeURIComponent(ate)}`
	);
}

export function criarOcupacao(dados: CriarOcupacaoRequest): Promise<Ocupacao> {
	return apiPost<CriarOcupacaoRequest, Ocupacao>('/providers/me/ocupacoes', dados);
}

export function removerOcupacao(id: string): Promise<void> {
	return apiDelete(`/providers/me/ocupacoes/${encodeURIComponent(id)}`);
}
