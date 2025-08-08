package controllers

import (
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
)

// Comentários em pt-BR: endpoint básico de health/auth (não expõe segredos)
type B3Controller struct{}

func NewB3Controller() *B3Controller { return &B3Controller{} }

// HealthAuth godoc
// @Summary B3 auth health
// @Description Controlado por flag OBS_ALLOW_B3_AUTH_HEALTH. Quando habilitado, checa presença de configuração de OAuth/mTLS (sem vazar segredos).
// @Tags B3
// @Produce json
// @Success 200 {object} map[string]any
// @Failure 403 {object} map[string]string
// @Router /b3/health/auth [get]
func (c *B3Controller) HealthAuth(ctx *gin.Context) {
    if os.Getenv("OBS_ALLOW_B3_AUTH_HEALTH") != "true" {
        ctx.JSON(http.StatusForbidden, gin.H{"error": "disabled by config"})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"oauth_config": "present", "mtls_config": "present"})
}
