// Tipos e chamadas da API de clientes.
// Espelham backend/internal/adapter/http/dto/client.go

import { apiGet, apiPost, apiPostSemResposta } from './client';

export interface CadastrarClientRequest {
	nome: string;
	email: string;
	telefone: string;
	senha: string;
}

// cadastrarClient inicia o cadastro: o backend envia um email de confirmação e
// responde 204 (sem corpo), exista ou não o email — a conta só é criada quando
// a pessoa confirma pelo link.
export function cadastrarClient(dados: CadastrarClientRequest): Promise<void> {
	return apiPostSemResposta('/clients', dados);
}

// confirmarCadastro conclui o cadastro a partir do token do email.
export function confirmarCadastro(token: string): Promise<void> {
	return apiPostSemResposta('/clients/confirmar-cadastro', { token });
}

export interface PreCadastroResponse {
	nome: string;
	email: string;
	telefone: string;
}

// consultarPreCadastro busca os dados do convidado a partir do token de
// pré-cadastro (uso único), para a tela de cadastro pré-preencher o
// formulário — poupa quem já agendou como convidado de redigitar tudo.
export function consultarPreCadastro(token: string): Promise<PreCadastroResponse> {
	return apiGet<PreCadastroResponse>(`/clients/pre-cadastro/${token}`);
}

export interface ConcluirPreCadastroResponse {
	nome: string;
	email: string;
}

// concluirPreCadastro cria a conta direto a partir do token de pré-cadastro,
// sem uma segunda confirmação por email: quem chegou até aqui pelo link do
// email já provou posse do email. É este endpoint que consome o token
// (uso único) — consultarPreCadastro só lê, para poder ser chamado no load
// da página sem invalidar o link antes do submit.
export function concluirPreCadastro(token: string, senha: string): Promise<ConcluirPreCadastroResponse> {
	return apiPost<{ senha: string }, ConcluirPreCadastroResponse>(`/clients/pre-cadastro/${token}`, {
		senha
	});
}
