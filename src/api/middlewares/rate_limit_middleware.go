package middlewares

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: Middleware simples de rate limit por tenant+path (janela deslizante por período)

type rlEntry struct {
	count int
	until time.Time
}

var (
	rlMu   sync.Mutex
	rlData = map[string]*rlEntry{}
)

func RateLimitMiddleware(max int, window time.Duration) gin.HandlerFunc {
	if max < 1 {
		max = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return func(c *gin.Context) {
		tenant := c.GetHeader("X-Tenant-Id")
		key := tenant + "|" + c.FullPath()
		now := time.Now()

		rlMu.Lock()
		entry, ok := rlData[key]
		if !ok || now.After(entry.until) {
			entry = &rlEntry{count: 0, until: now.Add(window)}
			rlData[key] = entry
		}
		entry.count++
		remaining := max - entry.count
		resetSec := int(time.Until(entry.until).Seconds())
		rlMu.Unlock()

		c.Header("X-RateLimit-Limit", itoa(max))
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", itoa(remaining))
		c.Header("X-RateLimit-Reset", itoa(resetSec))

		if entry.count > max {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func itoa(i int) string { return strconv.Itoa(i) }
