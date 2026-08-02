// Tipos e chamadas dos direitos do titular sobre os próprios dados.
// Espelham backend/internal/adapter/http/dto/lgpd.go

import { apiDelete, apiGet } from './client';

export interface AgendamentoExportado {
	data: string;
	inicioMinutos: number;
	fimMinutos: number;
	status: string;
	criadoEm: string;
}

export interface DadosExportados {
	id: string;
	nome: string;
	email: string;
	telefone?: string;
	criadoEm: string;
	agendamentos: AgendamentoExportado[];
	// total é quantos agendamentos existem ao todo; truncado avisa que a lista
	// não os traz todos.
	total: number;
	truncado: boolean;
}

export function exportarMeusDados(): Promise<DadosExportados> {
	return apiGet<DadosExportados>('/clients/me/dados');
}

// removerMinhaConta anonimiza o cadastro. Os agendamentos permanecem na agenda
// do prestador, sem identificar quem era.
export function removerMinhaConta(): Promise<void> {
	return apiDelete('/clients/me');
}

// baixarComoJson dispara o download de um objeto como arquivo.
//
// Feito no cliente a partir do JSON que a API devolveu, em vez de seguir o
// Content-Disposition da resposta: a chamada precisa do cookie de sessão, e um
// link ou window.open não carregaria o mesmo estado de autenticação de forma
// confiável entre navegadores.
//
// O revokeObjectURL é obrigatório — sem ele o Blob fica retido enquanto a aba
// viver, e este é um arquivo que pode ter centenas de agendamentos.
export function baixarComoJson(dados: unknown, nomeArquivo: string) {
	const blob = new Blob([JSON.stringify(dados, null, 2)], { type: 'application/json' });
	const url = URL.createObjectURL(blob);
	const link = document.createElement('a');
	link.href = url;
	link.download = nomeArquivo;
	link.click();
	URL.revokeObjectURL(url);
}
