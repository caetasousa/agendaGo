// Tipos e chamadas da API de clientes.
// Espelham backend/internal/adapter/http/dto/client.go

import { apiPostSemResposta } from './client';

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
