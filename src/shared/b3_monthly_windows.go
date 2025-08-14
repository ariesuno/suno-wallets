package shared

import (
	"time"

	"suno-wallets/src/shared/helpers"
)

// B3Window representa uma janela temporal para busca B3
type B3Window struct {
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	IsCurrentMonth bool      `json:"isCurrentMonth"`
}

// B3MonthlyWindowsConfig configuração para geração de janelas
type B3MonthlyWindowsConfig struct {
	MinStart time.Time      // Data mínima de início (default: 2019-11-01 00:00:00 America/Sao_Paulo)
	Now      time.Time      // "Agora" injetável para testes (default: time.Now().In(loc))
	Location *time.Location // Timezone (default: America/Sao_Paulo)
}

// GenerateB3MonthlyWindows gera janelas mensais para busca B3 seguindo as regras:
// 1. Meses fechados: [início do mês, fim do mês] completos de minStart até último mês completo
// 2. Mês corrente: [1º dia do mês às 00:00:00, D-1 às 23:59:59.999999] em America/Sao_Paulo
// 3. Se hoje for dia 1, não criar janela do mês corrente (pois D-1 não existe)
// 4. Nunca projetar datas futuras
func GenerateB3MonthlyWindows(config B3MonthlyWindowsConfig) []B3Window {
	// Defaults
	loc := config.Location
	if loc == nil {
		var err error
		loc, err = time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			helpers.LogError("Falha ao carregar timezone America/Sao_Paulo, usando UTC", err, nil)
			loc = time.UTC
		}
	}

	now := config.Now
	if now.IsZero() {
		now = time.Now().In(loc)
	} else {
		now = now.In(loc)
	}

	minStart := config.MinStart
	if minStart.IsZero() {
		minStart = time.Date(2019, 11, 1, 0, 0, 0, 0, loc)
	} else {
		minStart = minStart.In(loc)
	}

	helpers.LogInfo("Gerando janelas B3", map[string]interface{}{
		"minStart": minStart.Format("2006-01-02 15:04:05 MST"),
		"now":      now.Format("2006-01-02 15:04:05 MST"),
		"timezone": loc.String(),
	})

	var windows []B3Window

	// 1. Gerar meses fechados completos até o último mês anterior ao mês atual
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	lastCompleteMonth := currentMonth.AddDate(0, -1, 0)

	// Iterar de minStart até lastCompleteMonth
	cursor := time.Date(minStart.Year(), minStart.Month(), 1, 0, 0, 0, 0, loc)
	for !cursor.After(lastCompleteMonth) {
		start := cursor
		if start.Before(minStart) {
			start = minStart
		}

		// Fim do mês: último dia às 23:59:59.999999
		end := cursor.AddDate(0, 1, 0).Add(-time.Nanosecond)

		windows = append(windows, B3Window{
			Start:          start,
			End:            end,
			IsCurrentMonth: false,
		})

		cursor = cursor.AddDate(0, 1, 0)
	}

	// 2. Tentar gerar janela do mês corrente
	// Regra: só criar se D-1 existir (ou seja, não estamos no dia 1)
	if now.Day() > 1 {
		start := currentMonth // 1º dia do mês às 00:00:00

		// D-1 às 23:59:59.999999
		yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 23, 59, 59, 999999999, loc)

		// Validação: só criar se end >= start (sempre verdade aqui, mas boa prática)
		if !yesterday.Before(start) {
			windows = append(windows, B3Window{
				Start:          start,
				End:            yesterday,
				IsCurrentMonth: true,
			})
		}
	}

	helpers.LogInfo("Janelas B3 geradas", map[string]interface{}{
		"totalWindows":    len(windows),
		"closedMonths":    len(windows) - ternary(len(windows) > 0 && windows[len(windows)-1].IsCurrentMonth, 1, 0),
		"hasCurrentMonth": len(windows) > 0 && windows[len(windows)-1].IsCurrentMonth,
		"firstWindow":     ternary(len(windows) > 0, windows[0].Start.Format("2006-01-02 15:04:05"), "none"),
		"lastWindow":      ternary(len(windows) > 0, windows[len(windows)-1].End.Format("2006-01-02 15:04:05"), "none"),
	})

	return windows
}

// firstOfMonthB3 retorna o primeiro dia do mês às 00:00:00 na timezone especificada
func firstOfMonthB3(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// endOfDayB3 retorna o final do dia (23:59:59.999999999) na timezone especificada
func endOfDayB3(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// ternary função ternária genérica para B3
func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
