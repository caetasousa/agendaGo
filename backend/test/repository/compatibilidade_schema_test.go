//go:build integration

// Guarda automática da compatibilidade N/N-1 entre código e schema.
//
// Rollback de código só é rápido enquanto a versão ANTERIOR da aplicação
// continua funcionando contra o schema JÁ MIGRADO — o Flyway é forward-only e
// o banco não volta junto com a imagem. Uma migration que aperta (coluna nova
// obrigatória, coluna removida, NOT NULL em coluna que era nula) quebra essa
// propriedade em silêncio: tudo passa no CI, e o defeito só aparece no dia do
// rollback, que é o pior dia possível para descobri-lo.
//
// Este teste transforma essa propriedade em falha de build.
package repository_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// versaoBaseCompatibilidade é a última migration ANTERIOR à adoção da política
// de dois deploys. Até ela, colunas obrigatórias entraram de uma vez, o que era
// consistente com o banco descartável daquele momento — e o banco foi recriado.
// Da próxima em diante, toda migration é comparada com a anterior.
const versaoBaseCompatibilidade = 19

// colunaSchema é o que importa para responder "o código anterior ainda
// escreve nesta tabela?": se a coluna aceita ausência no INSERT.
type colunaSchema struct {
	anulavel   bool
	temDefault bool
}

// aceitaAusencia informa se um INSERT que ignora a coluna funciona. É a
// pergunta exata que o código da versão anterior faz ao banco.
func (c colunaSchema) aceitaAusencia() bool {
	return c.anulavel || c.temDefault
}

func snapshotColunas(t *testing.T, pool *pgxpool.Pool) map[string]colunaSchema {
	t.Helper()
	linhas, err := pool.Query(context.Background(), `
		SELECT table_name, column_name, is_nullable, column_default IS NOT NULL
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name <> 'flyway_schema_history'`)
	if err != nil {
		t.Fatalf("ler colunas do schema: %v", err)
	}
	defer linhas.Close()

	schema := make(map[string]colunaSchema)
	for linhas.Next() {
		var tabela, coluna, anulavel string
		var temDefault bool
		if err := linhas.Scan(&tabela, &coluna, &anulavel, &temDefault); err != nil {
			t.Fatalf("escanear coluna: %v", err)
		}
		schema[tabela+"."+coluna] = colunaSchema{anulavel: anulavel == "YES", temDefault: temDefault}
	}
	if err := linhas.Err(); err != nil {
		t.Fatalf("percorrer colunas: %v", err)
	}
	return schema
}

// incompatibilidades devolve, em texto, tudo que impediria a versão anterior do
// código de continuar escrevendo no banco já migrado.
func incompatibilidades(antes, depois map[string]colunaSchema) []string {
	var achados []string

	for nome, novo := range depois {
		if _, existia := antes[nome]; existia {
			continue
		}
		// Coluna nova e obrigatória: o INSERT do código anterior não a
		// menciona, e o banco recusa a linha inteira.
		if !novo.aceitaAusencia() {
			achados = append(achados, "coluna nova obrigatória: "+nome)
		}
	}

	for nome, velho := range antes {
		novo, aindaExiste := depois[nome]
		if !aindaExiste {
			// O código anterior ainda lê e escreve esta coluna.
			achados = append(achados, "coluna removida: "+nome)
			continue
		}
		if velho.aceitaAusencia() && !novo.aceitaAusencia() {
			achados = append(achados, "coluna apertada para obrigatória: "+nome)
		}
	}

	return achados
}

func TestCompatibilidadeDeSchema(t *testing.T) {
	base := snapshotColunas(t, novoPoolAte(t, versaoBaseCompatibilidade))

	t.Run("migrations novas mantêm a versão anterior do código funcionando", func(t *testing.T) {
		// Sem versaoMax: aplica tudo que existe hoje. Enquanto não houver
		// migration depois da base, não há o que comparar — e é assim que deve
		// ser: o teste vira guarda no instante em que a primeira aparecer.
		atual := snapshotColunas(t, novoPool(t))

		for _, achado := range incompatibilidades(base, atual) {
			t.Errorf(`%s

Uma migration depois da V%d quebrou o rollback: a versão anterior da aplicação
deixa de conseguir escrever no banco já migrado. Divida em dois deploys —
primeiro a coluna anulável, depois (com o código novo estável) o SET NOT NULL.
Ver a seção de expand/contract em docs/tecnologias.md.`, achado, versaoBaseCompatibilidade)
		}
	})

	// Os dois casos abaixo provam que o verificador ACUSA de verdade. Sem eles,
	// enquanto não houver migration nova, o teste acima passaria mesmo estando
	// quebrado, e ninguém saberia.
	t.Run("acusa coluna nova obrigatória", func(t *testing.T) {
		pool := novoPool(t)
		antes := snapshotColunas(t, pool)
		_, err := pool.Exec(context.Background(),
			`ALTER TABLE providers ADD COLUMN aperto_de_teste BOOLEAN NOT NULL`)
		if err != nil {
			t.Fatalf("aplicar mudança sintética: %v", err)
		}

		achados := incompatibilidades(antes, snapshotColunas(t, pool))
		if len(achados) != 1 || achados[0] != "coluna nova obrigatória: providers.aperto_de_teste" {
			t.Errorf("esperava acusar a coluna obrigatória nova, got: %v", achados)
		}
	})

	t.Run("aceita coluna nova anulável e acusa o aperto seguinte", func(t *testing.T) {
		pool := novoPool(t)
		antes := snapshotColunas(t, pool)

		// Passo expand: anulável, o código anterior segue inserindo.
		if _, err := pool.Exec(context.Background(),
			`ALTER TABLE providers ADD COLUMN expand_de_teste BOOLEAN`); err != nil {
			t.Fatalf("aplicar expand sintético: %v", err)
		}
		expandido := snapshotColunas(t, pool)
		if achados := incompatibilidades(antes, expandido); len(achados) != 0 {
			t.Errorf("coluna anulável não deveria acusar nada, got: %v", achados)
		}

		// Passo contract: só é seguro depois que o código novo estabilizou.
		if _, err := pool.Exec(context.Background(),
			`ALTER TABLE providers ALTER COLUMN expand_de_teste SET NOT NULL`); err != nil {
			t.Fatalf("aplicar contract sintético: %v", err)
		}
		achados := incompatibilidades(expandido, snapshotColunas(t, pool))
		if len(achados) != 1 || achados[0] != "coluna apertada para obrigatória: providers.expand_de_teste" {
			t.Errorf("esperava acusar o aperto, got: %v", achados)
		}
	})

	t.Run("acusa coluna removida", func(t *testing.T) {
		pool := novoPool(t)
		antes := snapshotColunas(t, pool)
		if _, err := pool.Exec(context.Background(),
			`ALTER TABLE providers DROP COLUMN descanso_minutos`); err != nil {
			t.Fatalf("aplicar remoção sintética: %v", err)
		}

		achados := incompatibilidades(antes, snapshotColunas(t, pool))
		if len(achados) != 1 || achados[0] != "coluna removida: providers.descanso_minutos" {
			t.Errorf("esperava acusar a coluna removida, got: %v", achados)
		}
	})
}
