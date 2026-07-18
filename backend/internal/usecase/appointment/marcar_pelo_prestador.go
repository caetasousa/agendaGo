package appointment

import (
	"time"

	"agendago/internal/domain/client"

	"github.com/google/uuid"
)

// MarcarPeloPrestadorInput contém os dados da marcação feita pelo próprio
// prestador (ex.: cliente ligou por telefone). ProviderID vem da identidade
// da sessão autenticada, nunca do corpo da requisição. É um registro
// puramente interno: só nome e uma observação livre e opcional — sem
// telefone, sem email, sem notificação ao cliente.
type MarcarPeloPrestadorInput struct {
	ProviderID    string
	Data          time.Time
	InicioMinutos int
	Nome          string
	Observacao    string
	Agora         time.Time
}

// MarcarPeloPrestadorUseCase registra na agenda do próprio prestador um
// agendamento para um cliente que o contatou por fora (telefone, WhatsApp).
// Reaproveita a barreira anti-overbooking de SolicitarUseCase, com duas
// diferenças: o slot é ofertável mesmo com a agenda fechada ao público (é o
// dono marcando), e o agendamento nasce CONFIRMADO direto — não há pedido de
// ninguém para aceitar. Cada marcação cria um cliente novo, só com o nome —
// sem email ou telefone não há chave de reuso nem como notificar ninguém. Só
// funciona se o prestador não tiver desativado a funcionalidade em
// Preferências (PermiteMarcacaoPeloPrestador).
type MarcarPeloPrestadorUseCase struct {
	solicitar    *SolicitarUseCase
	clientRepo   repositorioClient
	providerRepo repositorioProvider
}

// NovoMarcarPeloPrestadorUseCase cria uma instância de MarcarPeloPrestadorUseCase com as dependências injetadas.
func NovoMarcarPeloPrestadorUseCase(
	solicitar *SolicitarUseCase,
	clientRepo repositorioClient,
	providerRepo repositorioProvider,
) *MarcarPeloPrestadorUseCase {
	return &MarcarPeloPrestadorUseCase{
		solicitar:    solicitar,
		clientRepo:   clientRepo,
		providerRepo: providerRepo,
	}
}

// Executar cria um cliente só-nome e reserva o slot na agenda do prestador da
// sessão, incluindo horários de agenda fechada ao público. O agendamento
// nasce CONFIRMADO direto — não há pedido de ninguém para o prestador
// aceitar — e pode ser cancelado por ele a qualquer momento, sem antecedência
// mínima. Não há notificação: sem email, não há para quem enviar. Retorna
// ErrMarcacaoPeloPrestadorNaoPermitida se o prestador desativou a
// funcionalidade em Preferências.
func (uc *MarcarPeloPrestadorUseCase) Executar(in MarcarPeloPrestadorInput) (*SolicitarOutput, error) {
	p, err := uc.providerRepo.BuscarPorID(in.ProviderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProviderNaoEncontrado
	}
	if !p.PermiteMarcacaoPeloPrestador {
		return nil, ErrMarcacaoPeloPrestadorNaoPermitida
	}

	convidado, err := client.NovoRegistradoPeloPrestador(uuid.NewString(), in.Nome)
	if err != nil {
		return nil, err
	}
	if err := uc.clientRepo.Salvar(convidado); err != nil {
		return nil, err
	}

	novo, err := uc.solicitar.reservarSlot(in.ProviderID, convidado.ID, in.Data, in.InicioMinutos, in.Agora, true, in.Observacao, true)
	if err != nil {
		return nil, err
	}

	return novaSaidaSolicitar(novo), nil
}
