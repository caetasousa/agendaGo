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

	await page.click('label[for="aceita-agendamentos"]');
	await page.fill('#descanso-minutos', '15');
	await page.click('button[type="submit"]');

	await expect(page.getByText('Salvo', { exact: true })).toBeVisible();
	await expect(page.locator('#aceita-agendamentos')).toBeChecked();
	await expect(page.locator('#descanso-minutos')).toHaveValue('15');
});

test('prestador começa com o expediente comercial sugerido e pode editá-lo', async ({ page }) => {
	await page.goto('/cadastro');
	await page.fill('#nome', 'Expediente Teste');
	await page.fill('#email', emailUnico('expediente'));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.goto('/painel/preferencias');
	const seletoresHorario = page.locator('select');
	await expect(seletoresHorario).toHaveCount(4);
	await expect(seletoresHorario.first()).toHaveValue('08:00');

	// remove o bloco da tarde, deixando só a manhã
	await page.locator('button:has-text("Remover")').last().click();
	await expect(seletoresHorario).toHaveCount(2);

	await page.click('button[type="submit"]');
	await expect(page.getByText('Salvo', { exact: true })).toBeVisible();

	await page.reload({ waitUntil: 'networkidle' });
	await expect(page.locator('select')).toHaveCount(2);
});

test('prestador define três períodos curtos no expediente padrão', async ({ page }) => {
	await page.goto('/cadastro');
	await page.fill('#nome', 'Tres Periodos Teste');
	await page.fill('#email', emailUnico('tres-periodos'));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.goto('/painel/preferencias');

	// remove os dois blocos sugeridos e monta três períodos curtos do zero
	await page.locator('button:has-text("Remover")').last().click();
	await page.locator('button:has-text("Remover")').last().click();
	await expect(page.locator('select')).toHaveCount(0);

	for (const [inicio, fim] of [
		['08:00', '10:00'],
		['11:00', '13:00'],
		['15:00', '17:00']
	]) {
		await page.click('button:has-text("+ Adicionar período")');
		const linhas = page.locator('select');
		const total = await linhas.count();
		await linhas.nth(total - 2).selectOption(inicio);
		await linhas.nth(total - 1).selectOption(fim);
	}

	await page.click('button[type="submit"]');
	await expect(page.getByText('Salvo', { exact: true })).toBeVisible();
	await expect(page.locator('select')).toHaveCount(6);

	await page.reload({ waitUntil: 'networkidle' });
	const seletoresAposReload = page.locator('select');
	await expect(seletoresAposReload).toHaveCount(6);
	await expect(seletoresAposReload.nth(0)).toHaveValue('08:00');
	await expect(seletoresAposReload.nth(2)).toHaveValue('11:00');
	await expect(seletoresAposReload.nth(4)).toHaveValue('15:00');
});

test('expediente padrão configurado aparece no calendário de disponibilidade', async ({ page }) => {
	await page.goto('/cadastro');
	await page.fill('#nome', 'Padrao Reflete Calendario');
	await page.fill('#email', emailUnico('padrao-calendario'));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.goto('/painel/preferencias');
	await page.click('label[for="aceita-agendamentos"]');
	// mantém só a manhã, editada para 09:00-11:00
	await page.locator('button:has-text("Remover")').last().click();
	const seletoresHorario = page.locator('select');
	await seletoresHorario.first().selectOption('09:00');
	await seletoresHorario.last().selectOption('11:00');
	await page.click('button[type="submit"]');
	await expect(page.getByText('Salvo', { exact: true })).toBeVisible();

	await page.goto('/painel/disponibilidade');
	await page.waitForSelector('button[data-data]');

	// procura, no mês atual, o primeiro dia útil futuro sem definição própria
	// (a asserção da regra de 24h já é coberta em disponibilidade.spec.ts —
	// aqui o interesse é só confirmar que o expediente configurado reflete)
	const chave = await page.evaluate(() => {
		const hoje = new Date();
		for (let i = 1; i <= 10; i++) {
			const data = new Date(hoje.getFullYear(), hoje.getMonth(), hoje.getDate() + i);
			const diaSemana = data.getDay();
			if (diaSemana === 0 || diaSemana === 6) continue;
			return `${data.getFullYear()}-${String(data.getMonth() + 1).padStart(2, '0')}-${String(data.getDate()).padStart(2, '0')}`;
		}
		throw new Error('nenhum dia útil encontrado nos próximos 10 dias');
	});

	const celula = page.locator(`button[data-data="${chave}"]`);
	await expect(celula).toHaveAttribute('title', 'Disponível (09:00–11:00)');
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
