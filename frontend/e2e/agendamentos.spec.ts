import { test, expect, type Page, type APIRequestContext } from '@playwright/test';
import { emailUnico, tokenDeConfirmacaoCadastro } from './helpers';

// O fluxo de agendamento gira em torno do link público do prestador
// (/agendar/{id}): convidados veem os horários livres sem cadastro e o login
// só é exigido na hora de solicitar.

async function cadastrarPrestadorAtivo(page: Page, nome: string, email: string): Promise<string> {
	await page.goto('/cadastro?tipo=prestador');
	await page.fill('#nome', nome);
	await page.fill('#email', email);
	await page.fill('#telefone', '(11) 99999-8888');
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.goto('/painel/preferencias');
	await page.click('label[for="aceita-agendamentos"]');
	await page.click('button[type="submit"]');
	await expect(page.getByText('Salvo', { exact: true })).toBeVisible();

	// o painel do prestador exibe o link público de agendamento
	await page.goto('/painel');
	const link = await page.locator('[data-link-agendamento]').textContent();

	await page.click('button:has-text("Sair")');
	await page.waitForURL('/');
	return new URL(link!.trim()).pathname;
}

// cadastrarCliente cria a conta e já confirma pelo link do email (via
// Mailpit), já que a conta de cliente só nasce após a confirmação — mas não
// loga: quem precisa de sessão chama entrar() em seguida.
async function cadastrarCliente(page: Page, request: APIRequestContext, nome: string, email: string) {
	await page.goto('/cadastro?tipo=cliente');
	await page.fill('#nome', nome);
	await page.fill('#email', email);
	await page.fill('#telefone', '(11) 99999-8888');
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await expect(page.getByText(`Enviamos um email para ${email}`)).toBeVisible();

	const token = await tokenDeConfirmacaoCadastro(request, email);
	await page.goto(`/confirmar-cadastro?token=${token}`);
	await expect(page.getByText('Cadastro confirmado!')).toBeVisible();
}

async function entrar(page: Page, email: string) {
	await page.goto('/login');
	await page.fill('#email', email);
	await page.fill('#senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
}

// escolhe o primeiro dia com horários livres e o primeiro slot do dia
async function escolherPrimeiroSlot(page: Page) {
	await page.waitForSelector('button[data-dia]:not([disabled])');
	await page.locator('button[data-dia]:not([disabled])').first().click();
	await page.waitForSelector('button[data-slot]');
	await page.locator('button[data-slot]').first().click();
}

// escolhe um dia bem à frente (próximo mês) e o primeiro slot dele — usado
// quando o teste depende de antecedência mínima (24h) para cancelar, já que
// o primeiro dia disponível pode cair perto demais de "agora" (ex.: domingo
// de manhã, cujo próximo horário livre é segunda 08:00, a menos de 24h).
async function escolherSlotDoProximoMes(page: Page) {
	await page.waitForSelector('button[aria-label="Próximo mês"]');
	await page.click('button[aria-label="Próximo mês"]');
	await escolherPrimeiroSlot(page);
}

test('convidado vê os horários pelo link público e pode entrar para agendar', async ({ page, request }) => {
	const linkPublico = await cadastrarPrestadorAtivo(
		page,
		`Prestador Link ${Date.now()}`,
		emailUnico('link-prestador')
	);
	const emailCliente = emailUnico('link-cliente');
	await cadastrarCliente(page, request, 'Cliente Convidado', emailCliente);

	// convidado (sem sessão) abre o link e enxerga o calendário com horários
	await page.goto(linkPublico);
	await escolherPrimeiroSlot(page);
	// o formulário de convidado aparece, com o atalho "Já tem conta? Entrar"
	await expect(page.getByText('Agende sem criar conta')).toBeVisible();

	// login pelo atalho volta para a página do prestador
	await page.getByRole('button', { name: 'Entrar' }).click();
	await page.waitForURL(`**/login?voltar=**`);
	await page.fill('#email', emailCliente);
	await page.fill('#senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL(`**${linkPublico}`);

	// agora logado como cliente, solicita o horário
	await escolherPrimeiroSlot(page);
	await page.click('button:has-text("Solicitar agendamento")');
	await expect(page.getByText('Solicitação enviada!')).toBeVisible();
});

test('convidado agenda sem cadastro informando nome/email/telefone e o prestador vê o contato', async ({
	page
}) => {
	const emailPrestador = emailUnico('convidado-prestador');
	const linkPublico = await cadastrarPrestadorAtivo(
		page,
		`Prestador Convidado ${Date.now()}`,
		emailPrestador
	);

	// visitante sem sessão preenche o formulário de convidado
	const emailConvidado = emailUnico('convidado');
	await page.goto(linkPublico);
	await escolherPrimeiroSlot(page);
	await page.fill('input[autocomplete="name"]', 'Convidada Silva');
	await page.fill('input[autocomplete="email"]', emailConvidado);
	await page.fill('input[autocomplete="tel"]', '(11) 99999-8888');
	await page.click('button:has-text("Solicitar agendamento")');
	await expect(page.getByText('Solicitação enviada!')).toBeVisible();

	// o prestador enxerga o nome e o telefone do convidado no agendamento
	await entrar(page, emailPrestador);
	await page.goto('/painel/agendamentos');
	const cartao = page.locator('li[data-agendamento]');
	await expect(cartao).toHaveAttribute('data-status', 'SOLICITADO');
	await expect(cartao.getByText('Convidada Silva')).toBeVisible();
	await expect(cartao.getByText('(11) 99999-8888')).toBeVisible();
});

test('convidado com e-mail de conta registrada é orientado a entrar', async ({ page, request }) => {
	const linkPublico = await cadastrarPrestadorAtivo(
		page,
		`Prestador Conta ${Date.now()}`,
		emailUnico('conta-prestador')
	);
	// cria uma conta de cliente — o e-mail dela será usado no formulário
	const emailComConta = emailUnico('conta-cliente');
	await cadastrarCliente(page, request, 'Cliente Com Conta', emailComConta);

	await page.goto(linkPublico);
	await escolherPrimeiroSlot(page);
	await page.fill('input[autocomplete="name"]', 'Impostora');
	await page.fill('input[autocomplete="email"]', emailComConta);
	await page.fill('input[autocomplete="tel"]', '(11) 99999-8888');
	await page.click('button:has-text("Solicitar agendamento")');

	// o backend rejeita (409) e a página mostra a orientação
	await expect(page.getByText('este e-mail já tem conta; entre para agendar')).toBeVisible();
});

test('link público rejeita telefone curto do convidado (validação leve)', async ({ page }) => {
	const linkPublico = await cadastrarPrestadorAtivo(
		page,
		`Prestador Tel ${Date.now()}`,
		emailUnico('tel-prestador')
	);

	await page.goto(linkPublico);
	await escolherPrimeiroSlot(page);
	await page.fill('input[autocomplete="name"]', 'Ana');
	await page.fill('input[autocomplete="email"]', emailUnico('tel-convidado'));
	await page.fill('input[autocomplete="tel"]', '123');
	// telefone com menos de 8 dígitos: o botão de solicitar fica desabilitado
	await expect(page.getByRole('button', { name: 'Solicitar agendamento' })).toBeDisabled();
});

test('fluxo completo: cliente agenda pelo calendário, prestador confirma, cliente cancela', async ({
	page,
	request
}) => {
	const emailPrestador = emailUnico('fluxo-prestador');
	const linkPublico = await cadastrarPrestadorAtivo(page, `Prestador Fluxo ${Date.now()}`, emailPrestador);
	const emailCliente = emailUnico('fluxo-cliente');
	await cadastrarCliente(page, request, 'Cliente Fluxo', emailCliente);
	await entrar(page, emailCliente);

	// cliente logado agenda direto pelo link, num dia bem à frente (a etapa de
	// cancelamento abaixo exige 24h de antecedência do horário confirmado)
	await page.goto(linkPublico);
	await escolherSlotDoProximoMes(page);
	const totalSlotsAntes = await page.locator('button[data-slot]').count();
	await page.click('button:has-text("Solicitar agendamento")');
	await expect(page.getByText('Solicitação enviada!')).toBeVisible();

	// slot reservado sai da oferta
	await expect(page.locator('button[data-slot]')).toHaveCount(totalSlotsAntes - 1);

	// prestador confirma
	await page.click('button:has-text("Sair")');
	await entrar(page, emailPrestador);
	await page.goto('/painel/agendamentos');
	const cartao = page.locator('li[data-agendamento]');
	await expect(cartao).toHaveAttribute('data-status', 'SOLICITADO');
	await expect(cartao.getByText('Cliente Fluxo')).toBeVisible();
	await page.click('button:has-text("Confirmar")');
	await expect(cartao).toHaveAttribute('data-status', 'CONFIRMADO');

	// cliente vê confirmado e cancela (horário futuro, antecedência ok)
	await page.click('button:has-text("Sair")');
	await entrar(page, emailCliente);
	await page.goto('/painel/agendamentos');
	await expect(page.locator('li[data-agendamento]')).toHaveAttribute('data-status', 'CONFIRMADO');
	await page.click('button:has-text("Cancelar")');
	await expect(page.locator('li[data-agendamento]')).toHaveAttribute('data-status', 'CANCELADO');
});

test('prestador recusa uma solicitação e o cliente vê o status', async ({ page, request }) => {
	const emailPrestador = emailUnico('recusa-prestador');
	const linkPublico = await cadastrarPrestadorAtivo(page, `Prestador Recusa ${Date.now()}`, emailPrestador);
	const emailCliente = emailUnico('recusa-cliente');
	await cadastrarCliente(page, request, 'Cliente Recusa', emailCliente);
	await entrar(page, emailCliente);

	await page.goto(linkPublico);
	await escolherPrimeiroSlot(page);
	await page.click('button:has-text("Solicitar agendamento")');
	await expect(page.getByText('Solicitação enviada!')).toBeVisible();

	await page.click('button:has-text("Sair")');
	await entrar(page, emailPrestador);
	await page.goto('/painel/agendamentos');
	await page.click('button:has-text("Cancelar")');
	await expect(page.locator('li[data-agendamento]')).toHaveAttribute('data-status', 'RECUSADO');

	await page.click('button:has-text("Sair")');
	await entrar(page, emailCliente);
	await page.goto('/painel/agendamentos');
	const cartaoCliente = page.locator('li[data-agendamento]');
	await expect(cartaoCliente).toHaveAttribute('data-status', 'RECUSADO');
	await expect(cartaoCliente.getByText('Recusado', { exact: true })).toBeVisible();
});

test('diretório do painel lista todos os prestadores com link para o calendário', async ({ page, request }) => {
	const nomePrestador = `Prestador Diretorio ${Date.now()}`;
	await cadastrarPrestadorAtivo(page, nomePrestador, emailUnico('diretorio-prestador'));
	const emailCliente = emailUnico('diretorio-cliente');
	await cadastrarCliente(page, request, 'Cliente Diretorio', emailCliente);
	await entrar(page, emailCliente);

	await page.goto('/painel/agendar');
	const cardPrestador = page.locator(`a:has-text("${nomePrestador}")`);
	await expect(cardPrestador).toBeVisible();
	await cardPrestador.click();
	await page.waitForURL('**/agendar/**');
	await expect(page.getByText('Escolha o dia')).toBeVisible();
});
