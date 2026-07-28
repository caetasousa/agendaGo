<!-- Set de ícones da interface: SVG inline com traço uniforme, para não
     depender de biblioteca externa nem de fonte de ícones. Todos partilham o
     mesmo viewBox 24 e herdam a cor do texto (currentColor). -->
<script lang="ts">
	type Nome =
		| 'inicio'
		| 'agendamentos'
		| 'disponibilidade'
		| 'preferencias'
		| 'equipe'
		| 'marcar'
		| 'agendar'
		| 'moderacao'
		| 'sair'
		| 'copiar'
		| 'check'
		| 'seta-direita';

	let {
		nome,
		tamanho = 18,
		classe = ''
	}: { nome: Nome; tamanho?: number; classe?: string } = $props();

	// Traçados no padrão de 24x24 com stroke — desenhados para ler bem a 18px.
	const tracados: Record<Nome, string> = {
		inicio: 'M3 10.5 12 3l9 7.5M5.5 9.5V20a1 1 0 0 0 1 1h11a1 1 0 0 0 1-1V9.5',
		agendamentos: 'M9 6h11M9 12h11M9 18h11M3.5 6l1 1 2-2M3.5 12l1 1 2-2M3.5 18l1 1 2-2',
		disponibilidade:
			'M8 2v3M16 2v3M3.5 9h17M4.5 5.5h15a1 1 0 0 1 1 1V20a1 1 0 0 1-1 1h-15a1 1 0 0 1-1-1V6.5a1 1 0 0 1 1-1ZM7.5 12.5h2M11 12.5h2M14.5 12.5h2M7.5 16.5h2M11 16.5h2',
		preferencias:
			'M4 6h10M18 6h2M4 12h2M10 12h10M4 18h8M16 18h4M16 6a2 2 0 1 0 0-.01M8 12a2 2 0 1 0 0-.01M14 18a2 2 0 1 0 0-.01',
		marcar:
			'M8 2v3M16 2v3M3.5 9h17M4.5 5.5h15a1 1 0 0 1 1 1V20a1 1 0 0 1-1 1h-15a1 1 0 0 1-1-1V6.5a1 1 0 0 1 1-1ZM12 12.5v5M9.5 15h5',
		agendar: 'M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14ZM16.2 16.2 21 21',
		equipe:
			'M9.5 11a3.25 3.25 0 1 0 0-6.5 3.25 3.25 0 0 0 0 6.5ZM3 19.5c0-3 2.9-4.75 6.5-4.75s6.5 1.75 6.5 4.75M16 5.2a3.25 3.25 0 0 1 0 6.1M18 14.9c2 .6 3.5 1.9 3.5 4.1',
		moderacao: 'M12 3l7.5 3v5.5c0 4.3-3 8.3-7.5 9.5-4.5-1.2-7.5-5.2-7.5-9.5V6L12 3Z',
		sair: 'M9 21H5.5a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1H9M16 16l4-4-4-4M20 12H9',
		copiar:
			'M9 9.5A1.5 1.5 0 0 1 10.5 8h8A1.5 1.5 0 0 1 20 9.5v9a1.5 1.5 0 0 1-1.5 1.5h-8A1.5 1.5 0 0 1 9 18.5v-9ZM5.5 16A1.5 1.5 0 0 1 4 14.5v-9A1.5 1.5 0 0 1 5.5 4h9A1.5 1.5 0 0 1 16 5.5',
		check: 'M20 6 9 17l-5-5',
		'seta-direita': 'M5 12h14M13 6l6 6-6 6'
	};
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	width={tamanho}
	height={tamanho}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="1.75"
	stroke-linecap="round"
	stroke-linejoin="round"
	class={classe}
	aria-hidden="true"
>
	<path d={tracados[nome]} />
</svg>
