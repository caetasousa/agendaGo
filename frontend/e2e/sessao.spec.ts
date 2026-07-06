import { test, expect } from '@playwright/test';
import { emailUnico } from './helpers';

test('acesso anônimo ao painel redireciona para login', async ({ page }) => {
	await page.goto('/painel');
	await page.waitForURL('/login');
});

test('logout pelo header volta para a home e mostra Entrar', async ({ page }) => {
	await page.goto('/cadastro');
	await page.fill('#nome', 'Sessao Teste');
	await page.fill('#email', emailUnico('sessao'));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	// header mostra o usuário logado e o link para o painel
	await expect(page.locator('header').getByText('Sessao Teste')).toBeVisible();

	await page.click('button:has-text("Sair")');
	await page.waitForURL('/');
	await expect(page.locator('header').getByRole('link', { name: 'Entrar' })).toBeVisible();
});
