package middlewares

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"suno-wallets/src/shared/helpers"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// IdempotencyMiddleware gerencia idempotência via headers
type IdempotencyMiddleware struct {
	redisClient *redis.Client
	ttl         time.Duration
	keyPrefix   string
}

// IdempotentResponse representa uma resposta cached
type IdempotentResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Timestamp  time.Time         `json:"timestamp"`
}

// NewIdempotencyMiddleware cria nova instância do middleware
func NewIdempotencyMiddleware(redisClient *redis.Client, ttl time.Duration) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{
		redisClient: redisClient,
		ttl:         ttl,
		keyPrefix:   "suno:idempotency:",
	}
}

// Handler retorna o middleware Gin para idempotência
func (m *IdempotencyMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verificar se é um método que precisa de idempotência
		if !m.needsIdempotency(c.Request.Method) {
			c.Next()
			return
		}

		// Obter idempotency key do header
		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey == "" {
			// Prosseguir sem idempotência se não há key
			c.Next()
			return
		}

		// Validar formato da key (deve ser UUID ou similar)
		if !m.isValidIdempotencyKey(idempotencyKey) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_idempotency_key",
				"message": "Idempotency-Key must be a valid UUID or similar identifier",
			})
			c.Abort()
			return
		}

		// Criar chave Redis única baseada em request + tenant
		tenantID := c.GetString("tenant_id") // Set pelo tenant middleware
		requestKey := m.generateRequestKey(c, idempotencyKey, tenantID)

		// Verificar se já existe resposta cached
		if cachedResponse, exists := m.getCachedResponse(c, requestKey); exists {
			helpers.LogInfo("idempotency hit", map[string]interface{}{
				"idempotency_key": idempotencyKey,
				"tenant_id":       tenantID,
				"path":            c.Request.URL.Path,
				"method":          c.Request.Method,
				"cached_at":       cachedResponse.Timestamp,
			})

			// Retornar resposta cached
			m.returnCachedResponse(c, cachedResponse)
			return
		}

		// Usar ResponseWriter customizado para capturar resposta
		customWriter := &IdempotentResponseWriter{
			ResponseWriter: c.Writer,
			body:           make([]byte, 0),
			headers:        make(map[string]string),
		}
		c.Writer = customWriter

		// Processar request normalmente
		c.Next()

		// Cache da resposta se foi bem-sucedida (2xx)
		if customWriter.statusCode >= 200 && customWriter.statusCode < 300 {
			response := IdempotentResponse{
				StatusCode: customWriter.statusCode,
				Headers:    customWriter.headers,
				Body:       string(customWriter.body),
				Timestamp:  time.Now(),
			}

			m.cacheResponse(c, requestKey, response)

			helpers.LogInfo("idempotency miss - cached", map[string]interface{}{
				"idempotency_key": idempotencyKey,
				"tenant_id":       tenantID,
				"path":            c.Request.URL.Path,
				"method":          c.Request.Method,
				"status_code":     customWriter.statusCode,
			})
		}
	}
}

// needsIdempotency verifica se o método HTTP precisa de idempotência
func (m *IdempotencyMiddleware) needsIdempotency(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

// isValidIdempotencyKey valida o formato da chave de idempotência
func (m *IdempotencyMiddleware) isValidIdempotencyKey(key string) bool {
	// Validação básica: deve ter pelo menos 16 caracteres
	if len(key) < 16 || len(key) > 128 {
		return false
	}

	// Permitir apenas caracteres alfanuméricos, hífens e underscores
	for _, char := range key {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '-' && char != '_' {
			return false
		}
	}

	return true
}

// generateRequestKey gera uma chave única para o request
func (m *IdempotencyMiddleware) generateRequestKey(c *gin.Context, idempotencyKey, tenantID string) string {
	// Incluir path, method, tenant e idempotency key para unicidade
	keyData := fmt.Sprintf("%s:%s:%s:%s",
		c.Request.Method,
		c.Request.URL.Path,
		tenantID,
		idempotencyKey)

	// Hash para garantir tamanho consistente
	hash := sha256.Sum256([]byte(keyData))
	return m.keyPrefix + hex.EncodeToString(hash[:])
}

// getCachedResponse recupera resposta cached do Redis
func (m *IdempotencyMiddleware) getCachedResponse(c *gin.Context, key string) (*IdempotentResponse, bool) {
	result, err := m.redisClient.Get(c.Request.Context(), key).Result()
	if err != nil {
		if err != redis.Nil {
			helpers.LogError("failed to get cached response", err, map[string]interface{}{
				"key": key,
			})
		}
		return nil, false
	}

	var response IdempotentResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		helpers.LogError("failed to unmarshal cached response", err, map[string]interface{}{
			"key": key,
		})
		return nil, false
	}

	return &response, true
}

// cacheResponse salva resposta no Redis
func (m *IdempotencyMiddleware) cacheResponse(c *gin.Context, key string, response IdempotentResponse) {
	data, err := json.Marshal(response)
	if err != nil {
		helpers.LogError("failed to marshal response for cache", err, map[string]interface{}{
			"key": key,
		})
		return
	}

	if err := m.redisClient.Set(c.Request.Context(), key, data, m.ttl).Err(); err != nil {
		helpers.LogError("failed to cache response", err, map[string]interface{}{
			"key": key,
			"ttl": m.ttl.String(),
		})
	}
}

// returnCachedResponse retorna uma resposta previamente cached
func (m *IdempotencyMiddleware) returnCachedResponse(c *gin.Context, response *IdempotentResponse) {
	// Definir headers
	for key, value := range response.Headers {
		c.Header(key, value)
	}

	// Adicionar header indicando que foi resposta cached
	c.Header("X-Idempotency-Cache", "HIT")
	c.Header("X-Idempotency-Timestamp", response.Timestamp.Format(time.RFC3339))

	// Retornar response
	c.Data(response.StatusCode, "application/json", []byte(response.Body))
	c.Abort()
}

// IdempotentResponseWriter captura resposta para caching
type IdempotentResponseWriter struct {
	gin.ResponseWriter
	body       []byte
	headers    map[string]string
	statusCode int
}

func (w *IdempotentResponseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return w.ResponseWriter.Write(data)
}

func (w *IdempotentResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode

	// Capturar headers importantes
	for key, values := range w.ResponseWriter.Header() {
		if len(values) > 0 {
			w.headers[key] = values[0]
		}
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *IdempotentResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}
