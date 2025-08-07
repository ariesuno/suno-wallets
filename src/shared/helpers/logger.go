package helpers

import (
	"encoding/json"
	"log"
	"strings"
	"time"
)

// LogLevel representa o nível de log
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogEntry representa uma entrada de log estruturada
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

var (
	currentLogLevel LogLevel = LogLevelInfo
	logFormat       string   = "json"
)

// InitLogger inicializa o sistema de logging
func InitLogger(level, format string) {
	currentLogLevel = LogLevel(level)
	logFormat = format
}

// LogDebug registra uma mensagem de debug
func LogDebug(message string, fields map[string]interface{}) {
	if shouldLog(LogLevelDebug) {
		writeLog(LogLevelDebug, message, fields)
	}
}

// LogInfo registra uma mensagem informativa
func LogInfo(message string, fields map[string]interface{}) {
	if shouldLog(LogLevelInfo) {
		writeLog(LogLevelInfo, message, fields)
	}
}

// LogWarn registra uma mensagem de aviso
func LogWarn(message string, fields map[string]interface{}) {
	if shouldLog(LogLevelWarn) {
		writeLog(LogLevelWarn, message, fields)
	}
}

// LogError registra uma mensagem de erro
func LogError(message string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	if shouldLog(LogLevelError) {
		writeLog(LogLevelError, message, fields)
	}
}

// LogAudit registra uma entrada de auditoria com campos obrigatórios
func LogAudit(action, tenantID, userID string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}

	// Campos obrigatórios para auditoria
	fields["action"] = action
	fields["tenant_id"] = tenantID
	fields["user_id"] = userID
	fields["audit"] = true

	LogInfo("Ação de auditoria registrada", fields)
}

// shouldLog verifica se o nível de log deve ser registrado
func shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
	}

	currentLevel := levels[currentLogLevel]
	messageLevel := levels[level]

	return messageLevel >= currentLevel
}

// writeLog escreve a entrada de log
func writeLog(level LogLevel, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     string(level),
		Message:   message,
		Fields:    fields,
	}

	if logFormat == "json" {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Erro ao converter log para JSON: %v", err)
			return
		}
		// mascarar campos sensíveis básicos
		masked := strings.ReplaceAll(string(jsonData), "\"cpf\":\"", "\"cpf\":\"***")
		log.Println(masked)
	} else {
		// Formato simples para desenvolvimento
		if len(fields) > 0 {
			fieldsJSON, _ := json.Marshal(fields)
			log.Printf("[%s] %s %s - %s", entry.Timestamp, entry.Level, entry.Message, string(fieldsJSON))
		} else {
			log.Printf("[%s] %s %s", entry.Timestamp, entry.Level, entry.Message)
		}
	}
}

// GetLoggerWithFields retorna um logger com campos pré-definidos
func GetLoggerWithFields(fields map[string]interface{}) *FieldLogger {
	return &FieldLogger{fields: fields}
}

// FieldLogger logger com campos pré-definidos
type FieldLogger struct {
	fields map[string]interface{}
}

// Info registra mensagem info com campos pré-definidos
func (fl *FieldLogger) Info(message string, additionalFields ...map[string]interface{}) {
	fields := mergeFields(fl.fields, additionalFields...)
	LogInfo(message, fields)
}

// Error registra mensagem de erro com campos pré-definidos
func (fl *FieldLogger) Error(message string, err error, additionalFields ...map[string]interface{}) {
	fields := mergeFields(fl.fields, additionalFields...)
	LogError(message, err, fields)
}

// Warn registra mensagem de aviso com campos pré-definidos
func (fl *FieldLogger) Warn(message string, additionalFields ...map[string]interface{}) {
	fields := mergeFields(fl.fields, additionalFields...)
	LogWarn(message, fields)
}

// Debug registra mensagem debug com campos pré-definidos
func (fl *FieldLogger) Debug(message string, additionalFields ...map[string]interface{}) {
	fields := mergeFields(fl.fields, additionalFields...)
	LogDebug(message, fields)
}

// mergeFields mescla os campos pré-definidos com campos adicionais
func mergeFields(base map[string]interface{}, additional ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copiar campos base
	for k, v := range base {
		result[k] = v
	}

	// Adicionar campos extras
	for _, fields := range additional {
		for k, v := range fields {
			result[k] = v
		}
	}

	return result
}
