package normalize

import (
    "encoding/json"
    "strconv"
    "strings"
    "time"
)

// Comentários em pt-BR: helpers de parsing normalizado (compartilhados entre parsers)

func stringify(v any) string {
    if v == nil { return "" }
    b, _ := json.Marshal(v)
    s := strings.Trim(string(b), "\"")
    return s
}

func strptr(s string) *string { if s == "" { return nil }; return &s }
func optstr(s string) *string { if s == "" { return nil }; return &s }
func itoa(i int) string      { return strconv.Itoa(i) }
func parseYMD(s string) time.Time {
    t, _ := time.Parse("2006-01-02", s)
    return t
}


