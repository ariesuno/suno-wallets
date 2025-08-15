package controllers

import (
	"net/http"
	"time"

	"suno-wallets/src/infrastructure/data"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthController controller para verificações de saúde
type HealthController struct {
	db          *gorm.DB
	redisClient *redis.Client
}

// NewHealthController cria uma nova instância do controller de saúde
func NewHealthController(db *gorm.DB, redisClient *redis.Client) *HealthController {
	return &HealthController{
		db:          db,
		redisClient: redisClient,
	}
}

// HealthResponse resposta do health check
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Version   string            `json:"version"`
}

// HealthCheck verifica a saúde da aplicação
// @Summary Health check
// @Description Verifica se a aplicação e seus serviços dependentes estão funcionando
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /health [get]
func (ctrl *HealthController) HealthCheck(c *gin.Context) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Services:  make(map[string]string),
		Version:   "1.0.0",
	}

	// Verificar conexão com banco de dados
	if sqlDB, err := ctrl.db.DB(); err != nil {
		response.Status = "error"
		response.Services["database"] = "error: " + err.Error()
	} else if err := sqlDB.Ping(); err != nil {
		response.Status = "error"
		response.Services["database"] = "error: " + err.Error()
	} else {
		response.Services["database"] = "ok"
	}

	// Verificar conexão com Redis
	if ctrl.redisClient != nil {
		if err := data.PingRedis(c.Request.Context(), ctrl.redisClient); err != nil {
			response.Status = "error"
			response.Services["redis"] = "error: " + err.Error()
		} else {
			response.Services["redis"] = "ok"
		}
	} else {
		response.Services["redis"] = "not configured"
	}

	// Determinar status HTTP
	statusCode := http.StatusOK
	if response.Status == "error" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// ReadinessCheck verifica se a aplicação está pronta para receber tráfego
// @Summary Readiness check
// @Description Verifica se a aplicação está pronta para receber requisições
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /ready [get]
func (ctrl *HealthController) ReadinessCheck(c *gin.Context) {
	// Verificar se pode executar uma query simples no banco
	var result int
	if err := ctrl.db.Raw("SELECT 1").Scan(&result).Error; err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not ready",
			"message": "Banco de dados não disponível",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now().UTC(),
	})
}

// LivenessCheck verifica se a aplicação está viva
// @Summary Liveness check
// @Description Verifica se a aplicação está viva (para Kubernetes)
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /live [get]
func (ctrl *HealthController) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().UTC(),
	})
}
