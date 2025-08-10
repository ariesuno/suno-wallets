package observability

import (
	"net/http"
	"time"

	obs "suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/build"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Comentários em pt-BR: endpoints de liveness/readiness/details

type HealthController struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHealthController(db *gorm.DB, rdb *redis.Client) *HealthController {
	return &HealthController{db: db, rdb: rdb}
}

// Live godoc
// @Summary Liveness
// @Description Verifica se o processo está vivo (não consulta dependências).
// @Tags Observability
// @Produce json
// @Success 200 {object} map[string]any
// @Router /health/live [get]
// GET /health/live
func (h *HealthController) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "UP",
		"service":   "suno-wallets",
		"version":   build.Version,
		"commit":    build.Commit,
		"startedAt": build.StartedAt(),
		"uptimeSec": build.UptimeSeconds(),
	})
}

// Ready godoc
// @Summary Readiness
// @Description Retorna 200 somente quando DB, Redis (se configurado) e migrações estão OK.
// @Tags Observability
// @Produce json
// @Success 200 {object} map[string]any
// @Failure 503 {object} map[string]any
// @Router /health/ready [get]
// GET /health/ready
func (h *HealthController) Ready(c *gin.Context) {
	dbStart := time.Now()
	dbErr := obs.CheckDB(c, h.db, 300*time.Millisecond)
	dbLatency := time.Since(dbStart).Milliseconds()

	var redisErr error
	var redisLatency int64
	if h.rdb != nil {
		rdStart := time.Now()
		redisErr = obs.CheckRedis(c, h.rdb, 200*time.Millisecond)
		redisLatency = time.Since(rdStart).Milliseconds()
	}

	migStart := time.Now()
	migErr := obs.CheckMigrations(c, h.db, 300*time.Millisecond)
	migLatency := time.Since(migStart).Milliseconds()

	if dbErr != nil || redisErr != nil || migErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "DOWN",
			"components": gin.H{
				"db":         gin.H{"status": statusOf(dbErr), "latencyMs": dbLatency},
				"redis":      gin.H{"status": statusOf(redisErr), "latencyMs": redisLatency},
				"migrations": gin.H{"status": statusOf(migErr), "latencyMs": migLatency},
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
		"components": gin.H{
			"db":         gin.H{"status": "UP", "latencyMs": dbLatency},
			"redis":      gin.H{"status": statusOf(redisErr), "latencyMs": redisLatency},
			"migrations": gin.H{"status": "UP", "latencyMs": migLatency},
		},
	})
}

// Details godoc
// @Summary Health details
// @Description Status detalhado dos componentes, com latências e counters básicos.
// @Tags Observability
// @Produce json
// @Success 200 {object} map[string]any
// @Router /health/details [get]
// GET /health/details
func (h *HealthController) Details(c *gin.Context) {
	dbStart := time.Now()
	dbErr := obs.CheckDB(c, h.db, 300*time.Millisecond)
	dbLatency := time.Since(dbStart).Milliseconds()

	var redisErr error
	var redisLatency int64
	if h.rdb != nil {
		rdStart := time.Now()
		redisErr = obs.CheckRedis(c, h.rdb, 200*time.Millisecond)
		redisLatency = time.Since(rdStart).Milliseconds()
	}

	agg := "UP"
	if dbErr != nil || redisErr != nil {
		agg = "DEGRADED"
	}
	if dbErr != nil && redisErr != nil {
		agg = "DOWN"
	}

	dbPool := gin.H{}
	if h.db != nil {
		if sqlDB, err := h.db.DB(); err == nil {
			st := sqlDB.Stats()
			dbPool = gin.H{
				"openConns": st.OpenConnections,
				"inUse":     st.InUse,
				"idle":      st.Idle,
				"waitCount": st.WaitCount,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": agg,
		"components": gin.H{
			"db":    gin.H{"status": statusOf(dbErr), "latencyMs": dbLatency, "pool": dbPool},
			"redis": gin.H{"status": statusOf(redisErr), "latencyMs": redisLatency},
		},
		"build": gin.H{"version": build.Version, "commit": build.Commit, "buildDate": build.BuildDate},
	})
}

func statusOf(err error) string {
	if err != nil {
		return "DOWN"
	}
	return "UP"
}
