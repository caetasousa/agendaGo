// Tipos e chamadas da API de prestadores.
// Espelham backend/internal/adapter/http/dto/provider.go

import { apiGet, apiPostSemResposta } from './client';
import { queryDaPagina, type Pagina, type Paginacao } from './paginacao';

export interface CadastrarProviderRequest {
	nome: string;
	email: string;
	telefone: string;
	senha: string;
}

// cadastrarProvider solicita o cadastro: a API envia um email de confirmação e
// responde sempre igual, exista ou não o endereço. A conta só nasce quando a
// pessoa clica no link — por isso não há resposta com o id.
export function cadastrarProvider(dados: CadastrarProviderRequest): Promise<void> {
	return apiPostSemResposta('/providers', dados);
}

// confirmarCadastroPrestador conclui o cadastro a partir do token do email e
// cria a conta do prestador.
export function confirmarCadastroPrestador(token: string): Promise<void> {
	return apiPostSemResposta('/providers/confirmar-cadastro', { token });
}

export interface PrestadorResumo {
	id: string;
	// Endereço público legível. O link por id continua funcionando — ver
	// BuscarResumoUseCase no backend —, mas o que se compartilha é o slug.
	slug: string;
	nome: string;
	duracaoAtendimentoMinutos: number;
	aceitaAgendamentos: boolean;
}

export interface ListarPrestadoresResponse extends Paginacao {
	prestadores: PrestadorResumo[];
}

// listarPrestadores devolve uma página da vitrine — rota pública. Quem está
// com a agenda desativada aparece sem horários.
export function listarPrestadores(pagina?: Pagina): Promise<ListarPrestadoresResponse> {
	return apiGet<ListarPrestadoresResponse>(`/providers${queryDaPagina(pagina)}`);
}

// buscarPrestador devolve a identificação pública de um prestador — usada
// pela página de agendamento acessada via link direto.
export function buscarPrestador(id: string): Promise<PrestadorResumo> {
	return apiGet<PrestadorResumo>(`/providers/${id}`);
}
