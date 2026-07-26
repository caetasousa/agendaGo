import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	// Sem isto o Vite só expõe variáveis com prefixo VITE_ (o SvelteKit não
	// configura o prefixo por conta própria), e `import.meta.env.PUBLIC_API_URL`
	// ficaria undefined no build de produção — a imagem cairia calada no
	// fallback localhost:8080 e o frontend publicado não acharia a API.
	envPrefix: 'PUBLIC_',
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) => filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// adapter-node gera o build de produção como servidor Node autônomo
			// (build/index.js) — o modo dev (npm run dev) não muda.
			adapter: adapter()
		})
	],
	test: {
		include: ['src/**/*.test.ts'],
		environment: 'node'
	}
});
