import { test, expect, type Page, type APIRequestContext } from '@playwright/test';
import { emailUnico, cadastrarPrestador, tokenDeConvite, SENHA_PADRAO } from './helpers';

// A Fase 2 separou a conta da agenda para que uma segunda pessoa pudesse
// operá-la sem a senha do dono. Este spec exerce o caminho inteiro dessa
// promessa: convidar, aceitar, entrar e operar de fato.

async function entrar(page: Page, email: string) {
	await page.goto('/login');
	await page.fill('#email', email);
	await page.fill('#senha', SENHA_PADRAO);
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
}

async function sair(page: Page) {
	await page.locator('header').getByText('Sair').click();
	await page.waitForURL('/');
}

// prepararDona cria a prestadora e a deixa logada no painel de equipe.
async function prepararDona(page: Page, request: APIRequestContext): Promise<string> {
	const email = emailUnico('dona');
	await cadastrarPrestador(page, request, `Agenda da Dona ${Date.now()}`, email);
	await entrar(page, email);
	await page.goto('/painel/equipe');
	return email;
}

test('dona convida, operadora cria o acesso e passa a operar a agenda', async ({ page, request }) => {
	const emailDona = await prepararDona(page, request);
	const emailOperadora = emailUnico('operadora');

	// A dona aparece na equipe, marcada como dona e sem opção de se remover.
	await expect(page.getByText(emailDona)).toBeVisible();
	await expect(page.getByText('Dona da agenda')).toBeVisible();

	await page.fill('#email-convite', emailOperadora);
	await page.click('button:has-text("Enviar convite")');
	await expect(page.getByText(`Convite enviado para ${emailOperadora}`)).toBeVisible();
	await expect(page.locator(`[data-convite="${emailOperadora}"]`)).toBeVisible();

	// A operadora recebe o link e cria o próprio acesso — sem nunca saber a
	// senha da dona.
	const token = await tokenDeConvite(request, emailOperadora);
	await sair(page);
	await page.goto(`/convite?token=${token}`);
	await expect(page.getByText(emailOperadora)).toBeVisible();

	await page.fill('#telefone', '(11) 98888-7777');
	await page.fill('#senha', SENHA_PADRAO);
	await page.fill('#confirmar-senha', SENHA_PADRAO);
	await page.click('button:has-text("Criar meu acesso")');
	await expect(page.getByText('Acesso criado')).toBeVisible();

	// O ponto do desenho: ela entra e cai na agenda de QUEM A CONVIDOU, não
	// numa agenda própria — o convite não cria agenda nenhuma.
	await entrar(page, emailOperadora);
	await page.goto('/painel/equipe');
	await expect(page.getByText(emailDona)).toBeVisible();
	await expect(page.getByText(emailOperadora)).toBeVisible();

	// E opera de fato: preferências é uma rota de gestão da agenda. A barra de
	// salvar só aparece com alteração pendente, daí o toggle antes do submit.
	await page.goto('/painel/preferencias');
	await page.click('label[for="aceita-agendamentos"]');
	await page.click('button[type="submit"]');
	await expect(page.getByText('Salvo', { exact: true })).toBeVisible();
});

test('operadora não administra a equipe — só o dono convida e remove', async ({ page, request }) => {
	await prepararDona(page, request);
	const emailOperadora = emailUnico('sem-poder');

	await page.fill('#email-convite', emailOperadora);
	await page.click('button:has-text("Enviar convite")');
	await expect(page.getByText(`Convite enviado para ${emailOperadora}`)).toBeVisible();

	const token = await tokenDeConvite(request, emailOperadora);
	await sair(page);
	await page.goto(`/convite?token=${token}`);
	await page.fill('#telefone', '(11) 98888-7777');
	await page.fill('#senha', SENHA_PADRAO);
	await page.fill('#confirmar-senha', SENHA_PADRAO);
	await page.click('button:has-text("Criar meu acesso")');
	await expect(page.getByText('Acesso criado')).toBeVisible();

	await entrar(page, emailOperadora);
	await page.goto('/painel/equipe');

	// Ela vê quem tem acesso, mas não recebe as ferramentas de administrá-lo.
	await expect(page.getByText('Com acesso')).toBeVisible();
	await expect(page.locator('#email-convite')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Remover acesso' })).toHaveCount(0);
});

test('convite para email que já tem conta é recusado', async ({ page, request }) => {
	await prepararDona(page, request);

	// Um prestador já cadastrado: ele é dono da própria agenda, e um segundo
	// vínculo o deixaria caindo na agenda errada.
	const emailOutroPrestador = emailUnico('ja-prestador');
	const paginaAuxiliar = await page.context().newPage();
	await cadastrarPrestador(paginaAuxiliar, request, 'Outro Prestador', emailOutroPrestador);
	await paginaAuxiliar.close();

	await page.fill('#email-convite', emailOutroPrestador);
	await page.click('button:has-text("Enviar convite")');

	// A mensagem não diz QUE TIPO de conta existe — o convite não pode virar
	// uma sonda de emails cadastrados.
	await expect(page.getByText(/não foi possível convidar este email/i)).toBeVisible();
	await expect(page.locator(`[data-convite="${emailOutroPrestador}"]`)).toHaveCount(0);
});

test('link de convite inválido explica o que fazer', async ({ page }) => {
	await page.goto('/convite?token=token-que-nao-existe');
	await expect(page.getByText('Convite inválido')).toBeVisible();
	await expect(page.getByText(/peça um novo a quem convidou/i)).toBeVisible();
});
