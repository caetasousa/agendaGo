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
	await expect(page.getByText('Olá, Prestador Teste')).toBeVisible();
	await expect(page.getByText('Conta de prestador')).toBeVisible();
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
	await expect(page.getByText('Conta de cliente')).toBeVisible();
});

test('cadastro sem tipo mostra a escolha explícita antes do formulário', async ({ page }) => {
	await page.goto('/cadastro');

	// os dois caminhos aparecem e nenhum campo do formulário é exibido ainda —
	// ninguém se cadastra num tipo sem ter escolhido
	await expect(page.locator('[data-escolher="cliente"]')).toBeVisible();
	await expect(page.locator('[data-escolher="prestador"]')).toBeVisible();
	await expect(page.locator('#nome')).toHaveCount(0);

	// escolher "cliente" revela o formulário do tipo certo
	await page.click('[data-escolher="cliente"]');
	await expect(page.locator('#nome')).toBeVisible();
	await expect(page.getByText('Conta de cliente')).toBeVisible();
});
