-- Marca quando o lembrete por email de um agendamento confirmado foi
-- enviado. NULL significa "ainda não enviado".
ALTER TABLE appointments ADD COLUMN lembrete_enviado_em TIMESTAMPTZ;
