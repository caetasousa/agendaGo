import { consultarConvite, type Convite } from '$lib/api/membros';

// Página pública: quem foi convidado ainda não tem conta, e é o aceite que a
// cria. Sem SSR pelo mesmo motivo das demais telas que falam com a API.
export const ssr = false;

export async function load({ url }): Promise<{ token: string; convite: Convite | null; erro: string | null }> {
	const token = url.searchParams.get('token') ?? '';
	if (token === '') {
		return { token, convite: null, erro: 'Link de convite inválido.' };
	}

	try {
		// Consultar não gasta o convite: a pessoa pode abrir o link mais de uma
		// vez antes de preencher o formulário.
		return { token, convite: await consultarConvite(token), erro: null };
	} catch {
		return { token, convite: null, erro: 'Este convite não é mais válido. Peça um novo a quem convidou.' };
	}
}
