import { test, expect, type Page, type APIRequestContext } from '@playwright/test';
import { emailUnico, cadastrarPrestador, tokenDeConvite, SENHA_PADRAO } from './helpers';

// A Fase 2 separou a conta da agenda para que uma segunda pessoa pudesse
// operá-la sem a senha do dono. Este spec exerce o caminho inteiro dessa
// promessa: convidar, aceitar, entrar e operar de fato.

async function entrar(page: Page, email: string) {
	await page.goto('/login');
	await page.fill('#email', email);
	await page.fill('#senha', SENHA_PADRAO);
	await page.click('button[type="submit"]');
	await page.waitForURL('/painel');
}

async function sair(page: Page) {
	await page.locator('header').getByText('Sair').click();
	await page.waitForURL('/');
}

// ativarEquipe liga o recurso em Configurações. A equipe nasce desligada — sem
// isso a rota /painel/equipe redireciona e a API recusa convidar e listar.
async function ativarEquipe(page: Page) {
	await page.goto('/painel/configuracoes');
	await page.click('label[for="permite-equipe"]');
	await page.click('button:has-text("Salvar alterações")');
	await expect(page.getByText('Salvo', { exact: true })).toBeVisible();
}

// prepararDona cria a prestadora, liga a equipe e a deixa logada no painel de equipe.
async function prepararDona(page: Page, request: APIRequestContext): Promise<string> {
	const email = emailUnico('dona');
	await cadastrarPrestador(page, request, `Agenda da Dona ${Date.now()}`, email);
	await entrar(page, email);
	await ativarEquipe(page);
	await page.goto('/painel/equipe');
	return email;
}

test('dona convida, operadora cria o acesso e passa a operar a agenda', async ({ page, request }) => {
	const emailDona = await prepararDona(page, request);
	const emailOperadora = emailUnico('operadora');

	// A dona aparece na equipe, marcada como dona e sem opção de se remover.
	await expect(page.getByText(emailDona)).toBeVisible();
	await expect(page.getByText('Dona da agenda')).toBeVisible();

	await page.fill('#email-convite', emailOperadora);
	await page.click('button:has-text("Enviar convite")');
	await expect(page.getByText(`Convite enviado para ${emailOperadora}`)).toBeVisible();
	await expect(page.locator(`[data-convite="${emailOperadora}"]`)).toBeVisible();

	// A operadora recebe o link e cria o próprio acesso — sem nunca saber a
	// senha da dona.
	const token = await tokenDeConvite(request, emailOperadora);
	await sair(page);
	await page.goto(`/convite?token=${token}`);
	await expect(page.getByText(emailOperadora)).toBeVisible();

	await page.fill('#telefone', '(11) 98888-7777');
	await page.fill('#senha', SENHA_PADRAO);
	await page.fill('#confirmar-senha', SENHA_PADRAO);
	await page.click('button:has-text("Criar meu acesso")');
	await expect(page.getByText('Acesso criado')).toBeVisible();

	// O ponto do desenho: ela entra e cai na agenda de QUEM A CONVIDOU, não
	// numa agenda própria — o convite não cria agenda nenhuma.
	await entrar(page, emailOperadora);
	await page.goto('/painel/equipe');
	await expect(page.getByText(emailDona)).toBeVisible();
	await expect(page.getByText(emailOperadora)).toBeVisible();

	// E opera de fato: configurações é uma rota de gestão da agenda. A barra de
	// salvar só aparece com alteração pendente, daí o toggle antes do submit.
	await page.goto('/painel/configuracoes');
	await page.click('label[for="aceita-agendamentos"]');
	await page.click('button[type="submit"]');
	await expect(page.getByText('Salvo', { exact: true })).toBeVisible();
});

test('operadora não administra a equipe — só o dono convida e remove', async ({ page, request }) => {
	await prepararDona(page, request);
	const emailOperadora = emailUnico('sem-poder');

	await page.fill('#email-convite', emailOperadora);
	await page.click('button:has-text("Enviar convite")');
	await expect(page.getByText(`Convite enviado para ${emailOperadora}`)).toBeVisible();

	const token = await tokenDeConvite(request, emailOperadora);
	await sair(page);
	await page.goto(`/convite?token=${token}`);
	await page.fill('#telefone', '(11) 98888-7777');
	await page.fill('#senha', SENHA_PADRAO);
	await page.fill('#confirmar-senha', SENHA_PADRAO);
	await page.click('button:has-text("Criar meu acesso")');
	await expect(page.getByText('Acesso criado')).toBeVisible();

	await entrar(page, emailOperadora);
	await page.goto('/painel/equipe');

	// Ela vê quem tem acesso, mas não recebe as ferramentas de administrá-lo.
	await expect(page.getByText('Com acesso')).toBeVisible();
	await expect(page.locator('#email-convite')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Remover acesso' })).toHaveCount(0);
});

test('remover acesso pede confirmação num modal e apaga a conta órfã', async ({ page, request }) => {
	await prepararDona(page, request);
	const emailOperadora = emailUnico('removida');

	await page.fill('#email-convite', emailOperadora);
	await page.getByRole('button', { name: 'Enviar convite' }).click();
	await expect(page.getByText(`Convite enviado para ${emailOperadora}`)).toBeVisible();

	const token = await tokenDeConvite(request, emailOperadora);
	const paginaConvidada = await page.context().newPage();
	await paginaConvidada.goto(`/convite?token=${token}`);
	await paginaConvidada.fill('#telefone', '(11) 98888-7777');
	await paginaConvidada.fill('#senha', SENHA_PADRAO);
	await paginaConvidada.fill('#confirmar-senha', SENHA_PADRAO);
	await paginaConvidada.click('button:has-text("Criar meu acesso")');
	await expect(paginaConvidada.getByText('Acesso criado')).toBeVisible();
	await paginaConvidada.close();

	await page.reload();
	await page.getByRole('button', { name: 'Remover acesso' }).click();

	// Modal do projeto, não o confirm() do navegador: dá para ler o que vai
	// acontecer e desistir.
	const modal = page.getByRole('dialog', { name: 'Remover acesso' });
	await expect(modal).toBeVisible();
	await expect(modal.getByText(/a conta é apagada junto/i)).toBeVisible();

	await modal.getByRole('button', { name: 'Cancelar' }).click();
	await expect(modal).toHaveCount(0);
	await expect(page.getByText(emailOperadora)).toBeVisible();

	await page.getByRole('button', { name: 'Remover acesso' }).click();
	await page.getByRole('dialog').getByRole('button', { name: 'Remover acesso' }).click();
	await expect(page.getByText(`${emailOperadora} não tem mais acesso`)).toBeVisible();

	// A conta foi apagada, então o email volta a estar livre para um novo convite.
	await page.fill('#email-convite', emailOperadora);
	await page.getByRole('button', { name: 'Enviar convite' }).click();
	await expect(page.getByText(`Convite enviado para ${emailOperadora}`)).toBeVisible();
});

test('convite para email que já tem conta é recusado', async ({ page, request }) => {
	await prepararDona(page, request);

	// Um prestador já cadastrado: ele é dono da própria agenda, e um segundo
	// vínculo o deixaria caindo na agenda errada.
	const emailOutroPrestador = emailUnico('ja-prestador');
	const paginaAuxiliar = await page.context().newPage();
	await cadastrarPrestador(paginaAuxiliar, request, 'Outro Prestador', emailOutroPrestador);
	await paginaAuxiliar.close();

	await page.fill('#email-convite', emailOutroPrestador);
	await page.click('button:has-text("Enviar convite")');

	// A mensagem não diz QUE TIPO de conta existe — o convite não pode virar
	// uma sonda de emails cadastrados.
	await expect(page.getByText(/não foi possível convidar este email/i)).toBeVisible();
	await expect(page.locator(`[data-convite="${emailOutroPrestador}"]`)).toHaveCount(0);
});

// A equipe nasce desligada: quem trabalha sozinho não vê a tela nem carrega a
// ideia de convidar alguém. Ligar em Configurações é o que a revela.
test('equipe só aparece no painel depois de ser ativada em Configurações', async ({
	page,
	request
}) => {
	const email = emailUnico('sem-equipe');
	await cadastrarPrestador(page, request, `Agenda Solo ${Date.now()}`, email);
	await entrar(page, email);

	await expect(page.getByRole('link', { name: 'Equipe' })).toHaveCount(0);
	// Rota fora do menu também não entra — volta para onde o recurso se liga.
	await page.goto('/painel/equipe');
	await page.waitForURL('/painel/configuracoes');

	await page.click('label[for="permite-equipe"]');
	await page.click('button:has-text("Salvar alterações")');
	await expect(page.getByText('Salvo', { exact: true })).toBeVisible();

	await expect(page.getByRole('link', { name: 'Equipe' }).first()).toBeVisible();
	await page.goto('/painel/equipe');
	await expect(page.getByText('Convidar alguém')).toBeVisible();
});

// Desligar com alguém dentro esconderia da dona uma pessoa que continua
// operando a agenda — a API recusa, e a tela mostra o motivo.
test('equipe não é desativada enquanto alguém ainda tem acesso', async ({ page, request }) => {
	await prepararDona(page, request);
	const emailOperadora = emailUnico('ainda-dentro');

	await page.fill('#email-convite', emailOperadora);
	await page.click('button:has-text("Enviar convite")');
	await expect(page.getByText(`Convite enviado para ${emailOperadora}`)).toBeVisible();

	await page.goto('/painel/configuracoes');
	await page.click('label[for="permite-equipe"]');
	await page.click('button:has-text("Salvar alterações")');

	await expect(page.getByText(/remova quem ainda tem acesso/i)).toBeVisible();

	// E o recurso continua ligado: a Equipe segue no menu.
	await page.reload({ waitUntil: 'networkidle' });
	await expect(page.locator('#permite-equipe')).toBeChecked();
});

test('link de convite inválido explica o que fazer', async ({ page }) => {
	await page.goto('/convite?token=token-que-nao-existe');
	await expect(page.getByText('Convite inválido')).toBeVisible();
	await expect(page.getByText(/peça um novo a quem convidou/i)).toBeVisible();
});

// Botão com texto de baixo contraste passa despercebido por qualquer asserção
// de conteúdo: o texto ESTÁ no DOM, e o Playwright clica nele normalmente. Foi
// assim que "Enviar convite" e "Criar meu acesso" foram parar invisíveis — as
// classes de cor usadas (`text-surface`, `bg-ink`) não existiam no tema, e o
// texto caiu numa cor herdada quase igual à do fundo.
//
// Comparar as duas cores por igualdade não basta: o primeiro bug era branco
// sobre branco, mas cinza-claro sobre branco é igualmente ilegível e passaria.
// Daí a razão de contraste da WCAG, com o piso de 4.5:1 para texto normal.
async function razaoDeContraste(locator: import('@playwright/test').Locator): Promise<number> {
	return locator.evaluate((el) => {
		const canal = (c: number) => {
			const v = c / 255;
			return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
		};
		const luminancia = (rgb: string) => {
			const [r, g, b] = rgb.match(/\d+/g)!.map(Number);
			return 0.2126 * canal(r) + 0.7152 * canal(g) + 0.0722 * canal(b);
		};

		const cor = getComputedStyle(el).color;
		let no: HTMLElement | null = el as HTMLElement;
		let fundo = 'rgba(0, 0, 0, 0)';
		while (no && (fundo === 'rgba(0, 0, 0, 0)' || fundo === 'transparent')) {
			fundo = getComputedStyle(no).backgroundColor;
			no = no.parentElement;
		}

		const a = luminancia(cor);
		const b = luminancia(fundo);
		const [claro, escuro] = a > b ? [a, b] : [b, a];
		return (claro + 0.05) / (escuro + 0.05);
	});
}

const CONTRASTE_MINIMO = 4.5;

test('os botões do convite têm texto legível', async ({ page, request }) => {
	await prepararDona(page, request);

	const enviar = page.getByRole('button', { name: 'Enviar convite' });
	await expect(enviar).toBeVisible();
	expect(await razaoDeContraste(enviar), 'texto do botão sem contraste com o fundo').toBeGreaterThanOrEqual(
		CONTRASTE_MINIMO
	);

	// E na página pública, que é a primeira coisa que a pessoa convidada vê.
	const emailConvidada = emailUnico('legibilidade');
	await page.fill('#email-convite', emailConvidada);
	await enviar.click();
	await expect(page.getByText(`Convite enviado para ${emailConvidada}`)).toBeVisible();

	const token = await tokenDeConvite(request, emailConvidada);
	await sair(page);
	await page.goto(`/convite?token=${token}`);

	const criar = page.getByRole('button', { name: 'Criar meu acesso' });
	await expect(criar).toBeVisible();
	expect(await razaoDeContraste(criar), 'texto do botão sem contraste com o fundo').toBeGreaterThanOrEqual(
		CONTRASTE_MINIMO
	);
});
