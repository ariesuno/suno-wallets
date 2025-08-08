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

// GET /health/ready
func (h *HealthController) Ready(c *gin.Context) {
	dbErr := obs.CheckDB(c, h.db, 300*time.Millisecond)
	redisErr := error(nil)
	if h.rdb != nil {
		redisErr = obs.CheckRedis(c, h.rdb, 200*time.Millisecond)
	}
	if dbErr != nil || redisErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "components": gin.H{"db": statusOf(dbErr), "redis": statusOf(redisErr)}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "UP", "components": gin.H{"db": "UP", "redis": "UP"}})
}

// GET /health/details
func (h *HealthController) Details(c *gin.Context) {
	dbErr := obs.CheckDB(c, h.db, 300*time.Millisecond)
	redisErr := error(nil)
	if h.rdb != nil {
		redisErr = obs.CheckRedis(c, h.rdb, 200*time.Millisecond)
	}
	agg := "UP"
	if dbErr != nil || redisErr != nil {
		agg = "DEGRADED"
	}
	if dbErr != nil && redisErr != nil {
		agg = "DOWN"
	}
	c.JSON(http.StatusOK, gin.H{
		"status":     agg,
		"components": gin.H{"db": statusOf(dbErr), "redis": statusOf(redisErr)},
		"build":      gin.H{"version": build.Version, "commit": build.Commit, "buildDate": build.BuildDate},
	})
}

func statusOf(err error) string {
	if err != nil {
		return "DOWN"
	}
	return "UP"
}
