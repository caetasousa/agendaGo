import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
	banirPrestador,
	detalharCliente,
	detalharPrestador,
	listarClientes,
	listarPrestadores,
	reativarCliente
} from './admin';

function mockFetch(resposta: Response) {
	const fn = vi.fn().mockResolvedValue(resposta);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.unstubAllGlobals();
});

describe('listagens de moderação', () => {
	it('prestadores e clientes consultam rotas de admin', async () => {
		const fn = vi.fn((url: string, init?: RequestInit) =>
			Promise.resolve(new Response(JSON.stringify({ usuarios: [] }), { status: 200 }))
		);
		vi.stubGlobal('fetch', fn);

		await listarPrestadores();
		expect(fn.mock.calls[0][0]).toContain('/admin/prestadores');
		await listarClientes();
		expect(fn.mock.calls[1][0]).toContain('/admin/clientes');
	});
});

describe('detalhes em leitura', () => {
	it('detalhar prestador consulta a rota com o id e devolve os agendamentos', async () => {
		const detalhe = {
			id: 'p-1',
			nome: 'João',
			email: 'joao@email.com',
			ativo: true,
			aceitaAgendamentos: true,
			descansoMinutos: 0,
			duracaoAtendimentoMinutos: 60,
			agendamentos: [
				{
					id: 'ag-1',
					data: '2026-08-10',
					inicioMinutos: 480,
					fimMinutos: 540,
					status: 'SOLICITADO',
					nomeCliente: 'Convidada',
					telefoneCliente: '(11) 99999-8888'
				}
			]
		};
		const fn = mockFetch(new Response(JSON.stringify(detalhe), { status: 200 }));
		const resultado = await detalharPrestador('p-1');
		expect(fn.mock.calls[0][0]).toContain('/admin/prestadores/p-1');
		expect(resultado.agendamentos[0].telefoneCliente).toBe('(11) 99999-8888');
	});

	it('detalhar cliente consulta a rota com o id', async () => {
		const detalhe = {
			id: 'c-1',
			nome: 'Convidada',
			email: 'convidada@email.com',
			telefone: '(11) 99999-8888',
			ativo: true,
			temConta: false,
			agendamentos: []
		};
		const fn = mockFetch(new Response(JSON.stringify(detalhe), { status: 200 }));
		const resultado = await detalharCliente('c-1');
		expect(fn.mock.calls[0][0]).toContain('/admin/clientes/c-1');
		expect(resultado.temConta).toBe(false);
	});

	it('propaga 404 quando o usuário não existe', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'não encontrado' }), { status: 404 }));
		await expect(detalharPrestador('fantasma')).rejects.toMatchObject({ status: 404 });
	});
});

describe('ações de moderação', () => {
	it('banir prestador faz POST na rota de banir', async () => {
		const fn = mockFetch(new Response(null, { status: 204 }));
		await banirPrestador('p-1');
		expect(fn.mock.calls[0][0]).toContain('/admin/prestadores/p-1/banir');
		expect(fn.mock.calls[0][1]).toMatchObject({ method: 'POST', credentials: 'include' });
	});

	it('reativar cliente faz POST na rota de reativar', async () => {
		const fn = mockFetch(new Response(null, { status: 204 }));
		await reativarCliente('c-1');
		expect(fn.mock.calls[0][0]).toContain('/admin/clientes/c-1/reativar');
	});
});
