import { test, expect } from '@playwright/test';
import { emailUnico } from './helpers';

test('cadastro como prestador leva ao painel com o tipo correto', async ({ page }) => {
	await page.goto('/cadastro');
	await page.fill('#nome', 'Prestador Teste');
	await page.fill('#email', emailUnico('prestador'));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');

	await page.waitForURL('/painel');
	await expect(page.getByText('Olá, Prestador Teste')).toBeVisible();
	await expect(page.getByText('Conta de prestador')).toBeVisible();
});

test('cadastro como cliente leva ao painel com o tipo correto', async ({ page }) => {
	await page.goto('/cadastro');
	await page.click('label:has-text("Cliente")');
	await page.fill('#nome', 'Cliente Teste');
	await page.fill('#email', emailUnico('cliente'));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');

	await page.waitForURL('/painel');
	await expect(page.getByText('Conta de cliente')).toBeVisible();
});
