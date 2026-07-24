# Regras de Negócio — agendaGo

Documento de referência das regras de negócio da agenda. Define como o prestador
configura sua disponibilidade, como os horários são ofertados ao cliente e como o
agendamento evolui até a conclusão.

## Visão geral do fluxo

A agenda se apoia em três camadas, refletidas no domínio
(`internal/domain/{user,availability,slot,appointment}`):

1. **Disponibilidade (regra)** — o prestador define quando trabalha.
2. **Slots (oferta)** — horários livres, **calculados** a partir da disponibilidade menos
   os agendamentos existentes.
3. **Agendamento (reserva)** — o cliente solicita um slot; o prestador confirma.

> A separação disponibilidade → slots → agendamento é o eixo do sistema. O cliente só
> consegue marcar em dias e horários efetivamente disponíveis.

---

## 1. Disponibilidade do prestador

### Expediente padrão
Não há grade recorrente por dia da semana: o prestador configura, em Preferências, um
único conjunto de **blocos de horário** (`horarios_padrao`) que vale igual de segunda a
sexta — quantos períodos quiser, inclusive blocos curtos, com os intervalos que fizer
sentido entre eles. Um prestador recém-cadastrado começa com uma sugestão de dia
comercial (08:00–12:00 e 14:00–18:00), livre para editar ou zerar. Sábados e domingos são
indisponíveis por padrão, independente do expediente configurado.

O expediente padrão só vale para o prestador com a agenda **ativa**
(`aceita_agendamentos = true`). Um prestador que não deseja atender mantém a flag
desativada e **nunca** oferta slots, mesmo com blocos configurados.

### Definições por data
O que o prestador persiste são apenas os **desvios** do padrão, data a data
(`date_exceptions`):
- **BLOQUEIO** — a data fica indisponível o dia inteiro (folga, feriado).
- **EXTRA** — a data ganha **horários personalizados**, que substituem o expediente
  padrão (serve tanto para atender num sábado quanto para mudar as horas de um dia útil).

A definição da data tem **precedência** sobre o expediente padrão, e é **uma por data**:
salvar de novo substitui a anterior (upsert).

### Validação dos blocos (estrita)
Ao salvar os horários de uma data, o sistema valida:
- Sem blocos **sobrepostos** no mesmo dia.
- `fim > início` (proíbe bloco invertido ou de duração zero).
- **Proíbe cruzar a meia-noite** — um expediente noturno deve ser partido em dois dias.
- Horários em **minutos cheios** (granularidade mínima a definir na implementação).
- Blocos **adjacentes** são mesclados (ex.: 08:00–12:00 + 12:00–14:00 → 08:00–14:00).

### Resolução da disponibilidade de um dia
A disponibilidade efetiva de uma data resolve-se nesta ordem:

```
definição da data  →  (se agenda ativa e dia útil) expediente padrão  →  indisponível
```

### Antecedência para alterar o dia de hoje
Qualquer alteração no dia **de hoje** — bloquear, definir/editar horários personalizados —
exige **24h de antecedência**: não se cancela nem se cria oferta "em cima da hora". A
única exceção é **restaurar o padrão a partir de um bloqueio**: isso só reduz a oferta
(nunca a amplia), então nunca surpreende um cliente. Hoje a regra é aplicada na interface;
quando existirem agendamentos, a validação passa a valer também no backend (um dia com
agendamento não poderá ser bloqueado).

---

## 2. Slots (horários ofertados)

Os slots são **calculados sob demanda**, não pré-gravados. O cliente, ao consultar,
recebe os horários livres calculados como:

```
slots livres = blocos do dia − agendamentos (SOLICITADO/CONFIRMADO)
```

> **Link público de agendamento** — cada prestador tem um link (`/agendar/{id}`,
> exibido no painel para compartilhar no Instagram/WhatsApp). Qualquer pessoa vê o
> calendário de horários livres sem cadastro; criar conta/entrar só é exigido na hora
> de solicitar.

### Fatiamento por duração + buffer
Cada bloco é fatiado em slots conforme a **duração do atendimento** somada ao **buffer**
do prestador. Enquanto não existe um domínio de serviços, a duração é única por
prestador (`duracao_atendimento_minutos`, configurável em Preferências, sugestão inicial
de 60 min):

- **Buffer configurável por prestador** (`descanso_minutos`: 0, 10, 15…) — intervalo de
  preparação/limpeza entre atendimentos. O próximo slot só abre após
  `duração + buffer`.
- **Sobra descartada** — só vira slot o intervalo que cabe o **atendimento inteiro
  (+ buffer)** dentro do bloco. O tempo que sobra no fim do bloco é ignorado; um
  atendimento nunca "vaza" para fora do bloco.
- **Nunca no passado** — dias anteriores a hoje não ofertam slots, e hoje só oferta
  horários que ainda não começaram.

---

## 3. Agendamento (reserva)

### Reserva ao solicitar + expiração (anti-overbooking)
Quando o cliente solicita um horário, o agendamento já **ocupa o intervalo** (reserva
pessimista) com um prazo de expiração (`expira_em = agora + TTL`). Outros clientes não
conseguem solicitar o mesmo intervalo.

- Se o prestador não confirmar dentro do TTL, a pendência vira **EXPIRADO** e o intervalo
  volta a ficar livre. A expiração é **lazy**: uma solicitação vencida deixa de ocupar o
  intervalo imediatamente e o status é efetivado na próxima leitura.
- O conflito é barrado na persistência **dentro de transação**: um lock na linha do
  prestador serializa as reservas concorrentes dele, e a checagem de conflito + INSERT
  acontecem sob esse lock. (Optamos pelo lock transacional em vez de uma constraint de
  exclusão para o schema não carregar regra de negócio — decisão registrada no CLAUDE.md.)

> Isso elimina a janela de overbooking entre "solicitar" e "confirmar".

### Agendamento sem cadastro (convidado)
Um visitante **sem conta** pode agendar pelo link público informando **nome, e-mail e
telefone** de contato. O sistema cria (ou reusa) um **cliente convidado** — um cliente
sem senha (`TemConta() == false`) — e reserva o slot como qualquer outra solicitação.

- O **telefone** passa por uma **validação leve**: exige ao menos 8 dígitos (formatação
  livre, sem verificação real). É guardado no cadastro do cliente para o prestador ter
  como retornar o contato.
- Se já existe um **convidado** com o mesmo e-mail, a reserva **reusa** esse convidado em
  vez de duplicar. Um cliente **banido** não agenda como convidado (bloqueado como
  qualquer inativo).
- E-mail de **conta registrada é rejeitado** (a resposta orienta a entrar): como o fluxo
  de convidado não verifica a posse do e-mail, aceitar permitiria criar agendamentos
  dentro da conta de um terceiro só conhecendo o e-mail dele.
- A rota pública tem **teto de solicitações por IP** (configurável; padrão 10/min) —
  sem ele, uma rajada encheria a agenda de um prestador com reservas falsas.
- Na listagem de agendamentos, o **prestador** enxerga o **nome, e-mail e telefone** do
  cliente — informação de contato que **não** é exposta na visão do próprio cliente.

### Cadastro de cliente e verificação por email
O cadastro de cliente (`POST /clients`) **não cria a conta na hora**: coleta nome, email,
**telefone** (obrigatório) e senha, guarda o cadastro pendente (com a senha já hasheada) e
envia um **email de confirmação**. A conta só nasce quando a pessoa clica no link
(`/confirmar-cadastro?token=...`) — prova de posse do email. O prestador, por outro lado,
cadastra e entra logado direto (sem essa etapa).

- **Herança do histórico de convidado**: se o email já pertence a um **convidado**, a
  confirmação **converte o mesmo registro** em conta (preservando o `client_id`), então
  todos os agendamentos que a pessoa fez como convidado passam a aparecer na conta dela.
  Essa é a razão de exigir verificação por email: sem prova de posse, qualquer um poderia
  reivindicar o histórico de um convidado só sabendo o email.
- **Email que já é conta**: a resposta na tela é **sempre a mesma** (anti-enumeração — não
  revela se o email existe), mas o email enviado é um aviso "você já tem conta, entre ou
  recupere a senha", em vez do link de confirmação.
- **Convidado banido** não vira conta ativa pelo cadastro.
- O **token de confirmação** é opaco (só o hash no banco), de **uso único**, com TTL de
  24h; um novo cadastro do mesmo email invalida os pendentes anteriores. As rotas têm teto
  por IP (mitiga spam de emails e força bruta de token).
- **Email é único no sistema**: um endereço só pode existir como cliente/convidado **ou**
  como prestador, nunca nos dois. O cadastro de prestador rejeita (409) um email já usado
  por cliente/convidado; o de cliente responde com o aviso "você já tem conta" quando o
  email pertence a um prestador (sem revelar isso na resposta HTTP).

### Login social (Google)
Cliente e prestador podem entrar sem senha, autenticando com Google. Diferente do cadastro
normal, a conta social **nasce ativa na hora** — não há confirmação por email, porque o
próprio Google já provou a posse do endereço.

- **Email verificado é exigido sempre**, antes de qualquer outra decisão — tanto para
  vincular a uma conta existente quanto para criar uma conta nova. Sem essa checagem,
  alguém poderia registrar um provedor OIDC com um email alheio ainda não confirmado e
  sequestrar a conta de outra pessoa, ou pré-criar uma conta num email que ainda não é seu.
- **Email inédito cria conta nova**, sem senha de verdade: o sistema gera uma senha
  aleatória, hasheia e descarta o valor em texto puro — ela nunca é comunicada e nunca serve
  para logar. É só para satisfazer a mesma invariante de domínio do cadastro por senha.
- **Email é único por tipo, mesma regra do cadastro normal**: um email que já é prestador
  não pode logar socialmente como cliente (nem o contrário) — o login social rejeita com o
  mesmo espírito de `ErrEmailJaCadastrado` do cadastro por senha, em vez de criar uma
  segunda conta paralela sob o mesmo email.
- **Prestador novo nasce com telefone pendente e fica travado até completar**: o Google
  não fornece telefone, mas o domínio de prestador exige um (é como o cliente entra em
  contato). Um prestador que nunca teve conta é criado na hora, no callback do Google, com
  um telefone-placeholder técnico (`TelefonePendente`, ver `usecase/auth/login_social.go`) e
  a agenda desativada (`AceitaAgendamentos=false`, o padrão de qualquer prestador novo). O
  frontend detecta esse placeholder (`MeResponse.telefonePendente`) e trava a navegação do
  painel em `/painel/preferencias` até um telefone de verdade ser salvo — nenhuma outra
  página do painel carrega enquanto isso. Cliente não passa por essa trava — o cadastro de
  cliente não exige telefone.
- **Convidados (sem conta) ficam de fora**: login social só se aplica a quem tem ou vai
  ganhar uma conta; o fluxo de agendamento sem cadastro não muda.
- **Banimento vale igual**: um cliente ou prestador banido pelo admin não consegue entrar
  por login social, mesma regra do login por senha.

### Confirmação
O agendamento só é **concluído após a confirmação do prestador**. Enquanto isso, fica
pendente (`SOLICITADO`) e ocupando o intervalo.

### Ciclo de vida (máquina de estados)

```
SOLICITADO ──► CONFIRMADO ──► REALIZADO
    │              │
    │              ├──► NÃO_COMPARECEU
    │              └──► CANCELADO
    ├──► RECUSADO        (cancelamento por cliente ou prestador)
    └──► EXPIRADO
```

- **SOLICITADO** — cliente pediu; ocupa o intervalo; aguardando o prestador.
- **CONFIRMADO** — prestador aceitou.
- **REALIZADO** — atendimento aconteceu.
- **RECUSADO** — prestador negou enquanto SOLICITADO.
- **EXPIRADO** — pendência venceu o TTL sem confirmação.
- **CANCELADO** — cancelado por cliente ou prestador (ver regra de antecedência).
- **NÃO_COMPARECEU** — confirmado, mas o cliente não apareceu.

Toda transição que encerra a reserva (RECUSADO, EXPIRADO, CANCELADO) **libera o intervalo**.

### Cancelamento
**Cliente e prestador** podem cancelar um agendamento `CONFIRMADO`, respeitando a
**antecedência mínima** (config: 24h antes do início). Cancelamentos dentro da janela
mínima são bloqueados (tratamento de penalidade fica fora de escopo por enquanto).
Ao cancelar, o intervalo volta a ficar livre.

O **cliente** também pode cancelar a própria solicitação ainda `SOLICITADO`, sem
exigência de antecedência — desistir de um pedido não confirmado não surpreende ninguém.
O **prestador** não cancela solicitações pendentes: para isso existe a recusa.

#### Cancelamento pelo convidado (por token no email)
O **convidado não tem conta**, então não conseguiria cancelar por rotas autenticadas.
Para isso, ao **confirmar** um agendamento de convidado, o sistema gera um **token de
cancelamento** de uso pessoal (opaco, só o hash vai ao banco — mesmo padrão do token de
recuperação de senha) e o envia no **email de confirmação** como um link
`/cancelar-agendamento/{token}`. O convidado abre o link, vê os detalhes e confirma o
cancelamento, **sem login**.

- O token substitui apenas a autenticação; a **regra de antecedência de 24h continua
  valendo** — o cancelamento por token passa pelo mesmo método de domínio, então cancelar
  um confirmado em cima da hora é bloqueado do mesmo jeito. A página avisa quando o prazo
  já passou.
- O token **não expira** por conta própria: vale enquanto o agendamento for cancelável.
- Ao cancelar, o **prestador é notificado** por email (cancelado pelo cliente).
- Cliente **com conta** não recebe esse link — ele cancela pelo painel.

---

## 4. Moderação (admin)

Um **administrador** modera prestadores e clientes. Ele é semeado no boot a partir de
`ADMIN_EMAIL`/`ADMIN_SENHA` (sem cadastro nem auto-registro), entra pela mesma tela de
login e cai no painel de moderação.

- **Banir** desativa o usuário (`ativo = false`): ele deixa de logar e as **sessões
  ativas dele são revogadas na hora** — sem isso, um banido com cookie válido manteria
  acesso até a sessão expirar. Um prestador banido também **some da vitrine** e **para de
  ofertar horários** — o link público dele passa a não mostrar slots, sem vazar o motivo.
  Um cliente banido também não agenda, nem logado nem como convidado. **Reversível** por
  reativar.
- **Histórico preservado** — banir não apaga nada; agendamentos existentes continuam.
- `ativo` (moderação, decisão do admin) é distinto de `aceita_agendamentos` (decisão do
  próprio prestador): um prestador ativo pode escolher não atender, mas um banido nunca
  oferta, independente da flag.

### Detalhe em leitura
O admin pode **abrir o detalhe** de um prestador ou de um cliente e ver **tudo o que
aquele usuário vê**, sem se passar por ele (nada de impersonation):

- **Prestador** — dados cadastrais (e-mail, duração do atendimento, intervalo de
  preparação, se aceita agendamentos) e a lista de **agendamentos recebidos**, com o
  **contato do cliente** (nome, e-mail, telefone) — a mesma visão do painel do prestador.
- **Cliente** — dados cadastrais (e-mail, telefone, se tem conta ou é convidado) e a lista
  de **agendamentos feitos**, com o nome do prestador.

É uma visão **somente leitura**: o admin não confirma, recusa nem cancela pelo detalhe —
para intervir no acesso do usuário existem banir/reativar. O detalhe reaproveita a mesma
listagem de agendamentos das pontas (com expiração lazy e nomes/contato já resolvidos).

---

## 5. Fuso horário

Todo o sistema assume um **fuso único fixo**: `America/Sao_Paulo`. Os horários são
interpretados e exibidos nesse fuso. O fuso deve ser centralizado em uma
constante/configuração única (não espalhar `time.Local` pelo código), para facilitar uma
eventual evolução para múltiplos fusos no futuro.

---

## 6. Notificações por email

O sistema envia email em cinco situações, sempre em português e melhor-esforço (uma falha
de envio nunca impede a operação que a disparou — só é registrada em log):

| Evento | Destinatário | Conteúdo |
|---|---|---|
| Novo pedido de horário | Prestador | Nome do cliente, data/hora, prazo para confirmar |
| Confirmação | Cliente | Nome do prestador, data/hora confirmada (+ link de cancelamento, se convidado) |
| Recusa | Cliente | Nome do prestador, data/hora recusada |
| Cancelamento | A outra parte (quem não cancelou) | Quem cancelou, data/hora |
| Lembrete (24h antes) | Cliente | Nome do prestador, data/hora do atendimento |

O envio é assíncrono (goroutine), para não atrasar a resposta HTTP da ação que o disparou.

### Recuperação de senha

Prestadores e clientes **com conta** (`TemConta() == true`) podem redefinir a senha por
email. Convidados (sem senha) não têm o que redefinir.

- O pedido gera um **token opaco de 256 bits**, do mesmo jeito que o token de sessão
  (`internal/pkg/token`): só o **hash SHA-256** é persistido, o token puro só existe no
  link do email.
- **TTL de 1h** a partir da emissão.
- **Uso único** — o token é apagado no momento em que é consumido (`DELETE ... RETURNING`
  atômico), então reusar o mesmo link uma segunda vez falha.
- Um novo pedido **invalida qualquer token anterior** do mesmo usuário — só o link mais
  recente funciona.
- **Resposta idêntica para email existente e inexistente** (sempre 204, mesmo corpo): o
  mesmo cuidado anti-enumeração já aplicado ao login (`internal/usecase/auth/auth.go`)
  vale aqui — a rota nunca revela quais emails estão cadastrados.
- Redefinir a senha **revoga todas as sessões ativas** do usuário (mesmo mecanismo do
  banimento pelo admin) — uma redefinição de senha é motivo razoável para exigir login de
  novo em todo dispositivo.
- A rota de solicitação tem o mesmo **teto de requisições por IP** dos logins, para
  mitigar tanto o esgotamento da cota diária do provedor de email quanto tentativas de
  adivinhar tokens por força bruta.

### Lembrete de agendamento

Um worker de fundo (`internal/adapter/worker/reminder.go`) checa periodicamente
(config: a cada 10 min) os agendamentos **CONFIRMADOs** cujo início está a **até 24h** de
distância e ainda não foram lembrados, e dispara o email de lembrete.

- **Nunca duplica**: marcar o lembrete como enviado é um `UPDATE` condicional
  (`WHERE lembrete_enviado_em IS NULL`) — funciona como uma reivindicação atômica, então
  mesmo sob concorrência só uma execução consegue enviar.
- Um agendamento confirmado a **menos de 24h** do início já recebe o lembrete no primeiro
  tick após a confirmação — redundante com o email de confirmação, mas inofensivo.

---

## Parâmetros fixados

Valores centralizados em `config/agendamento.go` e no domínio:

| Parâmetro | Descrição | Valor |
|---|---|---|
| TTL da pendência | Prazo até uma solicitação não confirmada expirar | 24h |
| Antecedência mínima de cancelamento | Prazo antes do início em que ainda se pode cancelar | 24h |
| Granularidade de minutos | Múltiplo mínimo dos horários dos blocos | 15 min |
| Duração do atendimento | Tamanho de cada slot ofertado (por prestador, editável) | sugestão inicial de 60 min |
| TTL do token de recuperação de senha | Prazo até o link de redefinição expirar | 1h |
| TTL do token de confirmação de cadastro | Prazo até o link de confirmação de cadastro expirar | 24h |
| Antecedência do lembrete de agendamento | Quanto antes do início o lembrete é disparado | 24h |
| Intervalo de checagem do worker de lembrete | Frequência do ticker que busca agendamentos a lembrar | 10 min |

---

## Mapa para o código

| Conceito | Local | Conteúdo |
|---|---|---|
| Prestador | `internal/domain/provider/` | `ativo`, `aceita_agendamentos`, `descanso_minutos`, `duracao_atendimento_minutos`, `HorariosPadrao` |
| Cliente | `internal/domain/client/` | conta ou convidado, `telefone` (contato), `ativo` (moderação) |
| Admin | `internal/domain/admin/` | moderador; semeado por env, sem cadastro |
| Disponibilidade | `internal/domain/availability/` | `TimeBlock`, `DateException` (bloqueio/extra), validação estrita |
| Slot | `internal/domain/slot/` | `Slot`, `Livres` (cálculo puro: duração + buffer, sobra descartada) |
| Agendamento | `internal/domain/appointment/` | `Appointment` + máquina de estados + expiração lazy |
| Token de recuperação de senha | `internal/domain/passwordreset/` | `Token`, uso único, TTL curto |
| Token de cancelamento (convidado) | `internal/domain/cancellation/` | `Token` por agendamento, gerado na confirmação, sem TTL |
| Cadastro pendente (verificação de email) | `internal/domain/signup/` | `Pendente` com nome/telefone/senha-hash, uso único, TTL 24h; converte convidado em conta preservando o ID |
| Login social (Google) | `internal/domain/socialidentity/`, `internal/domain/oauthstate/`, `internal/usecase/auth/login_social.go`, `internal/adapter/oauth/google.go` | vínculo `(provedor, sub)` → usuário; state/nonce de uso único (CSRF/replay); vincula por email verificado ou cria conta sem senha real; prestador novo nasce com `TelefonePendente` |
| Orquestração | `internal/usecase/{provider,availability,appointment,admin,auth}/` | preferências, slots, solicitar, confirmar, recusar, cancelar (por sessão ou por token), concluir, moderar, recuperar/redefinir senha, lembrar, login social |
| Notificações | `internal/adapter/email/`, `internal/adapter/worker/` | templates, transporte SMTP, worker de lembrete |
| Configuração | `config/agendamento.go`, `config/server.go`, `config/email.go`, `config/oauth.go` | fuso fixo, TTL, antecedência mínima, credenciais do admin, SMTP, credenciais Google OAuth |
| Persistência | `migrations/` | `providers`, `horarios_padrao`, `clients`, `admins`, `date_exceptions`, `appointments` (anti-overbooking por lock transacional no repositório), `sessions`, `password_reset_tokens`, `cancelamento_tokens`, `cadastros_pendentes`, `providers.telefone`, `social_identities`, `oauth_states` |
