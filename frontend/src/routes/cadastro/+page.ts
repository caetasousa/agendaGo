// O form só deve ser interativo depois que o Svelte hidratar os listeners
// (onsubmit). Com SSR, existe uma janela real em que o HTML chega do
// servidor mas o JS ainda não anexou os handlers, e um clique no botão
// dispara o submit nativo do form (GET com os campos na querystring).
export const ssr = false;
