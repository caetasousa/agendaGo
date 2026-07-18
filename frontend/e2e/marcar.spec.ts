import { test, expect, type Page } from '@playwright/test';
import { emailUnico, tokenDeConfirmacaoCadastro } from './helpers';

// O prestador marca na própria agenda um cliente que o contatou por fora
// (ex.: telefone), pela tela /painel/marcar — nasce CONFIRMADO direto (sem
// pedido para aceitar) e ele cancela a qualquer momento, sem antecedência.

async function cadastrarPrestadorComAgenda(page: Page, ativarAgenda: boolean): Promise<void> {
	await page.goto('/cadastro?tipo=prestador');
	await page.fill('#nome', 'Prestador Marcar');
	await page.fill('#email', emailUnico('marcar'));
	await page.fill('#telefone', '(11) 99999-8888');
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	if (ativarAgenda) {
		await page.goto('/painel/preferencias');
		await page.click('label[for="aceita-agendamentos"]');
		await page.click('button[type="submit"]');
		await expect(page.getByText('Salvo', { exact: true })).toBeVisible();
	}
}

async function marcarPrimeiroSlot(page: Page) {
	await page.goto('/painel/marcar');
	await page.waitForSelector('button[data-dia]:not([disabled])');
	await page.locator('button[data-dia]:not([disabled])').first().click();
	await page.waitForSelector('button[data-slot]');
	await page.locator('button[data-slot]').first().click();
}

test('prestador marca para um cliente só com nome e observação, já CONFIRMADO, e cancela quando quiser', async ({
	page
}) => {
	await cadastrarPrestadorComAgenda(page, true);

	await marcarPrimeiroSlot(page);
	await page.fill('input[autocomplete="name"]', 'Cliente Do Telefone');
	await page.fill('textarea', 'ligou pedindo horário de manhã');
	await page.click('button:has-text("Marcar horário")');
	await expect(page.getByText('Marcação registrada!')).toBeVisible();

	// aparece já CONFIRMADO na lista, sem etapa de aceitar/recusar
	await page.goto('/painel/agendamentos');
	const cartao = page.locator('li[data-agendamento]');
	await expect(cartao).toHaveAttribute('data-status', 'CONFIRMADO');
	await expect(cartao.getByText('Cliente Do Telefone', { exact: true })).toBeVisible();
	await expect(cartao.getByRole('button', { name: 'Confirmar' })).toHaveCount(0);
	await expect(cartao.getByRole('button', { name: 'Recusar' })).toHaveCount(0);

	// a mensagem pronta pro cliente aparece no card, com data/hora/observação
	await expect(cartao.getByText('Agendamento confirmado para')).toContainText(
		'Cliente Do Telefone'
	);
	await expect(cartao.getByText('Agendamento confirmado para')).toContainText(
		'ligou pedindo horário de manhã'
	);

	// e o prestador cancela quando quiser, sem restrição de antecedência
	await page.click('button:has-text("Cancelar")');
	await expect(cartao).toHaveAttribute('data-status', 'CANCELADO');
});

test('prestador marca mesmo com a agenda fechada ao público', async ({ page }) => {
	// agenda nasce desativada e permanece assim — o público não vê horários,
	// mas o dono marca normalmente
	await cadastrarPrestadorComAgenda(page, false);

	await marcarPrimeiroSlot(page);
	await page.fill('input[autocomplete="name"]', 'Cliente Fechado');
	await page.click('button:has-text("Marcar horário")');
	await expect(page.getByText('Marcação registrada!')).toBeVisible();
});

test('cliente não acessa a tela de marcar do prestador', async ({ page, request }) => {
	const email = emailUnico('cliente-marcar');

	await page.goto('/cadastro?tipo=cliente');
	await page.fill('#nome', 'Cliente Marcar');
	await page.fill('#email', email);
	await page.fill('#telefone', '(11) 99999-8888');
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await expect(page.getByText(`Enviamos um email para ${email}`)).toBeVisible();

	const token = await tokenDeConfirmacaoCadastro(request, email);
	await page.goto(`/confirmar-cadastro?token=${token}`);
	await expect(page.getByText('Cadastro confirmado!')).toBeVisible();

	await page.goto('/login');
	await page.fill('#email', email);
	await page.fill('#senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.goto('/painel/marcar');
	await page.waitForURL('/painel');
});
