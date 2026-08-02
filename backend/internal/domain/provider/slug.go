package provider

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	// ErrSlugInvalido é retornado quando o slug tem caracteres fora de
	// [a-z0-9-], começa ou termina com hífen, ou está fora de [3, 60].
	ErrSlugInvalido = errors.New("o endereço deve ter de 3 a 60 caracteres, usando apenas letras, números e hífens")
	// ErrSlugReservado é retornado para um slug que colidiria com uma rota do
	// próprio site.
	ErrSlugReservado = errors.New("este endereço não está disponível")
)

const (
	tamanhoMinimoSlug = 3
	tamanhoMaximoSlug = 60
)

// formatoSlug aceita minúsculas, dígitos e hífens internos.
var formatoSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// semAcento mapeia as acentuadas do português para a letra base.
//
// Tabela explícita em vez de normalização Unicode (golang.org/x/text/norm):
// aquilo resolveria qualquer alfabeto, mas arrasta tabelas Unicode para dentro
// do binário — e o projeto acabou de tirar 10 MB dele por esse tipo de peso.
// Para nome de prestador em português, isto cobre o caso inteiro.
var semAcento = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n',
}

// reservados são os slugs que colidiriam com caminhos do próprio site. O link
// público é /agendar/{slug}, então a colisão real hoje é limitada; a lista
// cobre também os caminhos de primeiro nível para o dia em que o link encurtar
// para /{slug} — mudar isso depois exigiria renomear prestadores já existentes.
var reservados = map[string]bool{
	"admin": true, "api": true, "painel": true, "login": true, "logout": true,
	"cadastro": true, "agendar": true, "convite": true, "conta": true,
	"sobre": true, "ajuda": true, "suporte": true, "termos": true,
	"privacidade": true, "swagger": true, "health": true, "ready": true,
	"static": true, "assets": true, "favicon": true, "robots": true,
}

// GerarSlug deriva um slug a partir de um texto livre: tira acentos, baixa
// para minúsculas e troca tudo que não for [a-z0-9] por hífen, colapsando
// repetidos e removendo os das pontas.
//
// Não valida nem garante unicidade — é só a normalização. Quem decide se o
// resultado serve é DefinirSlug; a unicidade é do banco.
func GerarSlug(texto string) string {
	var b strings.Builder
	anteriorFoiHifen := false
	for _, r := range strings.ToLower(texto) {
		if base, ok := semAcento[r]; ok {
			r = base
		}
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			anteriorFoiHifen = false
		case !anteriorFoiHifen && b.Len() > 0:
			b.WriteRune('-')
			anteriorFoiHifen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// SlugReservado informa se o slug colide com um caminho do próprio site.
func SlugReservado(slug string) bool {
	return reservados[slug]
}

// DefinirSlug troca o endereço público do prestador.
//
// ⚠️ O link antigo deixa de funcionar assim que isto é chamado — quem tiver
// compartilhado o endereço anterior recebe 404. A tela avisa antes de salvar; o
// domínio não tem como saber quem já compartilhou.
//
// Retorna ErrSlugInvalido para formato fora do padrão e ErrSlugReservado para
// um dos caminhos do site. A unicidade NÃO é verificada aqui: quem responde por
// ela é o índice do banco, e checar antes abriria uma janela de corrida entre a
// checagem e o UPDATE.
func (p *Provider) DefinirSlug(slug string) error {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if len(slug) < tamanhoMinimoSlug || len(slug) > tamanhoMaximoSlug {
		return ErrSlugInvalido
	}
	if !formatoSlug.MatchString(slug) {
		return ErrSlugInvalido
	}
	if SlugReservado(slug) {
		return ErrSlugReservado
	}
	p.Slug = slug
	p.AtualizadoEm = time.Now()
	return nil
}
