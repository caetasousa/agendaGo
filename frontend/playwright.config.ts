import { defineConfig } from '@playwright/test';

// Pré-requisito: `docker compose up` na raiz do repositório
// (API em :8080 e web em :5173 no ar). Sem webServer de propósito:
// o compose já serve o app completo, incluindo banco e migrations.
export default defineConfig({
	testDir: './e2e',
	use: { baseURL: 'http://localhost:5173' },
	projects: [{ name: 'chromium', use: { browserName: 'chromium' } }]
});
