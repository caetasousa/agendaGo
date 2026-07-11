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

### Fatiamento por serviço + buffer
Cada bloco é fatiado em slots conforme a **duração do serviço escolhido** somada ao
**buffer** do prestador:

- **Buffer configurável por prestador** (`buffer_minutos`: 0, 10, 15…) — intervalo de
  preparação/limpeza entre atendimentos. O próximo slot só abre após
  `duração_serviço + buffer`.
- **Sobra descartada** — só vira slot o intervalo que cabe o **serviço inteiro (+ buffer)**
  dentro do bloco. O tempo que sobra no fim do bloco é ignorado; um atendimento nunca
  "vaza" para fora do bloco.

---

## 3. Agendamento (reserva)

### Reserva ao solicitar + expiração (anti-overbooking)
Quando o cliente solicita um horário, o agendamento já **ocupa o intervalo** (reserva
pessimista) com um prazo de expiração (`expira_em = agora + TTL`). Outros clientes não
conseguem solicitar o mesmo intervalo.

- Se o prestador não confirmar dentro do TTL, a pendência vira **EXPIRADO** e o intervalo
  volta a ficar livre.
- O conflito é barrado também no banco, por uma constraint de exclusão sobre
  `(provider_id, intervalo)` para os status que ocupam horário, dentro de transação.

> Isso elimina a janela de overbooking entre "solicitar" e "confirmar".

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
**Cliente e prestador** podem cancelar um agendamento `CONFIRMADO`, respeitando uma
**antecedência mínima** (config; ex.: até 24h antes do início). Cancelamentos dentro da
janela mínima são bloqueados (tratamento de penalidade fica fora de escopo por enquanto).
Ao cancelar, o intervalo volta a ficar livre.

---

## 4. Fuso horário

Todo o sistema assume um **fuso único fixo**: `America/Sao_Paulo`. Os horários são
interpretados e exibidos nesse fuso. O fuso deve ser centralizado em uma
constante/configuração única (não espalhar `time.Local` pelo código), para facilitar uma
eventual evolução para múltiplos fusos no futuro.

---

## Parâmetros a calibrar na implementação

Três valores numéricos ainda precisam ser fixados (sugestões iniciais entre parênteses):

| Parâmetro | Descrição | Sugestão inicial |
|---|---|---|
| TTL da pendência | Prazo até uma solicitação não confirmada expirar | 24h |
| Antecedência mínima de cancelamento | Prazo antes do início em que ainda se pode cancelar | 24h |
| Granularidade de minutos | Múltiplo mínimo dos horários dos blocos | a definir (ex.: 5 ou 15 min) |

---

## Mapa para o código

| Conceito | Local | Conteúdo |
|---|---|---|
| Prestador | `internal/domain/provider/` | `aceita_agendamentos`, `descanso_minutos`, `HorariosPadrao` (expediente configurável) |
| Disponibilidade | `internal/domain/availability/` | `TimeBlock`, `DateException` (bloqueio/extra), validação estrita |
| Slot | `internal/domain/slot/` | `Slot`, `SlotsLivres` (cálculo puro: duração + buffer, sobra descartada) |
| Agendamento | `internal/domain/appointment/` | `Appointment` + máquina de estados |
| Orquestração | `internal/usecase/{provider,availability,appointment}/` | atualizar preferências, solicitar, confirmar, recusar, cancelar, expirar |
| Configuração | `config/` | fuso fixo, TTL, antecedência mínima |
| Persistência | `migrations/` | `providers`, `horarios_padrao`, `date_exceptions`, `appointments` (exclusion constraint anti-overbooking) |
