package handler

import (
	"errors"
	"net/http"
	"time"

	"agendago/internal/adapter/http/dto"
	ucadmin "agendago/internal/usecase/admin"

	"github.com/go-chi/chi/v5"
)

// AdminHandler concentra os handlers de moderação: listar e detalhar
// prestadores/clientes e bani-los/reativá-los. Todas as rotas ficam sob ExigirAdmin.
type AdminHandler struct {
	moderar  *ucadmin.ModerarUseCase
	detalhar *ucadmin.DetalharUseCase
}

// NovoAdminHandler cria uma instância de AdminHandler com os usecases injetados.
func NovoAdminHandler(moderar *ucadmin.ModerarUseCase, detalhar *ucadmin.DetalharUseCase) *AdminHandler {
	return &AdminHandler{moderar: moderar, detalhar: detalhar}
}

// ListarPrestadores godoc
//
//	@Summary		Listar prestadores (admin)
//	@Description	Lista todos os prestadores com o status de moderação (ativo/banido)
//	@Tags			admin
//	@Produce		json
//	@Success		200	{object}	dto.ListarUsuariosResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/admin/prestadores [get]
func (h *AdminHandler) ListarPrestadores(w http.ResponseWriter, r *http.Request) {
	usuarios, err := h.moderar.ListarPrestadores()
	if err != nil {
		responderErroInterno(w, r, err)
		return
	}
	responderJSON(w, http.StatusOK, dto.ListarUsuariosResponse{Usuarios: usuariosParaDTO(usuarios)})
}

// ListarClientes godoc
//
//	@Summary		Listar clientes (admin)
//	@Description	Lista os clientes com conta e o status de moderação (ativo/banido)
//	@Tags			admin
//	@Produce		json
//	@Success		200	{object}	dto.ListarUsuariosResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/admin/clientes [get]
func (h *AdminHandler) ListarClientes(w http.ResponseWriter, r *http.Request) {
	usuarios, err := h.moderar.ListarClientes()
	if err != nil {
		responderErroInterno(w, r, err)
		return
	}
	responderJSON(w, http.StatusOK, dto.ListarUsuariosResponse{Usuarios: usuariosParaDTO(usuarios)})
}

// DetalharPrestador godoc
//
//	@Summary		Detalhar prestador (admin)
//	@Description	Dados cadastrais do prestador e os agendamentos que ele recebeu (leitura)
//	@Tags			admin
//	@Produce		json
//	@Param			id	path		string	true	"ID do prestador"
//	@Success		200	{object}	dto.DetalhePrestadorResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/admin/prestadores/{id} [get]
func (h *AdminHandler) DetalharPrestador(w http.ResponseWriter, r *http.Request) {
	d, err := h.detalhar.Prestador(chi.URLParam(r, "id"), time.Now())
	if err != nil {
		h.responder(w, r, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.DetalhePrestadorResponse{
		ID:                 d.ID,
		Nome:               d.Nome,
		Email:              d.Email,
		Ativo:              d.Ativo,
		AceitaAgendamentos: d.AceitaAgendamentos,
		DescansoMinutos:    d.DescansoMinutos,
		DuracaoMinutos:     d.DuracaoMinutos,
		Agendamentos:       agendamentosAdminParaDTO(d.Agendamentos),
	})
}

// DetalharCliente godoc
//
//	@Summary		Detalhar cliente (admin)
//	@Description	Dados cadastrais do cliente e os agendamentos que ele fez (leitura)
//	@Tags			admin
//	@Produce		json
//	@Param			id	path		string	true	"ID do cliente"
//	@Success		200	{object}	dto.DetalheClienteResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/admin/clientes/{id} [get]
func (h *AdminHandler) DetalharCliente(w http.ResponseWriter, r *http.Request) {
	d, err := h.detalhar.Cliente(chi.URLParam(r, "id"), time.Now())
	if err != nil {
		h.responder(w, r, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.DetalheClienteResponse{
		ID:           d.ID,
		Nome:         d.Nome,
		Email:        d.Email,
		Telefone:     d.Telefone,
		Ativo:        d.Ativo,
		TemConta:     d.TemConta,
		Agendamentos: agendamentosAdminParaDTO(d.Agendamentos),
	})
}

// BanirPrestador godoc
//
//	@Summary		Banir um prestador
//	@Description	Desativa o prestador: ele deixa de logar, some da vitrine e não oferta horários
//	@Tags			admin
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/admin/prestadores/{id}/banir [post]
func (h *AdminHandler) BanirPrestador(w http.ResponseWriter, r *http.Request) {
	h.responder(w, r, h.moderar.BanirPrestador(chi.URLParam(r, "id")))
}

// ReativarPrestador godoc
//
//	@Summary		Reativar um prestador
//	@Description	Reverte o banimento de um prestador
//	@Tags			admin
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/admin/prestadores/{id}/reativar [post]
func (h *AdminHandler) ReativarPrestador(w http.ResponseWriter, r *http.Request) {
	h.responder(w, r, h.moderar.ReativarPrestador(chi.URLParam(r, "id")))
}

// BanirCliente godoc
//
//	@Summary		Banir um cliente
//	@Description	Desativa o cliente: ele deixa de logar
//	@Tags			admin
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/admin/clientes/{id}/banir [post]
func (h *AdminHandler) BanirCliente(w http.ResponseWriter, r *http.Request) {
	h.responder(w, r, h.moderar.BanirCliente(chi.URLParam(r, "id")))
}

// ReativarCliente godoc
//
//	@Summary		Reativar um cliente
//	@Description	Reverte o banimento de um cliente
//	@Tags			admin
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/admin/clientes/{id}/reativar [post]
func (h *AdminHandler) ReativarCliente(w http.ResponseWriter, r *http.Request) {
	h.responder(w, r, h.moderar.ReativarCliente(chi.URLParam(r, "id")))
}

func (h *AdminHandler) responder(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ucadmin.ErrProviderNaoEncontrado), errors.Is(err, ucadmin.ErrClientNaoEncontrado):
		responderErro(w, http.StatusNotFound, err.Error())
	default:
		responderErroInterno(w, r, err)
	}
}

func usuariosParaDTO(us []ucadmin.UsuarioResumo) []dto.UsuarioModeracaoDTO {
	resultado := make([]dto.UsuarioModeracaoDTO, 0, len(us))
	for _, u := range us {
		resultado = append(resultado, dto.UsuarioModeracaoDTO{
			ID:                 u.ID,
			Nome:               u.Nome,
			Email:              u.Email,
			Ativo:              u.Ativo,
			AceitaAgendamentos: u.AceitaAgendamentos,
		})
	}
	return resultado
}

const layoutDataAdmin = "2006-01-02"

func agendamentosAdminParaDTO(as []ucadmin.AgendamentoResumo) []dto.AgendamentoAdminDTO {
	resultado := make([]dto.AgendamentoAdminDTO, 0, len(as))
	for _, a := range as {
		resultado = append(resultado, dto.AgendamentoAdminDTO{
			ID:              a.ID,
			Data:            a.Data.Format(layoutDataAdmin),
			InicioMinutos:   a.InicioMinutos,
			FimMinutos:      a.FimMinutos,
			Status:          a.Status,
			NomeCliente:     a.NomeCliente,
			EmailCliente:    a.EmailCliente,
			TelefoneCliente: a.TelefoneCliente,
			NomePrestador:   a.NomePrestador,
		})
	}
	return resultado
}
