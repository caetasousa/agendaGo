import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
	cancelarAgendamento,
	confirmarAgendamento,
	consultarSlots,
	listarAgendamentosDoCliente,
	listarAgendamentosDoPrestador,
	solicitarAgendamento,
	solicitarConvidado
} from './appointments';
import { ApiError } from './client';

function mockFetch(resposta: Response) {
	const fn = vi.fn().mockResolvedValue(resposta);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.unstubAllGlobals();
});

describe('consultarSlots', () => {
	it('envia GET público com o período', async () => {
		const fn = mockFetch(new Response(JSON.stringify({ dias: [] }), { status: 200 }));
		await consultarSlots('prestador-1', '2026-08-10', '2026-08-16');
		expect(fn.mock.calls[0][0]).toContain('/providers/prestador-1/slots?de=2026-08-10&ate=2026-08-16');
	});
});

describe('solicitarAgendamento', () => {
	it('envia POST com o slot desejado e devolve a solicitação', async () => {
		const criado = {
			id: 'ag-1',
			data: '2026-08-10',
			inicioMinutos: 480,
			fimMinutos: 540,
			status: 'SOLICITADO',
			expiraEm: '2026-08-02T10:00:00Z'
		};
		const fn = mockFetch(new Response(JSON.stringify(criado), { status: 201 }));

		const resultado = await solicitarAgendamento({
			providerId: 'prestador-1',
			data: '2026-08-10',
			inicioMinutos: 480
		});
		expect(resultado.status).toBe('SOLICITADO');
		expect(fn.mock.calls[0][0]).toContain('/agendamentos');
		expect(fn.mock.calls[0][1]).toMatchObject({ method: 'POST', credentials: 'include' });
	});

	it('lança ApiError quando o horário já está ocupado', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'horário indisponível' }), { status: 409 }));
		await expect(
			solicitarAgendamento({ providerId: 'prestador-1', data: '2026-08-10', inicioMinutos: 480 })
		).rejects.toThrowError(new ApiError(409, 'horário indisponível'));
	});
});

describe('solicitarConvidado', () => {
	it('envia POST público com slot e contato do convidado', async () => {
		const criado = {
			id: 'ag-2',
			data: '2026-08-10',
			inicioMinutos: 480,
			fimMinutos: 540,
			status: 'SOLICITADO',
			expiraEm: '2026-08-02T10:00:00Z'
		};
		const fn = mockFetch(new Response(JSON.stringify(criado), { status: 201 }));

		const resultado = await solicitarConvidado({
			providerId: 'prestador-1',
			data: '2026-08-10',
			inicioMinutos: 480,
			nome: 'Convidada Silva',
			email: 'convidada@email.com',
			telefone: '(11) 99999-8888'
		});
		expect(resultado.status).toBe('SOLICITADO');
		expect(fn.mock.calls[0][0]).toContain('/agendamentos/convidado');
		expect(fn.mock.calls[0][1]).toMatchObject({ method: 'POST', credentials: 'include' });
		expect(JSON.parse(fn.mock.calls[0][1].body as string)).toMatchObject({
			nome: 'Convidada Silva',
			email: 'convidada@email.com',
			telefone: '(11) 99999-8888'
		});
	});

	it('propaga 400 quando o telefone é inválido', async () => {
		mockFetch(new Response(JSON.stringify({ erro: 'telefone é obrigatório' }), { status: 400 }));
		await expect(
			solicitarConvidado({
				providerId: 'prestador-1',
				data: '2026-08-10',
				inicioMinutos: 480,
				nome: 'Ana',
				email: 'ana@email.com',
				telefone: '123'
			})
		).rejects.toMatchObject({ status: 400 });
	});
});

describe('listagens', () => {
	it('cliente e prestador consultam rotas próprias', async () => {
		// um Response por chamada: o corpo só pode ser lido uma vez
		const fn = vi.fn((url: string, init?: RequestInit) =>
			Promise.resolve(new Response(JSON.stringify({ agendamentos: [] }), { status: 200 }))
		);
		vi.stubGlobal('fetch', fn);

		await listarAgendamentosDoCliente();
		expect(fn.mock.calls[0][0]).toContain('/clients/me/agendamentos');

		await listarAgendamentosDoPrestador();
		expect(fn.mock.calls[1][0]).toContain('/providers/me/agendamentos');
	});
});

describe('transições', () => {
	it('confirmar e cancelar fazem POST sem corpo na rota da ação', async () => {
		const fn = mockFetch(new Response(null, { status: 204 }));
		await confirmarAgendamento('ag-1');
		expect(fn.mock.calls[0][0]).toContain('/agendamentos/ag-1/confirmar');

		await cancelarAgendamento('ag-1');
		expect(fn.mock.calls[1][0]).toContain('/agendamentos/ag-1/cancelar');
		expect(fn.mock.calls[1][1]).toMatchObject({ method: 'POST' });
	});

	it('propaga a mensagem de antecedência insuficiente no cancelamento', async () => {
		mockFetch(
			new Response(JSON.stringify({ erro: 'cancelamento exige antecedência mínima antes do início' }), {
				status: 409
			})
		);
		await expect(cancelarAgendamento('ag-1')).rejects.toMatchObject({ status: 409 });
	});
});
