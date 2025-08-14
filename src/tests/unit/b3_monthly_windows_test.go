package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateB3MonthlyWindows(t *testing.T) {
	// Setup timezone
	loc, err := time.LoadLocation("America/Sao_Paulo")
	require.NoError(t, err)

	// Cenário de referência: 2025-08-14 10:00:00 America/Sao_Paulo
	t.Run("Cenário referência - 2025-08-14", func(t *testing.T) {
		now := time.Date(2025, 8, 14, 10, 0, 0, 0, loc)
		minStart := time.Date(2019, 11, 1, 0, 0, 0, 0, loc)

		windows := GenerateB3MonthlyWindows(B3MonthlyWindowsConfig{
			MinStart: minStart,
			Now:      now,
			Location: loc,
		})

		// Deve gerar meses completos de 2019-11 até 2025-07 + janela corrente 2025-08
		expectedClosedMonths := (2025-2019)*12 + (7 - 11) + 1 // Nov 2019 até Jul 2025
		expectedTotal := expectedClosedMonths + 1             // + mês corrente

		assert.Equal(t, expectedTotal, len(windows), "Total de janelas incorreto")

		// Primeira janela: 2019-11-01 00:00:00 → 2019-11-30 23:59:59.999999999
		firstWindow := windows[0]
		assert.Equal(t, time.Date(2019, 11, 1, 0, 0, 0, 0, loc), firstWindow.Start)
		assert.Equal(t, time.Date(2019, 11, 30, 23, 59, 59, 999999999, loc), firstWindow.End)
		assert.False(t, firstWindow.IsCurrentMonth)

		// Última janela fechada: 2025-07-01 00:00:00 → 2025-07-31 23:59:59.999999999
		secondToLastWindow := windows[len(windows)-2]
		assert.Equal(t, time.Date(2025, 7, 1, 0, 0, 0, 0, loc), secondToLastWindow.Start)
		assert.Equal(t, time.Date(2025, 7, 31, 23, 59, 59, 999999999, loc), secondToLastWindow.End)
		assert.False(t, secondToLastWindow.IsCurrentMonth)

		// Janela corrente: 2025-08-01 00:00:00 → 2025-08-13 23:59:59.999999999
		lastWindow := windows[len(windows)-1]
		assert.Equal(t, time.Date(2025, 8, 1, 0, 0, 0, 0, loc), lastWindow.Start)
		assert.Equal(t, time.Date(2025, 8, 13, 23, 59, 59, 999999999, loc), lastWindow.End)
		assert.True(t, lastWindow.IsCurrentMonth)
	})

	// Dia 1 do mês: não deve gerar janela corrente
	t.Run("Dia 1 do mês - não gerar janela corrente", func(t *testing.T) {
		now := time.Date(2025, 8, 1, 9, 0, 0, 0, loc)
		minStart := time.Date(2019, 11, 1, 0, 0, 0, 0, loc)

		windows := GenerateB3MonthlyWindows(B3MonthlyWindowsConfig{
			MinStart: minStart,
			Now:      now,
			Location: loc,
		})

		// Não deve ter janela corrente - apenas até julho 2025
		expectedClosedMonths := (2025-2019)*12 + (7 - 11) + 1 // Nov 2019 até Jul 2025
		assert.Equal(t, expectedClosedMonths, len(windows))

		// Última janela deve ser julho 2025
		lastWindow := windows[len(windows)-1]
		assert.Equal(t, time.Date(2025, 7, 1, 0, 0, 0, 0, loc), lastWindow.Start)
		assert.Equal(t, time.Date(2025, 7, 31, 23, 59, 59, 999999999, loc), lastWindow.End)
		assert.False(t, lastWindow.IsCurrentMonth)
	})

	// Véspera do início: now = 2019-11-15
	t.Run("Véspera do início - 2019-11-15", func(t *testing.T) {
		now := time.Date(2019, 11, 15, 14, 30, 0, 0, loc)
		minStart := time.Date(2019, 11, 1, 0, 0, 0, 0, loc)

		windows := GenerateB3MonthlyWindows(B3MonthlyWindowsConfig{
			MinStart: minStart,
			Now:      now,
			Location: loc,
		})

		// Deve gerar apenas 1 janela: mês corrente 2019-11
		assert.Equal(t, 1, len(windows))

		window := windows[0]
		assert.Equal(t, time.Date(2019, 11, 1, 0, 0, 0, 0, loc), window.Start)
		assert.Equal(t, time.Date(2019, 11, 14, 23, 59, 59, 999999999, loc), window.End)
		assert.True(t, window.IsCurrentMonth)
	})

	// Fronteira de ano: now = 2023-01-10
	t.Run("Fronteira de ano - 2023-01-10", func(t *testing.T) {
		now := time.Date(2023, 1, 10, 16, 45, 0, 0, loc)
		minStart := time.Date(2019, 11, 1, 0, 0, 0, 0, loc)

		windows := GenerateB3MonthlyWindows(B3MonthlyWindowsConfig{
			MinStart: minStart,
			Now:      now,
			Location: loc,
		})

		// Meses completos: Nov 2019 até Dez 2022 + Jan 2023 corrente
		expectedClosedMonths := (2022-2019)*12 + (12 - 11) + 1 // Nov 2019 até Dez 2022
		expectedTotal := expectedClosedMonths + 1              // + Jan 2023 corrente

		assert.Equal(t, expectedTotal, len(windows))

		// Última janela fechada: 2022-12-01 → 2022-12-31 23:59:59.999999999
		secondToLastWindow := windows[len(windows)-2]
		assert.Equal(t, time.Date(2022, 12, 1, 0, 0, 0, 0, loc), secondToLastWindow.Start)
		assert.Equal(t, time.Date(2022, 12, 31, 23, 59, 59, 999999999, loc), secondToLastWindow.End)
		assert.False(t, secondToLastWindow.IsCurrentMonth)

		// Janela corrente: 2023-01-01 → 2023-01-09 23:59:59.999999999
		lastWindow := windows[len(windows)-1]
		assert.Equal(t, time.Date(2023, 1, 1, 0, 0, 0, 0, loc), lastWindow.Start)
		assert.Equal(t, time.Date(2023, 1, 9, 23, 59, 59, 999999999, loc), lastWindow.End)
		assert.True(t, lastWindow.IsCurrentMonth)
	})

	// Teste de DST/horário de verão (robustez)
	t.Run("DST - horário de verão", func(t *testing.T) {
		// Período que costumava ter mudança de horário no Brasil (outubro)
		// Mesmo sem DST oficial, o teste valida robustez
		now := time.Date(2023, 10, 15, 14, 0, 0, 0, loc)
		minStart := time.Date(2023, 10, 1, 0, 0, 0, 0, loc)

		windows := GenerateB3MonthlyWindows(B3MonthlyWindowsConfig{
			MinStart: minStart,
			Now:      now,
			Location: loc,
		})

		// Deve gerar apenas 1 janela corrente
		assert.Equal(t, 1, len(windows))

		window := windows[0]
		assert.Equal(t, time.Date(2023, 10, 1, 0, 0, 0, 0, loc), window.Start)
		assert.Equal(t, time.Date(2023, 10, 14, 23, 59, 59, 999999999, loc), window.End)
		assert.True(t, window.IsCurrentMonth)

		// Verificar que timezone foi preservada
		assert.Equal(t, loc, window.Start.Location())
		assert.Equal(t, loc, window.End.Location())
	})

	// Teste com minStart customizado
	t.Run("MinStart customizado", func(t *testing.T) {
		now := time.Date(2023, 3, 10, 12, 0, 0, 0, loc)
		minStart := time.Date(2023, 1, 15, 8, 30, 0, 0, loc) // Meio do janeiro

		windows := GenerateB3MonthlyWindows(B3MonthlyWindowsConfig{
			MinStart: minStart,
			Now:      now,
			Location: loc,
		})

		// Jan (parcial), Feb (completo), Mar (corrente)
		assert.Equal(t, 3, len(windows))

		// Primeira janela: começa no minStart, termina fim de janeiro
		firstWindow := windows[0]
		assert.Equal(t, minStart, firstWindow.Start)
		assert.Equal(t, time.Date(2023, 1, 31, 23, 59, 59, 999999999, loc), firstWindow.End)
		assert.False(t, firstWindow.IsCurrentMonth)

		// Segunda janela: fevereiro completo
		secondWindow := windows[1]
		assert.Equal(t, time.Date(2023, 2, 1, 0, 0, 0, 0, loc), secondWindow.Start)
		assert.Equal(t, time.Date(2023, 2, 28, 23, 59, 59, 999999999, loc), secondWindow.End)
		assert.False(t, secondWindow.IsCurrentMonth)

		// Terceira janela: março corrente
		thirdWindow := windows[2]
		assert.Equal(t, time.Date(2023, 3, 1, 0, 0, 0, 0, loc), thirdWindow.Start)
		assert.Equal(t, time.Date(2023, 3, 9, 23, 59, 59, 999999999, loc), thirdWindow.End)
		assert.True(t, thirdWindow.IsCurrentMonth)
	})

	// Teste de defaults (quando config vazio)
	t.Run("Defaults", func(t *testing.T) {
		// Passar config vazio para testar defaults
		windows := GenerateB3MonthlyWindows(B3MonthlyWindowsConfig{})

		// Deve usar defaults e gerar pelo menos algumas janelas
		assert.Greater(t, len(windows), 0, "Deve gerar pelo menos uma janela com defaults")

		// Primeira janela deve começar em 2019-11-01 (default minStart)
		firstWindow := windows[0]
		expectedStart := time.Date(2019, 11, 1, 0, 0, 0, 0, firstWindow.Start.Location())
		assert.Equal(t, expectedStart, firstWindow.Start)
	})
}

// Teste de funções auxiliares
func TestB3HelperFunctions(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	require.NoError(t, err)

	t.Run("firstOfMonthB3", func(t *testing.T) {
		input := time.Date(2023, 5, 15, 14, 30, 45, 123456789, loc)
		expected := time.Date(2023, 5, 1, 0, 0, 0, 0, loc)
		assert.Equal(t, expected, firstOfMonthB3(input))
	})

	t.Run("endOfDayB3", func(t *testing.T) {
		input := time.Date(2023, 5, 15, 14, 30, 45, 123456789, loc)
		expected := time.Date(2023, 5, 15, 23, 59, 59, 999999999, loc)
		assert.Equal(t, expected, endOfDayB3(input))
	})

	t.Run("ternary", func(t *testing.T) {
		assert.Equal(t, "yes", ternary(true, "yes", "no"))
		assert.Equal(t, "no", ternary(false, "yes", "no"))
		assert.Equal(t, 42, ternary(true, 42, 24))
		assert.Equal(t, 24, ternary(false, 42, 24))
	})
}
