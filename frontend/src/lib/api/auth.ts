// Tipos e chamadas da API de autenticação.
// Espelham backend/internal/adapter/http/dto/auth.go

import { ApiError, apiGet, apiPost, apiPostVazio } from './client';
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

export interface MeResponse {
	id: string;
	nome: string;
	email: string;
	tipo: string;
	aceitaAgendamentos?: boolean;
	descansoMinutos?: number;
	horariosPadrao?: Bloco[];
}

export function loginProvider(dados: LoginRequest): Promise<LoginResponse> {
	return apiPost<LoginRequest, LoginResponse>('/auth/provider/login', dados);
}

export function loginClient(dados: LoginRequest): Promise<LoginResponse> {
	return apiPost<LoginRequest, LoginResponse>('/auth/client/login', dados);
}

// login tenta autenticar como prestador; se as credenciais não corresponderem
// a um prestador (401), tenta como cliente. O backend expõe rotas de login
// separadas por tipo de usuário — esta função abstrai essa escolha do usuário.
export async function login(dados: LoginRequest): Promise<LoginResponse> {
	try {
		return await loginProvider(dados);
	} catch (e) {
		if (e instanceof ApiError && e.status === 401) {
			return loginClient(dados);
		}
		throw e;
	}
}

export function logout(): Promise<void> {
	return apiPostVazio('/auth/logout');
}

export function me(): Promise<MeResponse> {
	return apiGet<MeResponse>('/auth/me');
}
