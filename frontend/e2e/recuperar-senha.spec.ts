import { test, expect } from '@playwright/test';
import { emailUnico, tokenDeRecuperacao } from './helpers';

// Fluxo completo: cadastra um prestador, pede recuperação, pega o token no
// Mailpit, redefine a senha e loga com a nova. Depende de o SMTP da API estar
// apontando para o Mailpit do compose (padrão de desenvolvimento).
test('recupera a senha e loga com a nova', async ({ page, request }) => {
	const email = emailUnico('recuperar');

	await page.goto('/cadastro');
	await page.fill('#nome', 'Prestador Recuperar');
	await page.fill('#email', email);
	await page.fill('#senha', 'senha-antiga1');
	await page.fill('#confirmar-senha', 'senha-antiga1');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
	await page.click('button:has-text("Sair")');
	await page.waitForURL('/');

	// solicita a recuperação a partir do link na tela de login
	await page.goto('/login');
	await page.click('a:has-text("Esqueci minha senha")');
	await page.waitForURL('/recuperar-senha');
	await page.fill('#email', email);
	await page.click('button[type="submit"]');
	await expect(page.getByText('Se este email estiver cadastrado')).toBeVisible();

	// pega o token do email capturado e abre a página de redefinição
	const token = await tokenDeRecuperacao(request, email);
	await page.goto(`/redefinir-senha?token=${token}`);
	await page.fill('#novaSenha', 'senha-nova1');
	await page.fill('#confirmacao', 'senha-nova1');
	await page.click('button[type="submit"]');
	await expect(page.getByText('Senha redefinida com sucesso')).toBeVisible();

	// a senha antiga não funciona mais; a nova sim
	await page.goto('/login');
	await page.fill('#email', email);
	await page.fill('#senha', 'senha-antiga1');
	await page.click('button[type="submit"]');
	await expect(page.getByText('credenciais inválidas')).toBeVisible();

	await page.fill('#senha', 'senha-nova1');
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
});

// A tela de recuperação responde igual exista ou não a conta — anti-enumeração.
test('recuperação responde igual para email desconhecido', async ({ page }) => {
	await page.goto('/recuperar-senha');
	await page.fill('#email', emailUnico('desconhecido'));
	await page.click('button[type="submit"]');
	await expect(page.getByText('Se este email estiver cadastrado')).toBeVisible();
});

// Validações client-side da página de redefinição, sem depender de email.
test('redefinição valida token ausente e senhas no cliente', async ({ page }) => {
	// sem token na URL: a página nem mostra o formulário
	await page.goto('/redefinir-senha');
	await expect(page.getByText('Link inválido')).toBeVisible();

	// com token qualquer, valida as senhas antes de chamar a API
	await page.goto('/redefinir-senha?token=qualquer-coisa');

	await page.fill('#novaSenha', 'curta');
	await page.fill('#confirmacao', 'curta');
	await page.click('button[type="submit"]');
	await expect(page.getByText('pelo menos 8 caracteres')).toBeVisible();

	await page.fill('#novaSenha', 'senha-longa1');
	await page.fill('#confirmacao', 'senha-diferente1');
	await page.click('button[type="submit"]');
	await expect(page.getByText('não coincidem')).toBeVisible();
});
