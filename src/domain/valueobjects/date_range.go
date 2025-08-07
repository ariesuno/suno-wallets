package valueobjects

import (
	"fmt"
	"time"
)

// DateRange representa um intervalo de datas
type DateRange struct {
	startDate time.Time
	endDate   time.Time
}

// NewDateRange cria um novo value object DateRange
func NewDateRange(startDate, endDate time.Time) (*DateRange, error) {
	// Normalizar datas para UTC sem hora
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, time.UTC)

	if start.After(end) {
		return nil, NewValidationError("data de início não pode ser posterior à data de fim")
	}

	// Verificar se o intervalo não é muito longo (máximo 10 anos)
	maxDuration := 10 * 365 * 24 * time.Hour
	if end.Sub(start) > maxDuration {
		return nil, NewValidationError("intervalo de datas não pode exceder 10 anos")
	}

	return &DateRange{
		startDate: start,
		endDate:   end,
	}, nil
}

// MustNewDateRange cria DateRange ou entra em pânico se inválido (para testes)
func MustNewDateRange(startDate, endDate time.Time) *DateRange {
	dr, err := NewDateRange(startDate, endDate)
	if err != nil {
		panic(fmt.Sprintf("date range inválido: %v", err))
	}
	return dr
}

// Today cria um DateRange para o dia atual
func Today() *DateRange {
	now := time.Now()
	dr, _ := NewDateRange(now, now)
	return dr
}

// ThisWeek cria um DateRange para a semana atual
func ThisWeek() *DateRange {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 { // Domingo = 0, queremos segunda = 0
		weekday = 7
	}
	weekday-- // Ajustar para segunda = 0

	startOfWeek := now.AddDate(0, 0, -weekday)
	endOfWeek := startOfWeek.AddDate(0, 0, 6)

	dr, _ := NewDateRange(startOfWeek, endOfWeek)
	return dr
}

// ThisMonth cria um DateRange para o mês atual
func ThisMonth() *DateRange {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	dr, _ := NewDateRange(startOfMonth, endOfMonth)
	return dr
}

// ThisYear cria um DateRange para o ano atual
func ThisYear() *DateRange {
	now := time.Now()
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	endOfYear := time.Date(now.Year(), 12, 31, 23, 59, 59, 999999999, now.Location())

	dr, _ := NewDateRange(startOfYear, endOfYear)
	return dr
}

// Last30Days cria um DateRange para os últimos 30 dias
func Last30Days() *DateRange {
	now := time.Now()
	start := now.AddDate(0, 0, -30)

	dr, _ := NewDateRange(start, now)
	return dr
}

// Last90Days cria um DateRange para os últimos 90 dias
func Last90Days() *DateRange {
	now := time.Now()
	start := now.AddDate(0, 0, -90)

	dr, _ := NewDateRange(start, now)
	return dr
}

// StartDate retorna a data de início do intervalo
func (dr *DateRange) StartDate() time.Time {
	return dr.startDate
}

// EndDate retorna a data de fim do intervalo
func (dr *DateRange) EndDate() time.Time {
	return dr.endDate
}

// Duration retorna a duração do intervalo
func (dr *DateRange) Duration() time.Duration {
	return dr.endDate.Sub(dr.startDate)
}

// Days retorna o número de dias no intervalo
func (dr *DateRange) Days() int {
	return int(dr.Duration().Hours() / 24)
}

// Contains verifica se uma data está dentro do intervalo
func (dr *DateRange) Contains(date time.Time) bool {
	return (date.Equal(dr.startDate) || date.After(dr.startDate)) &&
		(date.Equal(dr.endDate) || date.Before(dr.endDate))
}

// ContainsRange verifica se outro intervalo está completamente dentro deste
func (dr *DateRange) ContainsRange(other *DateRange) bool {
	if other == nil {
		return false
	}

	return dr.Contains(other.startDate) && dr.Contains(other.endDate)
}

// Overlaps verifica se há sobreposição com outro intervalo
func (dr *DateRange) Overlaps(other *DateRange) bool {
	if other == nil {
		return false
	}

	return dr.startDate.Before(other.endDate) && dr.endDate.After(other.startDate)
}

// Union retorna a união de dois intervalos (menor início, maior fim)
func (dr *DateRange) Union(other *DateRange) (*DateRange, error) {
	if other == nil {
		return nil, NewValidationError("date range não pode ser nulo")
	}

	var start, end time.Time

	if dr.startDate.Before(other.startDate) {
		start = dr.startDate
	} else {
		start = other.startDate
	}

	if dr.endDate.After(other.endDate) {
		end = dr.endDate
	} else {
		end = other.endDate
	}

	return NewDateRange(start, end)
}

// Intersection retorna a interseção de dois intervalos
func (dr *DateRange) Intersection(other *DateRange) (*DateRange, error) {
	if other == nil {
		return nil, NewValidationError("date range não pode ser nulo")
	}

	if !dr.Overlaps(other) {
		return nil, NewValidationError("intervalos não se sobrepõem")
	}

	var start, end time.Time

	if dr.startDate.After(other.startDate) {
		start = dr.startDate
	} else {
		start = other.startDate
	}

	if dr.endDate.Before(other.endDate) {
		end = dr.endDate
	} else {
		end = other.endDate
	}

	return NewDateRange(start, end)
}

// Split divide o intervalo em subintervalos menores
func (dr *DateRange) Split(duration time.Duration) ([]*DateRange, error) {
	if duration <= 0 {
		return nil, NewValidationError("duração deve ser positiva")
	}

	var ranges []*DateRange
	current := dr.startDate

	for current.Before(dr.endDate) {
		end := current.Add(duration)
		if end.After(dr.endDate) {
			end = dr.endDate
		}

		subRange, err := NewDateRange(current, end)
		if err != nil {
			return nil, err
		}

		ranges = append(ranges, subRange)
		current = end.Add(time.Nanosecond) // Próximo período
	}

	return ranges, nil
}

// Equals verifica se dois intervalos são iguais
func (dr *DateRange) Equals(other *DateRange) bool {
	if other == nil {
		return false
	}

	return dr.startDate.Equal(other.startDate) && dr.endDate.Equal(other.endDate)
}

// String implementa fmt.Stringer
func (dr *DateRange) String() string {
	return fmt.Sprintf("%s a %s",
		dr.startDate.Format("2006-01-02"),
		dr.endDate.Format("2006-01-02"))
}

// FormattedBR retorna o intervalo formatado para o padrão brasileiro
func (dr *DateRange) FormattedBR() string {
	return fmt.Sprintf("%s a %s",
		dr.startDate.Format("02/01/2006"),
		dr.endDate.Format("02/01/2006"))
}

// IsInPast verifica se o intervalo está completamente no passado
func (dr *DateRange) IsInPast() bool {
	return dr.endDate.Before(time.Now())
}

// IsInFuture verifica se o intervalo está completamente no futuro
func (dr *DateRange) IsInFuture() bool {
	return dr.startDate.After(time.Now())
}

// IsCurrent verifica se o intervalo inclui o momento atual
func (dr *DateRange) IsCurrent() bool {
	return dr.Contains(time.Now())
}

// Validate valida o intervalo de datas
func (dr *DateRange) Validate() error {
	if dr.startDate.After(dr.endDate) {
		return NewValidationError("data de início não pode ser posterior à data de fim")
	}

	return nil
}

// MarshalJSON implementa json.Marshaler
func (dr *DateRange) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"start_date":"%s","end_date":"%s","formatted":"%s"}`,
		dr.startDate.Format(time.RFC3339),
		dr.endDate.Format(time.RFC3339),
		dr.String())), nil
}
