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
