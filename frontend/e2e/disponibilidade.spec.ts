import { test, expect } from '@playwright/test';
import { emailUnico } from './helpers';

test('prestador define grade semanal, cria e remove uma exceção', async ({ page }) => {
	await page.goto('/cadastro');
	await page.fill('#nome', 'Disponibilidade Teste');
	await page.fill('#email', emailUnico('disponibilidade'));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.click('a:has-text("Disponibilidade")');
	await page.waitForURL('/painel/disponibilidade');

	// adiciona um bloco no primeiro dia (Domingo) e salva
	const primeiroDia = page.locator('div.rounded-md.border').first();
	await primeiroDia.getByText('+ Adicionar bloco').click();
	await page.click('button:has-text("Salvar grade")');

	await expect(page.getByText('Grade semanal salva.')).toBeVisible();

	// recarrega e confirma persistência
	await page.reload({ waitUntil: 'networkidle' });
	const inputsHorario = page.locator('input[type="time"]');
	await expect(inputsHorario.first()).toHaveValue('08:00');

	// cria uma exceção de bloqueio
	await page.fill('#nova-data', '2026-12-25');
	await page.click('button:has-text("Adicionar exceção")');
	await expect(page.getByText('2026-12-25')).toBeVisible();

	// remove a exceção criada
	await page.click('li:has-text("2026-12-25") >> text=Remover');
	await expect(page.getByText('2026-12-25')).not.toBeVisible();
});

test('cliente é redirecionado do painel de disponibilidade para o painel', async ({ page }) => {
	await page.goto('/cadastro');
	await page.click('label:has-text("Cliente")');
	await page.fill('#nome', 'Cliente Disponibilidade');
	await page.fill('#email', emailUnico('cliente-disponibilidade'));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.goto('/painel/disponibilidade');
	await page.waitForURL('/painel');
});
