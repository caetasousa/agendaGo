import { expect } from '@playwright/test';

// Gera um email único por execução para isolar dados entre specs e runs,
// sem depender de reset de banco entre testes.
export function emailUnico(prefixo: string): string {
	const sufixo = Math.random().toString(36).slice(2, 8);
	return `${prefixo}-${Date.now()}-${sufixo}@email.com`;
}

// URL da API do Mailpit, o catcher de SMTP do docker-compose. Os testes que
// exercem email leem as mensagens capturadas aqui — logo, dependem de o SMTP
// da API estar apontando para o Mailpit (padrão do compose), não para um
// provedor real.
const MAILPIT_API = 'http://localhost:8025/api/v1';

// buscaTokenNoMailpit faz polling no Mailpit até achar, entre os emails
// enviados a destinatario, um cujo assunto contenha assuntoContem, e extrai o
// token do primeiro link "?token=..." do corpo HTML. Compartilhado entre a
// recuperação de senha e a confirmação de cadastro, que seguem o mesmo padrão
// de link — o envio do email é assíncrono no backend, daí o polling.
async function buscaTokenNoMailpit(
	request: import('@playwright/test').APIRequestContext,
	destinatario: string,
	assuntoContem: string
): Promise<string> {
	for (let tentativa = 0; tentativa < 20; tentativa++) {
		const resposta = await request.get(
			`${MAILPIT_API}/search?query=${encodeURIComponent('to:' + destinatario)}`
		);
		if (resposta.ok()) {
			const corpo = await resposta.json();
			const mensagem = (corpo.messages ?? []).find((m: { Subject: string }) =>
				m.Subject.includes(assuntoContem)
			);
			if (mensagem) {
				const detalhe = await request.get(`${MAILPIT_API}/message/${mensagem.ID}`);
				const html = (await detalhe.json()).HTML as string;
				const encontrado = html.match(/token=([^"&]+)/);
				if (encontrado) return encontrado[1];
			}
		}
		await new Promise((r) => setTimeout(r, 250));
	}
	throw new Error(`email "${assuntoContem}" para ${destinatario} não chegou no Mailpit`);
}

// tokenDeRecuperacao espera o email de recuperação de senha de destinatario
// chegar no Mailpit e extrai o token do link `/redefinir-senha?token=...`.
export async function tokenDeRecuperacao(
	request: import('@playwright/test').APIRequestContext,
	destinatario: string
): Promise<string> {
	return buscaTokenNoMailpit(request, destinatario, 'Redefinição de senha');
}

// tokenDeConfirmacaoCadastro espera o email de confirmação de cadastro de
// destinatario chegar no Mailpit e extrai o token do link
// `/confirmar-cadastro?token=...`.
export async function tokenDeConfirmacaoCadastro(
	request: import('@playwright/test').APIRequestContext,
	destinatario: string
): Promise<string> {
	return buscaTokenNoMailpit(request, destinatario, 'Confirme seu cadastro');
}

// SENHA_PADRAO é a senha usada por todas as contas criadas nos testes.
export const SENHA_PADRAO = '12345678';

// cadastrarPrestador cria a conta de prestador ponta a ponta: preenche o
// formulário, pega o link no Mailpit e confirma. A conta só existe depois da
// confirmação — o cadastro não loga mais direto, para o email ser verificado
// antes de o prestador aparecer na vitrine.
export async function cadastrarPrestador(
	page: import('@playwright/test').Page,
	request: import('@playwright/test').APIRequestContext,
	nome: string,
	email: string
): Promise<void> {
	await page.goto('/cadastro?tipo=prestador');
	await page.fill('#nome', nome);
	await page.fill('#email', email);
	await page.fill('#telefone', '(11) 99999-8888');
	await page.fill('#senha', SENHA_PADRAO);
	await page.fill('#confirmar-senha', SENHA_PADRAO);
	await page.click('button[type="submit"]');
	await expect(page.getByText(`Enviamos um email para ${email}`)).toBeVisible();

	const token = await tokenDeConfirmacaoCadastro(request, email);
	await page.goto(`/confirmar-cadastro?token=${token}&tipo=prestador`);
	await expect(page.getByText('Cadastro confirmado!')).toBeVisible();
}

// entrar faz o login pelo formulário e espera cair no painel.
export async function entrar(
	page: import('@playwright/test').Page,
	email: string,
	senha = SENHA_PADRAO
): Promise<void> {
	await page.goto('/login');
	await page.fill('#email', email);
	await page.fill('#senha', senha);
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
}

// cadastrarPrestadorELogar encadeia o cadastro confirmado e o login — o ponto
// de partida da maioria dos testes de painel.
export async function cadastrarPrestadorELogar(
	page: import('@playwright/test').Page,
	request: import('@playwright/test').APIRequestContext,
	nome: string,
	email: string
): Promise<void> {
	await cadastrarPrestador(page, request, nome, email);
	await entrar(page, email);
}

// tokenDeConvite espera o email de convite de membro chegar no Mailpit e
// extrai o token do link `/convite?token=...`.
export async function tokenDeConvite(
	request: import('@playwright/test').APIRequestContext,
	destinatario: string
): Promise<string> {
	return buscaTokenNoMailpit(request, destinatario, 'Convite para operar uma agenda');
}
