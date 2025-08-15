package normalize

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// Comentários em pt-BR: helpers de parsing normalizado (compartilhados entre parsers)

func stringify(v any) string {
	if v == nil {
		return ""
	}

	// Converte para string baseado no tipo
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		// Fallback para JSON
		b, _ := json.Marshal(v)
		s := strings.Trim(string(b), "\"")
		return s
	}
}

func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func optstr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func itoa(i int) string     { return strconv.Itoa(i) }
func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', 2, 64) }
func parseYMD(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}
