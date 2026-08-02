package usecase_test

import (
	"testing"

	ucprovider "agendago/internal/usecase/provider"
)

// O requisito que este teste protege: quem compartilhou /agendar/{uuid} antes
// do slug existir não pode receber 404. O caminho por id fica para sempre.
func TestBuscarResumoAceitaSlugEUUID(t *testing.T) {
	usuarios, membros, providers := fakesDePrestador()
	_, p := criarPrestador(usuarios, membros, providers, "11111111-2222-3333-4444-555555555555", "João Barbeiro", "joao@email.com", "11999998888", "hash")
	providers.Salvar(p)

	buscar := ucprovider.NovoBuscarResumoUseCase(providers, membros)

	t.Run("acha pelo slug", func(t *testing.T) {
		r, err := buscar.Executar("joao-barbeiro")
		if err != nil {
			t.Fatalf("esperava sucesso pelo slug, got: %v", err)
		}
		if r.ID != p.ID {
			t.Errorf("esperava o prestador %s, got: %s", p.ID, r.ID)
		}
	})

	t.Run("continua achando pelo id — link antigo não quebra", func(t *testing.T) {
		r, err := buscar.Executar(p.ID)
		if err != nil {
			t.Fatalf("esperava sucesso pelo id, got: %v", err)
		}
		if r.ID != p.ID {
			t.Errorf("esperava o prestador %s, got: %s", p.ID, r.ID)
		}
	})

	t.Run("identificador desconhecido devolve não encontrado", func(t *testing.T) {
		_, err := buscar.Executar("nao-existe")
		if err != ucprovider.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
	})
}
