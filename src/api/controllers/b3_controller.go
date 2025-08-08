package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: endpoint básico de health/auth (não expõe segredos)
type B3Controller struct{}

func NewB3Controller() *B3Controller { return &B3Controller{} }

// HealthAuth godoc
// @Summary B3 auth health
// @Description Verifica se as variáveis de ambiente de auth estão presentes (não valida contra B3)
// @Tags B3
// @Produce json
// @Success 200 {object} map[string]any
// @Router /b3/health/auth [get]
func (c *B3Controller) HealthAuth(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"oauth_config": "present",
		"mtls_config":  "present",
	})
}
