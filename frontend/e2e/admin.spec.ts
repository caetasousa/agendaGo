import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { emailUnico, tokenDeConfirmacaoCadastro } from './helpers';

// Credenciais do admin semeado pelo docker-compose (ADMIN_EMAIL/ADMIN_SENHA).
const ADMIN_EMAIL = 'admin@agendago.dev';
const ADMIN_SENHA = 'admin12345';

async function cadastrarPrestador(page: Page, nome: string, email: string) {
	await page.goto('/cadastro?tipo=prestador');
	await page.fill('#nome', nome);
	await page.fill('#email', email);
	await page.fill('#telefone', '(11) 99999-8888');
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
	await page.click('button:has-text("Sair")');
	await page.waitForURL('/');
}

// cadastrarClienteLogado cria a conta de cliente, confirma pelo link do
// email (via Mailpit) e loga — a conta só nasce após a confirmação.
async function cadastrarClienteLogado(
	page: Page,
	request: APIRequestContext,
	nome: string,
	email: string
) {
	await page.goto('/cadastro?tipo=cliente');
	await page.fill('#nome', nome);
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
}

async function entrarComoAdmin(page: Page) {
	await page.goto('/login');
	await page.fill('#email', ADMIN_EMAIL);
	await page.fill('#senha', ADMIN_SENHA);
	await page.click('button[type="submit"]');
	await page.waitForURL('/admin');
}

test('admin entra pelo login unificado e cai no painel de moderação', async ({ page }) => {
	await entrarComoAdmin(page);
	await expect(page.getByRole('heading', { name: 'Moderação' })).toBeVisible();
	await expect(page.locator('header').getByText('Moderação')).toBeVisible();
});

test('admin bane um prestador, que deixa de conseguir logar, e depois reativa', async ({ page }) => {
	const nome = `Prestador Moderado ${Date.now()}`;
	const email = emailUnico('moderado');
	await cadastrarPrestador(page, nome, email);

	await entrarComoAdmin(page);

	const linha = page.locator(`li[data-usuario]:has-text("${nome}")`);
	await expect(linha).toHaveAttribute('data-ativo', 'true');
	await linha.getByRole('button', { name: 'Banir' }).click();
	await expect(linha).toHaveAttribute('data-ativo', 'false');
	await expect(linha.getByText('Banido')).toBeVisible();

	// prestador banido não loga (403 → mensagem de desativado)
	await page.locator('header').getByText('Sair').click();
	await page.waitForURL('/');
	await page.goto('/login');
	await page.fill('#email', email);
	await page.fill('#senha', '12345678');
	await page.click('button[type="submit"]');
	await expect(page.getByText('usuário desativado')).toBeVisible();

	// admin reativa e o prestador volta a logar
	await entrarComoAdmin(page);
	const linhaReativar = page.locator(`li[data-usuario]:has-text("${nome}")`);
	await linhaReativar.getByRole('button', { name: 'Reativar' }).click();
	await expect(linhaReativar).toHaveAttribute('data-ativo', 'true');

	await page.locator('header').getByText('Sair').click();
	await page.waitForURL('/');
	await page.goto('/login');
	await page.fill('#email', email);
	await page.fill('#senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
});

test('admin abre o detalhe em leitura de um prestador pela lista', async ({ page }) => {
	const nome = `Prestador Detalhe ${Date.now()}`;
	const email = emailUnico('detalhe');
	await cadastrarPrestador(page, nome, email);

	await entrarComoAdmin(page);
	// clica no nome do prestador na lista → página de detalhe
	await page.locator(`li[data-usuario] a[data-detalhe]:has-text("${nome}")`).click();
	await page.waitForURL('**/admin/prestadores/**');

	await expect(page.getByRole('heading', { name: nome })).toBeVisible();
	await expect(page.getByText(email)).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Agendamentos recebidos' })).toBeVisible();
	// prestador recém-cadastrado ainda não recebeu agendamentos
	await expect(page.getByText('ainda não recebeu agendamentos')).toBeVisible();

	// o link de voltar retorna à moderação
	await page.getByRole('link', { name: '← Voltar à moderação' }).click();
	await page.waitForURL('**/admin');
});

test('prestador banido some da vitrine pública', async ({ page, request }) => {
	const nome = `Prestador Vitrine ${Date.now()}`;
	const email = emailUnico('vitrine-ban');
	await cadastrarPrestador(page, nome, email);

	// aparece na vitrine antes do banimento (visto por um cliente)
	await cadastrarClienteLogado(page, request, 'Cliente Vitrine', emailUnico('cliente-vitrine'));
	await page.goto('/painel/agendar');
	await expect(page.locator(`a:has-text("${nome}")`)).toBeVisible();
	await page.click('button:has-text("Sair")');
	await page.waitForURL('/');

	// admin bane
	await entrarComoAdmin(page);
	await page.locator(`li[data-usuario]:has-text("${nome}")`).getByRole('button', { name: 'Banir' }).click();
	await expect(page.locator(`li[data-usuario]:has-text("${nome}")`)).toHaveAttribute('data-ativo', 'false');
	await page.locator('header').getByText('Sair').click();
	await page.waitForURL('/');

	// um cliente novo já não vê o prestador banido na vitrine
	await cadastrarClienteLogado(page, request, 'Cliente Vitrine 2', emailUnico('cliente-vitrine-2'));
	await page.goto('/painel/agendar');
	await expect(page.locator(`a:has-text("${nome}")`)).toHaveCount(0);
});

test('prestador logado não acessa /admin (é mandado ao painel)', async ({ page }) => {
	// cadastro já deixa o prestador logado no /painel
	await page.goto('/cadastro?tipo=prestador');
	await page.fill('#nome', 'Prestador Guard');
	await page.fill('#email', emailUnico('guard'));
	await page.fill('#telefone', '(11) 99999-8888');
	await page.fill('#senha', '12345678');
	await page.fill('#confirmar-senha', '12345678');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');

	await page.goto('/admin');
	await page.waitForURL('/painel');
});
