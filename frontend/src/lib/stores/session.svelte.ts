// Estado de sessão compartilhado entre header, home e páginas autenticadas.
import { logout, me, type MeResponse } from '$lib/api/auth';

export class Sessao {
	usuario = $state<MeResponse | null>(null);
	carregando = $state(true);

	// carregar consulta /auth/me e trata qualquer falha (401 ou rede) como
	// deslogado — o header não pode quebrar por causa da API fora do ar.
	// Se a store já foi populada (ex: pelo guard do painel), não repete a chamada.
	async carregar(): Promise<void> {
		if (this.usuario) {
			this.carregando = false;
			return;
		}
		try {
			this.usuario = await me();
		} catch {
			this.usuario = null;
		} finally {
			this.carregando = false;
		}
	}

	// definir popula a store com o usuário autenticado (usado pelo guard do painel).
	definir(u: MeResponse): void {
		this.usuario = u;
		this.carregando = false;
	}

	// limpar zera o estado local sem tocar na API.
	limpar(): void {
		this.usuario = null;
		this.carregando = false;
	}

	// sair encerra a sessão na API e limpa o estado local mesmo se a rede falhar.
	async sair(): Promise<void> {
		try {
			await logout();
		} finally {
			this.limpar();
		}
	}
}

export const sessao = new Sessao();
