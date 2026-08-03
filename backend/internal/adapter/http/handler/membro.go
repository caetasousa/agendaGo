package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"agendago/internal/adapter/http/dto"
	"agendago/internal/domain/membro"
	"agendago/internal/domain/usuario"
	ucauth "agendago/internal/usecase/auth"
	ucmembro "agendago/internal/usecase/membro"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// MembroHandler concentra os handlers de equipe: convidar alguém para operar a
// agenda, aceitar o convite e administrar quem tem acesso.
// identidadeDoContexto é recebida como função pelo mesmo motivo dos demais
// handlers — evitar import cycle com o pacote middleware.
type MembroHandler struct {
	convidar             *ucmembro.ConvidarUseCase
	cancelarConvite      *ucmembro.CancelarConviteUseCase
	consultarConvite     *ucmembro.ConsultarConviteUseCase
	aceitarConvite       *ucmembro.AceitarConviteUseCase
	listarEquipe         *ucmembro.ListarEquipeUseCase
	removerMembro        *ucmembro.RemoverMembroUseCase
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool)
}

func NovoMembroHandler(
	convidar *ucmembro.ConvidarUseCase,
	cancelarConvite *ucmembro.CancelarConviteUseCase,
	consultarConvite *ucmembro.ConsultarConviteUseCase,
	aceitarConvite *ucmembro.AceitarConviteUseCase,
	listarEquipe *ucmembro.ListarEquipeUseCase,
	removerMembro *ucmembro.RemoverMembroUseCase,
	identidadeDoContexto func(r *http.Request) (ucauth.Identidade, bool),
) *MembroHandler {
	return &MembroHandler{
		convidar:             convidar,
		cancelarConvite:      cancelarConvite,
		consultarConvite:     consultarConvite,
		aceitarConvite:       aceitarConvite,
		listarEquipe:         listarEquipe,
		removerMembro:        removerMembro,
		identidadeDoContexto: identidadeDoContexto,
	}
}

// ListarEquipe godoc
//
//	@Summary		Listar quem opera a agenda
//	@Description	Devolve os membros da agenda do prestador autenticado e os convites ainda não aceitos.
//	@Tags			membros
//	@Produce		json
//	@Success		200	{object}	dto.EquipeResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/providers/me/membros [get]
func (h *MembroHandler) ListarEquipe(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	out, err := h.listarEquipe.Executar(id.ProviderID)
	if err != nil {
		switch {
		case errors.Is(err, ucmembro.ErrEquipeDesativada):
			responderErro(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ucmembro.ErrProviderNaoEncontrado):
			responderErro(w, http.StatusNotFound, err.Error())
		default:
			responderErroInterno(w, r, err)
		}
		return
	}

	resp := dto.EquipeResponse{
		Membros:   make([]dto.MembroResponse, 0, len(out.Membros)),
		Pendentes: make([]dto.ConvitePendenteResponse, 0, len(out.Pendentes)),
	}
	for _, m := range out.Membros {
		resp.Membros = append(resp.Membros, dto.MembroResponse{
			ID: m.ID, Email: m.Email, Papel: m.Papel, Ativo: m.Ativo, EhDono: m.EhDono, CriadoEm: m.CriadoEm,
		})
	}
	for _, c := range out.Pendentes {
		resp.Pendentes = append(resp.Pendentes, dto.ConvitePendenteResponse{
			Email: c.Email, Papel: c.Papel, ExpiraEm: c.ExpiraEm,
		})
	}
	responderJSON(w, http.StatusOK, resp)
}

// Convidar godoc
//
//	@Summary		Convidar alguém para operar a agenda
//	@Description	Envia um convite por email. Só o dono da agenda pode convidar. Recusa emails que já tenham conta no sistema.
//	@Tags			membros
//	@Accept			json
//	@Param			body	body	dto.ConvidarMembroRequest	true	"Email e papel"
//	@Success		204
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Router			/providers/me/membros [post]
func (h *MembroHandler) Convidar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	var req dto.ConvidarMembroRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responderErro(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if err := req.Validar(); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			responderErro(w, http.StatusBadRequest, mensagemValidacao(ve[0]))
			return
		}
		responderErro(w, http.StatusBadRequest, err.Error())
		return
	}

	papel := membro.PapelOperador
	if req.Papel != "" {
		papel = membro.Papel(req.Papel)
	}

	err := h.convidar.Executar(ucmembro.ConvidarInput{
		ProviderID: id.ProviderID,
		Email:      req.Email,
		Papel:      papel,
	})
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	// 409 e não 400: o pedido está bem formado, o que conflita é o estado do
	// sistema — aquele email já pertence a alguém.
	case errors.Is(err, ucmembro.ErrEmailIndisponivel):
		responderErro(w, http.StatusConflict, err.Error())
	case errors.Is(err, ucmembro.ErrEquipeDesativada):
		responderErro(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ucmembro.ErrProviderNaoEncontrado):
		responderErro(w, http.StatusNotFound, err.Error())
	default:
		responderErroInterno(w, r, err)
	}
}

// CancelarConvite godoc
//
//	@Summary		Cancelar um convite pendente
//	@Description	Invalida o link enviado a um email que ainda não aceitou. Só o dono da agenda pode cancelar.
//	@Tags			membros
//	@Param			email	path	string	true	"Email convidado"
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/providers/me/convites/{email} [delete]
func (h *MembroHandler) CancelarConvite(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	err := h.cancelarConvite.Executar(ucmembro.CancelarInput{
		ProviderID: id.ProviderID,
		Email:      chi.URLParam(r, "email"),
	})
	if err != nil {
		responderErroInterno(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Remover godoc
//
//	@Summary		Remover o acesso de alguém à agenda
//	@Description	Apaga o vínculo e encerra as sessões da pessoa removida. Só o dono da agenda pode remover, e o dono não pode ser removido.
//	@Tags			membros
//	@Param			id	path	string	true	"ID do vínculo"
//	@Success		204
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/providers/me/membros/{id} [delete]
func (h *MembroHandler) Remover(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identidadeDoContexto(r)
	if !ok {
		responderErro(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	err := h.removerMembro.Executar(ucmembro.RemoverInput{
		ProviderID: id.ProviderID,
		MembroID:   chi.URLParam(r, "id"),
	})
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ucmembro.ErrMembroNaoEncontrado):
		responderErro(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ucmembro.ErrNaoRemoveDono):
		responderErro(w, http.StatusConflict, err.Error())
	default:
		responderErroInterno(w, r, err)
	}
}

// ConsultarConvite godoc
//
//	@Summary		Consultar um convite pelo token
//	@Description	Devolve de qual agenda é o convite, para a tela de aceite. Rota pública — quem foi convidado ainda não tem conta. Consultar não gasta o convite.
//	@Tags			membros
//	@Produce		json
//	@Param			token	query		string	true	"Token do convite"
//	@Success		200		{object}	dto.ConviteResponse
//	@Failure		400		{object}	map[string]string
//	@Router			/membros/convite [get]
func (h *MembroHandler) ConsultarConvite(w http.ResponseWriter, r *http.Request) {
	tokenPuro := r.URL.Query().Get("token")
	if tokenPuro == "" {
		responderErro(w, http.StatusBadRequest, "token é obrigatório")
		return
	}

	out, err := h.consultarConvite.Executar(tokenPuro)
	switch {
	case err == nil:
		responderJSON(w, http.StatusOK, dto.ConviteResponse{
			Email: out.Email, NomeAgenda: out.NomeAgenda, Papel: out.Papel,
		})
	case errors.Is(err, ucmembro.ErrConviteInvalido):
		responderErro(w, http.StatusBadRequest, err.Error())
	default:
		responderErroInterno(w, r, err)
	}
}

// AceitarConvite godoc
//
//	@Summary		Aceitar um convite e criar o acesso
//	@Description	Cria a conta de quem foi convidado e o vínculo com a agenda. Rota pública, protegida pelo token de uso único. Nenhuma agenda nova é criada.
//	@Tags			membros
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.AceitarConviteRequest	true	"Token, telefone e senha"
//	@Success		201		{object}	dto.ConviteResponse
//	@Failure		400		{object}	map[string]string
//	@Router			/membros/aceitar-convite [post]
func (h *MembroHandler) AceitarConvite(w http.ResponseWriter, r *http.Request) {
	var req dto.AceitarConviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responderErro(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if err := req.Validar(); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			responderErro(w, http.StatusBadRequest, mensagemValidacao(ve[0]))
			return
		}
		responderErro(w, http.StatusBadRequest, err.Error())
		return
	}

	out, err := h.aceitarConvite.Executar(ucmembro.AceitarInput{
		Token:    req.Token,
		Telefone: req.Telefone,
		Senha:    req.Senha,
	})
	switch {
	case err == nil:
		// Não emite sessão: quem aceitou vai para o login, como em qualquer
		// cadastro do sistema.
		responderJSON(w, http.StatusCreated, dto.ConviteResponse{NomeAgenda: out.NomeAgenda})
	case errors.Is(err, ucmembro.ErrConviteInvalido):
		responderErro(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, usuario.ErrTelefoneObrigatorio):
		responderErro(w, http.StatusBadRequest, err.Error())
	default:
		responderErroInterno(w, r, err)
	}
}
