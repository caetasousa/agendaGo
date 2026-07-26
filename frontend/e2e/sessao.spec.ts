import { test, expect } from '@playwright/test';
import { cadastrarPrestadorELogar, emailUnico } from './helpers';

test('acesso anônimo ao painel redireciona para login', async ({ page }) => {
	await page.goto('/painel');
	await page.waitForURL('/login');
});

test('logout pelo header volta para a home e mostra Entrar', async ({ page, request }) => {
	await cadastrarPrestadorELogar(page, request, 'Sessao Teste', emailUnico('sessao'));

	// dentro do painel quem identifica a sessão é a sidebar; o header guarda a saída
	await expect(page.getByRole('complementary').getByText('Sessao Teste')).toBeVisible();

	await page.click('button:has-text("Sair")');
	await page.waitForURL('/');
	await expect(page.locator('header').getByRole('link', { name: 'Entrar' })).toBeVisible();
});
