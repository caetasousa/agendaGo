// Mesmo motivo do /login: sem SSR, o SvelteKit só renderiza a página após o
// JS hidratar, eliminando a janela em que um clique dispara o submit nativo
// do form antes do onsubmit ser anexado.
export const ssr = false;
