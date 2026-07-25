import { test, expect } from '@playwright/test';
import { emailUnico, tokenDeConfirmacaoCadastro } from './helpers';

test('cadastro como prestador leva ao painel com o tipo correto', async ({ page }) => {
	await page.goto('/cadastro?tipo=prestador');
	await page.fill('#nome', 'Prestador Teste');
	await page.fill('#email', emailUnico('prestador'));
	await page.fill('#telefone', '(11) 99999-8888');
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');

	await page.waitForURL('/painel');
	// a saudação usa só o primeiro nome; nome completo e tipo de conta ficam na
	// sidebar, para um nome longo não quebrar o título da página
	await expect(page.getByRole('heading', { name: 'Olá, Prestador' })).toBeVisible();
	const sidebar = page.getByRole('complementary');
	await expect(sidebar.getByText('Prestador Teste')).toBeVisible();
	await expect(sidebar.getByText('Prestador', { exact: true })).toBeVisible();
});

test('cadastro como cliente exige confirmação por email antes de logar', async ({ page, request }) => {
	const email = emailUnico('cliente');

	await page.goto('/cadastro?tipo=cliente');
	await page.fill('#nome', 'Cliente Teste');
	await page.fill('#email', email);
	await page.fill('#telefone', '(11) 99999-8888');
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');

	// a conta só nasce quando o link do email de confirmação é aberto
	await expect(page.getByText(`Enviamos um email para ${email}`)).toBeVisible();

	const token = await tokenDeConfirmacaoCadastro(request, email);
	await page.goto(`/confirmar-cadastro?token=${token}`);
	await expect(page.getByText('Cadastro confirmado!')).toBeVisible();

	await page.goto('/login');
	await page.fill('#email', email);
	await page.fill('#senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
	// o tipo da conta é exibido no bloco de identidade da sidebar
	await expect(page.getByRole('complementary').getByText('Cliente', { exact: true })).toBeVisible();
});

test('cadastro sem tipo mostra a escolha explícita antes do formulário', async ({ page }) => {
	await page.goto('/cadastro');

	// os dois caminhos aparecem e nenhum campo do formulário é exibido ainda —
	// ninguém se cadastra num tipo sem ter escolhido
	await expect(page.locator('[data-escolher="cliente"]')).toBeVisible();
	await expect(page.locator('[data-escolher="prestador"]')).toBeVisible();
	await expect(page.locator('#nome')).toHaveCount(0);

	// escolher "cliente" revela o formulário do tipo certo — o tipo escolhido
	// fica marcado no segmento, que também permite trocar sem voltar
	await page.click('[data-escolher="cliente"]');
	await expect(page.locator('#nome')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Quero agendar', pressed: true })).toBeVisible();
});
