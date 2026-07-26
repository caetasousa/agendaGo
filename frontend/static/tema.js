// Aplica o tema salvo antes da primeira pintura (evita flash de tema errado).
// Padrão da marca é escuro; só troca para claro se o usuário escolheu.
//
// Vive num arquivo estático, e não inline no app.html, por causa da CSP: o
// SvelteKit só calcula hash dos scripts que ele mesmo gera, então um <script>
// inline escrito à mão seria bloqueado por `script-src 'self'`. Carregado sem
// defer/async no <head>, continua rodando antes da primeira pintura.
(function () {
	try {
		var salvo = localStorage.getItem('theme');
		document.documentElement.dataset.theme = salvo === 'light' ? 'light' : 'dark';
	} catch (e) {
		document.documentElement.dataset.theme = 'dark';
	}
})();
