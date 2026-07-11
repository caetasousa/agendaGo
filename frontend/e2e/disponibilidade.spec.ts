import { test, expect, type Page } from '@playwright/test';
import { emailUnico } from './helpers';

// Formata a data como YYYY-MM-DD no fuso local, igual ao data-data das células.
function chaveLocal(data: Date): string {
	const ano = data.getFullYear();
	const mes = String(data.getMonth() + 1).padStart(2, '0');
	const dia = String(data.getDate()).padStart(2, '0');
	return `${ano}-${mes}-${dia}`;
}

async function cadastrarPrestador(page: Page, prefixo: string) {
	await page.goto('/cadastro');
	await page.fill('#nome', 'Disponibilidade Teste');
	await page.fill('#email', emailUnico(prefixo));
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
}

// A agenda nasce desativada; o expediente padrão dos dias úteis só vale após
// ativar em Preferências.
async function ativarAgenda(page: Page) {
	await page.goto('/painel/preferencias');
	await page.check('#aceita-agendamentos');
	await page.click('button:has-text("Salvar")');
	await expect(page.getByText('Preferências salvas.')).toBeVisible();
}

async function irParaDisponibilidade(page: Page) {
	await page.goto('/painel/disponibilidade');
	await page.waitForSelector('button[data-data]');
}

test('agenda desativada mostra aviso e todos os dias indisponíveis', async ({ page }) => {
	await cadastrarPrestador(page, 'agenda-desativada');
	await irParaDisponibilidade(page);

	await expect(page.getByText('Sua agenda está desativada')).toBeVisible();

	const hoje = new Date();
	const celulaHoje = page.locator(`button[data-data="${chaveLocal(hoje)}"]`);
	await expect(celulaHoje).toHaveAttribute('data-estado', 'indisponivel');
});

test('prestador bloqueia um dia útil e depois restaura o padrão', async ({ page }) => {
	await cadastrarPrestador(page, 'bloqueio');
	await ativarAgenda(page);
	await irParaDisponibilidade(page);

	// dia 15 do mês seguinte, sempre no futuro; se cair no fim de semana
	// (indisponível), usa o dia 16 ou 17 — algum dos três é dia útil
	await page.click('button[aria-label="Próximo mês"]');
	const hoje = new Date();
	let celula = page.locator('__nunca__');
	for (const dia of [15, 16, 17]) {
		const chave = chaveLocal(new Date(hoje.getFullYear(), hoje.getMonth() + 1, dia));
		const candidata = page.locator(`button[data-data="${chave}"]`);
		if ((await candidata.getAttribute('data-estado')) === 'disponivel') {
			celula = candidata;
			break;
		}
	}

	// bloqueia pelo modal
	await celula.click();
	await page.click('button:has-text("Marcar indisponível")');
	await expect(celula).toHaveAttribute('data-estado', 'bloqueado');

	// recarrega e confirma persistência
	await page.reload({ waitUntil: 'networkidle' });
	await page.waitForSelector('button[data-data]');
	await page.click('button[aria-label="Próximo mês"]');
	await expect(celula).toHaveAttribute('data-estado', 'bloqueado');

	// restaura o padrão pelo modal
	await celula.click();
	await page.click('button:has-text("Restaurar padrão")');
	await expect(celula).toHaveAttribute('data-estado', 'disponivel');
});

test('prestador define horários próprios num sábado', async ({ page }) => {
	await cadastrarPrestador(page, 'personalizado');
	await ativarAgenda(page);
	await irParaDisponibilidade(page);

	// primeiro sábado do mês seguinte
	await page.click('button[aria-label="Próximo mês"]');
	const hoje = new Date();
	const primeiroDia = new Date(hoje.getFullYear(), hoje.getMonth() + 1, 1);
	const sabado = new Date(primeiroDia);
	sabado.setDate(1 + ((6 - primeiroDia.getDay() + 7) % 7));

	const celula = page.locator(`button[data-data="${chaveLocal(sabado)}"]`);
	await expect(celula).toHaveAttribute('data-estado', 'indisponivel');

	// abre o modal (vem preenchido com o expediente padrão) e salva
	await celula.click();
	await page.click('button:has-text("Salvar horários")');
	await expect(celula).toHaveAttribute('data-estado', 'personalizado');

	// horários aparecem na célula (08:00–12:00 e 14:00–18:00)
	await expect(celula.getByText('8h–12h')).toBeVisible();
});

test('qualquer alteração no dia de hoje em cima da hora é impedida', async ({ page }) => {
	await cadastrarPrestador(page, 'antecedencia');
	await ativarAgenda(page);
	await irParaDisponibilidade(page);

	const celulaHoje = page.locator(`button[data-data="${chaveLocal(new Date())}"]`);
	const estadoInicial = await celulaHoje.getAttribute('data-estado');

	await celulaHoje.click();
	await expect(
		page.getByText('Alterar a disponibilidade do dia de hoje exige ao menos 24h de antecedência.')
	).toBeVisible();

	// nem ampliar (salvar horários) nem reduzir (marcar indisponível) é permitido hoje,
	// independente do dia começar disponível ou indisponível
	await expect(page.locator('button:has-text("Salvar horários")')).toBeDisabled();
	await expect(page.locator('button:has-text("Marcar indisponível")')).toBeDisabled();

	await page.click('button:has-text("Cancelar")');
	await expect(celulaHoje).toHaveAttribute('data-estado', estadoInicial!);
});

test('restaurar padrão a partir de um bloqueio de hoje continua permitido', async ({ page }) => {
	await cadastrarPrestador(page, 'restaurar-hoje');
	await ativarAgenda(page);
	await irParaDisponibilidade(page);

	const celulaHoje = page.locator(`button[data-data="${chaveLocal(new Date())}"]`);

	// só é possível reproduzir o bloqueio de hoje via API, já que a UI já
	// impede bloquear hoje pela própria regra sob teste
	await page.evaluate(async (hoje) => {
		await fetch(`http://localhost:8080/providers/me/dias/${hoje}`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			credentials: 'include',
			body: JSON.stringify({ tipo: 'bloqueio', blocos: [] })
		});
	}, chaveLocal(new Date()));

	await page.reload({ waitUntil: 'networkidle' });
	await page.waitForSelector('button[data-data]');
	await expect(celulaHoje).toHaveAttribute('data-estado', 'bloqueado');

	await celulaHoje.click();
	await expect(page.locator('button:has-text("Restaurar padrão")')).toBeEnabled();
	await page.click('button:has-text("Restaurar padrão")');
	await expect(celulaHoje).not.toHaveAttribute('data-estado', 'bloqueado');
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
