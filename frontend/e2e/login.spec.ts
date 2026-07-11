import { test, expect } from '@playwright/test';
import { emailUnico } from './helpers';

test('login unificado autentica cliente e senha errada mostra erro', async ({ page }) => {
	const email = emailUnico('login-cliente');

	// cadastra um cliente e sai, para depois logar pela tela de login unificada
	await page.goto('/cadastro');
	await page.click('label:has-text("Cliente")');
	await page.fill('#nome', 'Cliente Login');
	await page.fill('#email', email);
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.click('button:has-text("Sair")');
	await page.waitForURL('/');

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
