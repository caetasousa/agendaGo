package usecase_test

import (
	"testing"
	"time"

	"agendago/internal/domain/availability"
	"agendago/internal/domain/convite"
	"agendago/internal/domain/membro"
	"agendago/internal/domain/provider"
	ucprovider "agendago/internal/usecase/provider"
	"agendago/test/repository/memoria"
)

func novoProviderComPreferencias(
	usuarios *memoria.UsuarioMemoria,
	membros *memoria.MembroMemoria,
	providers *memoria.ProviderMemoria,
) *provider.Provider {
	_, p := criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaDeTeste)
	return p
}

func TestAtualizarPreferenciasProvider(t *testing.T) {
	t.Run("ativa a agenda e define o descanso", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		novoProviderComPreferencias(usuarios, membros, providers)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())

		out, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			UsuarioID:                 "provider-1",
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           15,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if !out.AceitaAgendamentos {
			t.Error("esperava agenda ativada")
		}
		if out.DescansoMinutos != 15 {
			t.Errorf("esperava descanso 15, got: %d", out.DescansoMinutos)
		}

		persistido, _ := providers.BuscarPorID("provider-1")
		if !persistido.AceitaAgendamentos || persistido.DescansoMinutos != 15 {
			t.Error("esperava que as preferências fossem persistidas")
		}
	})

	t.Run("desativa a agenda", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		p := novoProviderComPreferencias(usuarios, membros, providers)
		p.AtivarAgenda()
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())

		out, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			UsuarioID:                 "provider-1",
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        false,
			DescansoMinutos:           0,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.AceitaAgendamentos {
			t.Error("esperava agenda desativada")
		}
	})

	t.Run("desativa e reativa a marcação pelo prestador", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		// a marcação pelo prestador nasce desativada; ativa antes de exercitar o toggle
		_, p := criarPrestador(usuarios, membros, providers, "provider-1", "João Silva", "joao@email.com", "11999998888", senhaDeTeste)
		p.PermiteMarcacaoPeloPrestador = true
		providers.Salvar(p)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())

		out, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			UsuarioID:                    "provider-1",
			ProviderID:                   "provider-1",
			Telefone:                     "11999998888",
			DuracaoAtendimentoMinutos:    60,
			AceitaAgendamentos:           true,
			DescansoMinutos:              0,
			PermiteMarcacaoPeloPrestador: false,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if out.PermiteMarcacaoPeloPrestador {
			t.Error("esperava marcação pelo prestador desativada na saída")
		}
		persistido, _ := providers.BuscarPorID("provider-1")
		if persistido.PermiteMarcacaoPeloPrestador {
			t.Error("esperava marcação pelo prestador desativada persistida")
		}

		out, err = uc.Executar(ucprovider.AtualizarPreferenciasInput{
			UsuarioID:                    "provider-1",
			ProviderID:                   "provider-1",
			Telefone:                     "11999998888",
			DuracaoAtendimentoMinutos:    60,
			AceitaAgendamentos:           true,
			DescansoMinutos:              0,
			PermiteMarcacaoPeloPrestador: true,
		})
		if err != nil {
			t.Fatalf("esperava sucesso na reativação, got: %v", err)
		}
		if !out.PermiteMarcacaoPeloPrestador {
			t.Error("esperava marcação pelo prestador reativada")
		}
	})

	t.Run("retorna erro quando descanso é negativo", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		novoProviderComPreferencias(usuarios, membros, providers)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())

		_, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			UsuarioID:                 "provider-1",
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           -1,
		})
		if err != provider.ErrDescansoInvalido {
			t.Errorf("esperava ErrDescansoInvalido, got: %v", err)
		}
	})

	t.Run("retorna erro quando prestador não existe", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())

		_, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			UsuarioID:                 "provider-1",
			ProviderID:                "id-inexistente",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           0,
		})
		if err != ucprovider.ErrProviderNaoEncontrado {
			t.Errorf("esperava ErrProviderNaoEncontrado, got: %v", err)
		}
	})

	t.Run("define o expediente padrão com três blocos curtos", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		novoProviderComPreferencias(usuarios, membros, providers)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())

		out, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			UsuarioID:                 "provider-1",
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           15,
			HorariosPadrao: []ucprovider.BlocoInput{
				{InicioMinutos: 8 * 60, FimMinutos: 10 * 60},
				{InicioMinutos: 11 * 60, FimMinutos: 13 * 60},
				{InicioMinutos: 15 * 60, FimMinutos: 17 * 60},
			},
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.HorariosPadrao) != 3 {
			t.Fatalf("esperava 3 blocos, got: %d", len(out.HorariosPadrao))
		}

		persistido, _ := providers.BuscarPorID("provider-1")
		if len(persistido.HorariosPadrao) != 3 {
			t.Error("esperava que o expediente padrão fosse persistido")
		}
	})

	t.Run("aceita expediente padrão vazio (nenhum horário)", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		novoProviderComPreferencias(usuarios, membros, providers)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())

		out, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			UsuarioID:                 "provider-1",
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        false,
			DescansoMinutos:           0,
			HorariosPadrao:            nil,
		})
		if err != nil {
			t.Fatalf("esperava sucesso, got: %v", err)
		}
		if len(out.HorariosPadrao) != 0 {
			t.Errorf("esperava nenhum bloco, got: %v", out.HorariosPadrao)
		}
	})

	t.Run("retorna erro quando um bloco do expediente padrão é inválido", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		novoProviderComPreferencias(usuarios, membros, providers)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())

		_, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			UsuarioID:                 "provider-1",
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           0,
			HorariosPadrao:            []ucprovider.BlocoInput{{InicioMinutos: 12 * 60, FimMinutos: 8 * 60}},
		})
		if err != availability.ErrFimAntesDoInicio {
			t.Errorf("esperava ErrFimAntesDoInicio, got: %v", err)
		}
	})

	t.Run("retorna erro quando blocos do expediente padrão se sobrepõem", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		novoProviderComPreferencias(usuarios, membros, providers)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())

		_, err := uc.Executar(ucprovider.AtualizarPreferenciasInput{
			UsuarioID:                 "provider-1",
			ProviderID:                "provider-1",
			Telefone:                  "11999998888",
			DuracaoAtendimentoMinutos: 60,
			AceitaAgendamentos:        true,
			DescansoMinutos:           0,
			HorariosPadrao: []ucprovider.BlocoInput{
				{InicioMinutos: 8 * 60, FimMinutos: 13 * 60},
				{InicioMinutos: 12 * 60, FimMinutos: 14 * 60},
			},
		})
		if err != availability.ErrBlocosSobrepostos {
			t.Errorf("esperava ErrBlocosSobrepostos, got: %v", err)
		}
	})

	t.Run("ativa e desativa a equipe numa agenda de uma pessoa só", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		novoProviderComPreferencias(usuarios, membros, providers)
		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())

		out, err := uc.Executar(entradaComEquipe(true))
		if err != nil {
			t.Fatalf("esperava sucesso ao ativar, got: %v", err)
		}
		if !out.PermiteEquipe {
			t.Error("esperava equipe ativada na saída")
		}
		persistido, _ := providers.BuscarPorID("provider-1")
		if !persistido.PermiteEquipe {
			t.Error("esperava equipe ativada persistida")
		}

		out, err = uc.Executar(entradaComEquipe(false))
		if err != nil {
			t.Fatalf("esperava sucesso ao desativar, got: %v", err)
		}
		if out.PermiteEquipe {
			t.Error("esperava equipe desativada na saída")
		}
	})

	// Desligar a equipe com alguém ainda dentro esconderia da dona uma pessoa
	// que continua operando a agenda — daí a recusa em vez do desligamento.
	t.Run("recusa desativar a equipe com outro membro na agenda", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		p := novoProviderComPreferencias(usuarios, membros, providers)
		p.AtivarEquipe()
		providers.Salvar(p)

		criarPrestador(usuarios, membros, providers, "operadora-1", "Recepção", "recepcao@email.com", "11988887777", senhaDeTeste)
		vinculo, err := membro.Novo("m-operadora-1", "operadora-1", "provider-1", membro.PapelOperador)
		if err != nil {
			t.Fatalf("vínculo de teste inválido: %v", err)
		}
		membros.Salvar(vinculo)

		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, memoria.NovoConviteMemoria())
		if _, err := uc.Executar(entradaComEquipe(false)); err != ucprovider.ErrEquipeComMembros {
			t.Errorf("esperava ErrEquipeComMembros, got: %v", err)
		}

		persistido, _ := providers.BuscarPorID("provider-1")
		if !persistido.PermiteEquipe {
			t.Error("esperava a equipe continuar ativada após a recusa")
		}
	})

	t.Run("recusa desativar a equipe com convite pendente", func(t *testing.T) {
		usuarios, membros, providers := fakesDePrestador()
		p := novoProviderComPreferencias(usuarios, membros, providers)
		p.AtivarEquipe()
		providers.Salvar(p)

		convites := memoria.NovoConviteMemoria()
		pendente, err := convite.Novo("hash-de-teste", "recepcao@email.com", "provider-1", membro.PapelOperador, time.Hour)
		if err != nil {
			t.Fatalf("convite de teste inválido: %v", err)
		}
		convites.Salvar(pendente)

		uc := ucprovider.NovoAtualizarPreferenciasUseCase(providers, usuarios, membros, convites)
		if _, err := uc.Executar(entradaComEquipe(false)); err != ucprovider.ErrEquipeComMembros {
			t.Errorf("esperava ErrEquipeComMembros, got: %v", err)
		}
	})
}

// entradaComEquipe monta o input mínimo válido das preferências variando só o
// que estes casos exercitam.
func entradaComEquipe(permite bool) ucprovider.AtualizarPreferenciasInput {
	return ucprovider.AtualizarPreferenciasInput{
		UsuarioID:                 "provider-1",
		ProviderID:                "provider-1",
		Telefone:                  "11999998888",
		DuracaoAtendimentoMinutos: 60,
		AceitaAgendamentos:        true,
		PermiteEquipe:             permite,
	}
}
