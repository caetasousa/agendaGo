package usecase_test

import (
	"agendago/internal/domain/membro"
	"agendago/internal/domain/provider"
	"agendago/internal/domain/usuario"
	"agendago/test/repository/memoria"
)

// Desde a V14 um prestador são três peças: a conta que loga (usuario), a
// agenda que ela opera (provider) e o vínculo entre as duas (membro). Este
// arquivo evita que cada teste remonte isso na mão.

// fakesDePrestador monta os três fakes já ligados entre si — o de agendas
// consulta o de vínculos para filtrar a vitrine, e o de vínculos consulta o de
// contas para saber quem está banido, espelhando os joins do Postgres.
func fakesDePrestador() (*memoria.UsuarioMemoria, *memoria.MembroMemoria, *memoria.ProviderMemoria) {
	usuarios := memoria.NovoUsuarioMemoria()
	membros := memoria.NovoMembroMemoriaCom(usuarios)
	providers := memoria.NovoProviderMemoriaCom(membros)
	return usuarios, membros, providers
}

// criarPrestador persiste conta, agenda e vínculo de dono, e devolve as duas
// pontas. O id da conta é o mesmo da agenda, como a migração V14 fez com os
// prestadores que já existiam — assim um teste que só conhece um id continua
// falando do mesmo prestador dos dois lados.
func criarPrestador(
	usuarios *memoria.UsuarioMemoria,
	membros *memoria.MembroMemoria,
	providers *memoria.ProviderMemoria,
	id, nome, email, telefone, senhaHash string,
) (*usuario.Usuario, *provider.Provider) {
	u, err := usuario.Novo(id, email, telefone, senhaHash)
	if err != nil {
		panic("conta de teste inválida: " + err.Error())
	}
	if err := usuarios.Salvar(u); err != nil {
		panic(err)
	}

	p, err := provider.Novo(id, nome)
	if err != nil {
		panic("agenda de teste inválida: " + err.Error())
	}
	if err := providers.Salvar(p); err != nil {
		panic(err)
	}

	m, err := membro.Novo("m-"+id, u.ID, p.ID, membro.PapelDono)
	if err != nil {
		panic(err)
	}
	if err := membros.Salvar(m); err != nil {
		panic(err)
	}
	return u, p
}

// senhaDeTeste é o hash usado quando o teste não se importa com a senha — só
// precisa de uma conta válida. Onde a senha importa (login, recuperação), o
// teste gera o hash de verdade com o hasher.
const senhaDeTeste = "hash-de-teste"
