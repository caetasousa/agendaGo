// Mesmo motivo do /login: sem SSR, a página só renderiza após o JS hidratar,
// eliminando a janela em que um clique dispara o submit nativo antes do
// onsubmit ser anexado.
export const ssr = false;
