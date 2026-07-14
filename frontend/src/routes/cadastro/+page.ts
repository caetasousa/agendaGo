// O form só deve ser interativo depois que o Svelte hidratar os listeners
// (onsubmit). Com SSR, existe uma janela real em que o HTML chega do
// servidor mas o JS ainda não anexou os handlers, e um clique no botão
// dispara o submit nativo do form (GET com os campos na querystring).
export const ssr = false;

import { ApiError } from '$lib/api/client';
import { consultarPreCadastro, type PreCadastroResponse } from '$lib/api/customer';

// Quem chega com ?pre=TOKEN veio do link "Criar minha conta" da página de
// cancelamento: o token (uso único) já prova posse do email, então busca os
// dados do convidado para pré-preencher o formulário. Token ausente, inválido
// ou já consumido é tratado como cadastro normal — nunca bloqueia a página.
export async function load({ url }): Promise<{ preCadastro: PreCadastroResponse | null }> {
	const token = url.searchParams.get('pre');
	if (!token) return { preCadastro: null };

	try {
		return { preCadastro: await consultarPreCadastro(token) };
	} catch (e) {
		void e;
		return { preCadastro: null };
	}
}
