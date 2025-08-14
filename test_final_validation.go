package main

import (
	"fmt"
	"os"
	"time"

	"suno-wallets/src/shared"
)

func main() {
	fmt.Println("=== VALIDAÇÃO FINAL DAS JANELAS B3 ===")

	// Cenário de aceitação: 2025-08-14
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		fmt.Printf("❌ Erro ao carregar timezone: %v\n", err)
		os.Exit(1)
	}

	now := time.Date(2025, 8, 14, 10, 0, 0, 0, loc)
	minStart := time.Date(2019, 11, 1, 0, 0, 0, 0, loc)

	fmt.Printf("Now: %s\n", now.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("MinStart: %s\n", minStart.Format("2006-01-02 15:04:05 MST"))

	windows := shared.GenerateB3MonthlyWindows(minStart, now)

	// Validações de aceitação
	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
	}{
		{"Total de janelas", 70, len(windows)},
		{"Primeira janela start", "2019-11-01 00:00:00", windows[0].Start.Format("2006-01-02 15:04:05")},
		{"Primeira janela end", "2019-11-30 23:59:59", windows[0].End.Format("2006-01-02 15:04:05")},
		{"Primeira janela é atual", false, windows[0].IsCurrentMonth},
	}

	// Última janela deve ser atual (agosto 2025)
	lastWindow := windows[len(windows)-1]
	tests = append(tests, []struct {
		name     string
		expected interface{}
		actual   interface{}
	}{
		{"Última janela start", "2025-08-01 00:00:00", lastWindow.Start.Format("2006-01-02 15:04:05")},
		{"Última janela end", "2025-08-13 23:59:59", lastWindow.End.Format("2006-01-02 15:04:05")},
		{"Última janela é atual", true, lastWindow.IsCurrentMonth},
	}...)

	// Executar testes
	passed := 0
	for _, test := range tests {
		if fmt.Sprintf("%v", test.expected) == fmt.Sprintf("%v", test.actual) {
			fmt.Printf("✅ %s: %v\n", test.name, test.actual)
			passed++
		} else {
			fmt.Printf("❌ %s: esperado %v, obtido %v\n", test.name, test.expected, test.actual)
		}
	}

	// Teste do dia 1 (não deve gerar janela corrente)
	fmt.Println("\n--- Teste Dia 1 do Mês ---")
	now1 := time.Date(2025, 8, 1, 9, 0, 0, 0, loc)
	windows1 := shared.GenerateB3MonthlyWindows(minStart, now1)

	hasCurrentMonth1 := len(windows1) > 0 && windows1[len(windows1)-1].IsCurrentMonth
	if !hasCurrentMonth1 && len(windows1) == 69 {
		fmt.Printf("✅ Dia 1 correto: %d janelas, sem janela corrente\n", len(windows1))
		passed++
	} else {
		fmt.Printf("❌ Dia 1 incorreto: %d janelas, janela corrente: %v\n", len(windows1), hasCurrentMonth1)
	}

	// Teste DST
	fmt.Println("\n--- Teste Robustez DST ---")
	nowDST := time.Date(2023, 10, 15, 14, 0, 0, 0, loc)
	minStartDST := time.Date(2023, 10, 1, 0, 0, 0, 0, loc)
	windowsDST := shared.GenerateB3MonthlyWindows(minStartDST, nowDST)

	if len(windowsDST) == 1 && windowsDST[0].IsCurrentMonth &&
		windowsDST[0].Start.Location() == loc && windowsDST[0].End.Location() == loc {
		fmt.Printf("✅ DST correto: timezone preservada\n")
		passed++
	} else {
		fmt.Printf("❌ DST incorreto\n")
	}

	// Resultado final
	total := len(tests) + 2 // +2 para dia 1 e DST
	fmt.Printf("\n=== RESULTADO FINAL ===\n")
	fmt.Printf("Testes aprovados: %d/%d\n", passed, total)

	if passed == total {
		fmt.Printf("🎉 TODAS AS VALIDAÇÕES PASSARAM!\n")
		fmt.Printf("✅ Implementação das janelas B3 está CORRETA\n")
		fmt.Printf("✅ Timezone America/Sao_Paulo funcional\n")
		fmt.Printf("✅ Lógica de mês corrente implementada\n")
		fmt.Printf("✅ Regra do dia 1 respeitada\n")
		fmt.Printf("✅ Robustez DST validada\n")
		os.Exit(0)
	} else {
		fmt.Printf("⚠️  Algumas validações falharam\n")
		os.Exit(1)
	}
}
