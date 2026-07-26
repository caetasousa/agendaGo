import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig(({ mode }) => {
	const dev = mode !== 'production';

	return {
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
					runes: ({ filename }) =>
						filename.split(/[/\\]/).includes('node_modules') ? undefined : true
				},

				// adapter-node gera o build de produção como servidor Node autônomo
				// (build/index.js) — o modo dev (npm run dev) não muda.
				adapter: adapter(),

				// Content-Security-Policy: a rede de proteção para o dia em que um XSS
				// escapar (dependência comprometida, um `@html` distraído). `mode: 'hash'`
				// deixa o SvelteKit hashear os scripts que ele mesmo gera, o que permite
				// `script-src 'self'` sem 'unsafe-inline'.
				//
				// Em desenvolvimento a API vive noutra origem (localhost:8080) e o Vite
				// abre um WebSocket para o HMR — daí as duas exceções, ausentes do build
				// de produção, onde o Caddy serve front e API na mesma origem.
				// As palavras-chave vão sem aspas: o SvelteKit acrescenta as aspas
				// simples exigidas pela sintaxe da CSP ao montar o cabeçalho.
				csp: {
					mode: 'hash',
					directives: {
						'default-src': ['self'],
						'script-src': ['self'],
						// Tailwind e o SvelteKit injetam <style> inline no SSR; hash não
						// cobre atributo style, que os componentes usam.
						'style-src': ['self', 'unsafe-inline'],
						'img-src': ['self', 'data:'],
						'font-src': ['self'],
						'connect-src': dev ? ['self', 'http://localhost:8080', 'ws:'] : ['self'],
						'object-src': ['none'],
						'base-uri': ['self'],
						'form-action': ['self'],
						'frame-ancestors': ['none']
					}
				}
			})
		],
		test: {
			include: ['src/**/*.test.ts'],
			environment: 'node'
		}
	};
});
