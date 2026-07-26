import { describe, expect, it } from 'vitest';
import { queryDaPagina } from './paginacao';

describe('queryDaPagina', () => {
	it('devolve string vazia quando nada é pedido — a API aplica o padrão dela', () => {
		expect(queryDaPagina()).toBe('');
		expect(queryDaPagina({})).toBe('');
	});

	it('monta a query com os parâmetros informados', () => {
		expect(queryDaPagina({ limite: 50, offset: 100 })).toBe('?limite=50&offset=100');
	});

	it('omite o que não foi informado', () => {
		expect(queryDaPagina({ offset: 20 })).toBe('?offset=20');
		expect(queryDaPagina({ limite: 10 })).toBe('?limite=10');
	});

	it('mantém offset 0 explícito — é a primeira página, não "sem valor"', () => {
		expect(queryDaPagina({ offset: 0 })).toBe('?offset=0');
	});
});
