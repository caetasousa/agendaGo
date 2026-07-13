import { test, expect } from '@playwright/test';
import { emailUnico, tokenDeConfirmacaoCadastro } from './helpers';

test('login unificado autentica cliente e senha errada mostra erro', async ({ page, request }) => {
	const email = emailUnico('login-cliente');

	// cadastra um cliente e confirma pelo link do email, para depois logar
	// pela tela de login unificada
	await page.goto('/cadastro');
	await page.click('label:has-text("Cliente")');
	await page.fill('#nome', 'Cliente Login');
	await page.fill('#email', email);
	await page.fill('#telefone', '(11) 99999-8888');
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await expect(page.getByText(`Enviamos um email para ${email}`)).toBeVisible();

	const token = await tokenDeConfirmacaoCadastro(request, email);
	await page.goto(`/confirmar-cadastro?token=${token}`);
	await expect(page.getByText('Cadastro confirmado!')).toBeVisible();

	// senha errada mostra erro genérico
	await page.goto('/login');
	await page.fill('#email', email);
	await page.fill('#senha', 'senha-errada');
	await page.click('button[type="submit"]');
	await expect(page.getByText('credenciais inválidas')).toBeVisible();

	// login correto pela tela unificada (tenta provider, cai para client)
	await page.fill('#senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
	await expect(page.getByText('Conta de cliente')).toBeVisible();
});
