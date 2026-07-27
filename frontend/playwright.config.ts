import { defineConfig } from '@playwright/test';

// Pré-requisito: `docker compose up` na raiz do repositório
// (API em :8080 e web em :5173 no ar). Sem webServer de propósito:
// o compose já serve o app completo, incluindo banco e migrations.
export default defineConfig({
	testDir: './e2e',

	// Só no CI: localmente, um teste que falha deve falhar na hora. No CI a
	// suíte disputa CPU com Postgres, API e Vite no mesmo runner, e parte do
	// que aparece como falha é lentidão momentânea.
	retries: process.env.CI ? 2 : 0,

	// `github` anota a falha na linha exata do arquivo, direto no commit; o
	// relatório HTML é o que o CI guarda como artefato para abrir depois.
	reporter: process.env.CI ? [['html', { open: 'never' }], ['github']] : 'list',

	use: {
		baseURL: 'http://localhost:5173',
		// Só na retentativa: o trace grava DOM, rede e screenshot de cada passo,
		// pesado demais para ligar em toda execução. Na primeira falha ele ainda
		// não existe; na repetição, sim — que é quando se vai investigar.
		trace: 'on-first-retry',
		screenshot: 'only-on-failure',
		video: 'retain-on-failure'
	},

	projects: [{ name: 'chromium', use: { browserName: 'chromium' } }]
});
