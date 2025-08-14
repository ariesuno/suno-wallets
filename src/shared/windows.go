package shared

import (
	"os"
	"time"

	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"
)

// B3Window representa uma janela temporal para busca B3
type B3Window struct {
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	IsCurrentMonth bool      `json:"isCurrentMonth"`
}

// GenerateB3MonthlyWindows gera janelas mensais para busca B3 seguindo as regras:
// 1. Meses fechados: [início do mês, fim do mês] completos de minStart até último mês completo
// 2. Mês corrente: [1º dia do mês às 00:00:00, D-1 às 23:59:59.999999] em America/Sao_Paulo
// 3. Se hoje for dia 1, não criar janela do mês corrente (pois D-1 não existe)
// 4. Nunca projetar datas futuras
func GenerateB3MonthlyWindows(minStart, now time.Time) []B3Window {
	// Carregar timezone America/Sao_Paulo
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		helpers.LogError("Falha ao carregar timezone America/Sao_Paulo, usando UTC", err, nil)
		loc = time.UTC
	}

	// Configurar defaults se necessário
	if now.IsZero() {
		now = time.Now().In(loc)
	} else {
		now = now.In(loc)
	}

	if minStart.IsZero() {
		// Usar env var B3_MIN_START_DATE ou default
		minStartStr := os.Getenv("B3_MIN_START_DATE")
		if minStartStr != "" {
			if parsed, err := time.ParseInLocation("2006-01-02", minStartStr, loc); err == nil {
				minStart = parsed
			} else {
				helpers.LogError("B3_MIN_START_DATE inválido, usando default", err, map[string]interface{}{
					"value": minStartStr,
				})
				minStart = time.Date(2019, 11, 1, 0, 0, 0, 0, loc)
			}
		} else {
			minStart = time.Date(2019, 11, 1, 0, 0, 0, 0, loc)
		}
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

		// Fim do mês: último dia às 23:59:59.999999999
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

		// D-1 às 23:59:59.999999999
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

	// Contar janelas por tipo para métricas
	closedMonths := len(windows) - ternaryInt(len(windows) > 0 && windows[len(windows)-1].IsCurrentMonth, 1, 0)
	hasCurrentMonth := len(windows) > 0 && windows[len(windows)-1].IsCurrentMonth

	helpers.LogInfo("Janelas B3 geradas", map[string]interface{}{
		"totalWindows":    len(windows),
		"closedMonths":    closedMonths,
		"hasCurrentMonth": hasCurrentMonth,
		"firstWindow":     ternaryStr(len(windows) > 0, windows[0].Start.Format("2006-01-02 15:04:05"), "none"),
		"lastWindow":      ternaryStr(len(windows) > 0, windows[len(windows)-1].End.Format("2006-01-02 15:04:05"), "none"),
	})

	// Registrar métricas Prometheus
	observability.IncB3WindowsGenerated("closed_month", closedMonths)
	if hasCurrentMonth {
		observability.IncB3WindowsGenerated("current_month", 1)
	}

	return windows
}

// ConvertToTimeWindows converte B3Window para [][2]time.Time (compatibilidade com código existente)
func ConvertToTimeWindows(windows []B3Window) [][2]time.Time {
	result := make([][2]time.Time, len(windows))
	for i, w := range windows {
		result[i] = [2]time.Time{w.Start, w.End}
	}
	return result
}

// Funções auxiliares para evitar imports desnecessários
func ternaryInt(cond bool, a, b int) int {
	if cond {
		return a
	}
	return b
}

func ternaryStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
