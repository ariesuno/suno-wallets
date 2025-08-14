package main

import (
	"fmt"
	"time"

	"suno-wallets/src/shared"
)

func main() {
	// Teste cenário de referência: 2025-08-14 10:00:00 America/Sao_Paulo
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		fmt.Printf("Erro ao carregar timezone: %v\n", err)
		return
	}

	now := time.Date(2025, 8, 14, 10, 0, 0, 0, loc)
	minStart := time.Date(2019, 11, 1, 0, 0, 0, 0, loc)

	fmt.Printf("Testando geração de janelas B3\n")
	fmt.Printf("Now: %s\n", now.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("MinStart: %s\n", minStart.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("==================================\n")

	windows := shared.GenerateB3MonthlyWindows(minStart, now)

	fmt.Printf("Total de janelas geradas: %d\n", len(windows))

	// Mostrar primeiras 5 janelas
	fmt.Printf("\nPrimeiras 5 janelas:\n")
	for i, w := range windows {
		if i >= 5 {
			break
		}
		fmt.Printf("%d. %s → %s (current: %v)\n",
			i+1,
			w.Start.Format("2006-01-02 15:04:05"),
			w.End.Format("2006-01-02 15:04:05"),
			w.IsCurrentMonth)
	}

	// Mostrar últimas 5 janelas
	fmt.Printf("\nÚltimas 5 janelas:\n")
	start := len(windows) - 5
	if start < 0 {
		start = 0
	}
	for i := start; i < len(windows); i++ {
		w := windows[i]
		fmt.Printf("%d. %s → %s (current: %v)\n",
			i+1,
			w.Start.Format("2006-01-02 15:04:05"),
			w.End.Format("2006-01-02 15:04:05"),
			w.IsCurrentMonth)
	}

	// Validar última janela (mês corrente)
	if len(windows) > 0 {
		lastWindow := windows[len(windows)-1]
		if lastWindow.IsCurrentMonth {
			expectedStart := time.Date(2025, 8, 1, 0, 0, 0, 0, loc)
			expectedEnd := time.Date(2025, 8, 13, 23, 59, 59, 999999999, loc)

			fmt.Printf("\nValidação janela corrente:\n")
			fmt.Printf("Esperado - Start: %s\n", expectedStart.Format("2006-01-02 15:04:05"))
			fmt.Printf("Real     - Start: %s\n", lastWindow.Start.Format("2006-01-02 15:04:05"))
			fmt.Printf("Esperado - End:   %s\n", expectedEnd.Format("2006-01-02 15:04:05"))
			fmt.Printf("Real     - End:   %s\n", lastWindow.End.Format("2006-01-02 15:04:05"))

			if lastWindow.Start.Equal(expectedStart) && lastWindow.End.Equal(expectedEnd) {
				fmt.Printf("✅ Janela corrente VÁLIDA\n")
			} else {
				fmt.Printf("❌ Janela corrente INVÁLIDA\n")
			}
		}
	}

	// Teste do dia 1 (não deve gerar janela corrente)
	fmt.Printf("\n==================================\n")
	fmt.Printf("Teste dia 1 do mês (não deve gerar janela corrente):\n")
	now1 := time.Date(2025, 8, 1, 9, 0, 0, 0, loc)
	windows1 := shared.GenerateB3MonthlyWindows(minStart, now1)

	hasCurrentMonth1 := len(windows1) > 0 && windows1[len(windows1)-1].IsCurrentMonth
	fmt.Printf("Total janelas: %d\n", len(windows1))
	fmt.Printf("Tem janela corrente: %v\n", hasCurrentMonth1)

	if !hasCurrentMonth1 {
		fmt.Printf("✅ Dia 1 CORRETO - sem janela corrente\n")
	} else {
		fmt.Printf("❌ Dia 1 INCORRETO - gerou janela corrente\n")
	}
}
