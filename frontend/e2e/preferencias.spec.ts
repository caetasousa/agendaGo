import { test, expect } from '@playwright/test';
import { emailUnico } from './helpers';

test('prestador acessa preferências, salva e vê o banner de sucesso', async ({ page }) => {
	await page.goto('/cadastro');
	await page.fill('#nome', 'Preferencias Teste');
	await page.fill('#email', emailUnico('preferencias'));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.click('a:has-text("Preferências")');
	await page.waitForURL('/painel/preferencias');

	await page.check('#aceita-agendamentos');
	await page.fill('#descanso-minutos', '15');
	await page.click('button[type="submit"]');

	await expect(page.getByText('Preferências salvas.')).toBeVisible();
	await expect(page.locator('#aceita-agendamentos')).toBeChecked();
	await expect(page.locator('#descanso-minutos')).toHaveValue('15');
});

test('cliente é redirecionado do painel de preferências para o painel', async ({ page }) => {
	await page.goto('/cadastro');
	await page.click('label:has-text("Cliente")');
	await page.fill('#nome', 'Cliente Preferencias');
	await page.fill('#email', emailUnico('cliente-preferencias'));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.goto('/painel/preferencias');
	await page.waitForURL('/painel');
});
