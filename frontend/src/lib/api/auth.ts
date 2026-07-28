// Tipos e chamadas da API de autenticação.
// Espelham backend/internal/adapter/http/dto/auth.go

import { ApiError, apiGet, apiPost, apiPostVazio, apiPostSemResposta, BASE_URL } from './client';
import type { Bloco } from './availability';

export interface LoginRequest {
	email: string;
	senha: string;
}

export interface LoginResponse {
	id: string;
	nome: string;
	tipo: string;
}

// A agenda que o usuário autenticado opera, e o papel dele nela. Fica num
// objeto próprio para separar o que é da conta (email, telefone) do que é da
// agenda — quando uma pessoa puder operar a agenda de outra, misturar os dois
// no topo deixaria de fazer sentido.
export interface ProviderDoMe {
	id: string;
	papel: string;
	aceitaAgendamentos: boolean;
	descansoMinutos: number;
	duracaoAtendimentoMinutos: number;
	horariosPadrao: Bloco[];
	permiteMarcacaoPeloPrestador: boolean;
}

export interface MeResponse {
	id: string;
	nome: string;
	email: string;
	telefone?: string;
	tipo: string;
	// provider só vem para o tipo provider — ausente para cliente e admin.
	provider?: ProviderDoMe;
	// telefonePendente é true quando o prestador entrou via login social e
	// ainda não confirmou um telefone de verdade — travamos o painel em
	// Preferências até ele completar o cadastro.
	telefonePendente?: boolean;
}

export function loginProvider(dados: LoginRequest): Promise<LoginResponse> {
	return apiPost<LoginRequest, LoginResponse>('/auth/provider/login', dados);
}

export function loginClient(dados: LoginRequest): Promise<LoginResponse> {
	return apiPost<LoginRequest, LoginResponse>('/auth/client/login', dados);
}

export function loginAdmin(dados: LoginRequest): Promise<LoginResponse> {
	return apiPost<LoginRequest, LoginResponse>('/auth/admin/login', dados);
}

// login tenta cada tipo de conta em sequência (prestador → cliente → admin). O
// backend expõe rotas separadas por tipo; esta função abstrai isso do usuário.
// Só o 401 (credenciais não conferem para aquele tipo) faz cair para o próximo:
// um 403 (usuário banido) é credencial válida e propaga o erro de imediato.
export async function login(dados: LoginRequest): Promise<LoginResponse> {
	const tentativas = [loginProvider, loginClient, loginAdmin];
	for (let i = 0; i < tentativas.length; i++) {
		try {
			return await tentativas[i](dados);
		} catch (e) {
			const ehUltima = i === tentativas.length - 1;
			if (!ehUltima && e instanceof ApiError && e.status === 401) {
				continue;
			}
			throw e;
		}
	}
	// inalcançável: o loop sempre retorna ou lança na última tentativa
	throw new Error('login: nenhuma tentativa retornou');
}

export function logout(): Promise<void> {
	return apiPostVazio('/auth/logout');
}

export type TipoContaSocial = 'client' | 'provider';

// urlLoginGoogle monta a URL de início do login social (navegação de
// navegador via redirect, não um fetch — o backend cuida de todo o fluxo
// OAuth e volta com o cookie de sessão já definido). voltar é ecoado pelo
// backend no redirect final, para preservar o destino pós-login (ex.: um
// link público de agendamento).
export function urlLoginGoogle(tipo: TipoContaSocial, voltar?: string): string {
	const query = voltar ? `?voltar=${encodeURIComponent(voltar)}` : '';
	return `${BASE_URL}/auth/${tipo}/google/start${query}`;
}

// urlEntrarComGoogle é a entrada da tela de login: não declara o tipo da
// conta. Quem já tem conta só pode ser cliente ou prestador, e o backend
// descobre qual pelo vínculo social ou pelo email — perguntar seria pedir ao
// usuário uma informação que o sistema já tem. Quem ainda não tem conta volta
// com ?erro=social_sem_conta, porque criar exigiria escolher o tipo.
export function urlEntrarComGoogle(voltar?: string): string {
	const query = voltar ? `?voltar=${encodeURIComponent(voltar)}` : '';
	return `${BASE_URL}/auth/google/start${query}`;
}

export function me(): Promise<MeResponse> {
	return apiGet<MeResponse>('/auth/me');
}

export interface RecuperarSenhaRequest {
	email: string;
}

export interface RedefinirSenhaRequest {
	token: string;
	novaSenha: string;
}

export function recuperarSenha(dados: RecuperarSenhaRequest): Promise<void> {
	return apiPostSemResposta('/auth/recuperar-senha', dados);
}

export function redefinirSenha(dados: RedefinirSenhaRequest): Promise<void> {
	return apiPostSemResposta('/auth/redefinir-senha', dados);
}
