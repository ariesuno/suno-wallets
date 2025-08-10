package controllers

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	b3client "suno-wallets/src/infrastructure/b3/client"
	b3auth "suno-wallets/src/infrastructure/b3/client/auth"
	b3cfg "suno-wallets/src/infrastructure/b3/config"
)

// Comentários em pt-BR: endpoint de health/auth da B3 (não expõe segredos)
type B3Controller struct {
	cfg   *b3cfg.B3Config
	creds b3auth.ClientCredentialsProvider
}

func NewB3Controller(cfg *b3cfg.B3Config, creds b3auth.ClientCredentialsProvider) *B3Controller {
	return &B3Controller{cfg: cfg, creds: creds}
}

// HealthAuth godoc
// @Summary B3 auth health
// @Description Controlado por flag OBS_ALLOW_B3_AUTH_HEALTH. Quando habilitado, valida Client Credentials (token) e montagem mTLS (.p12). Não chama endpoints de dados, nem loga segredos.
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

	// Timeout curto para validações
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// 1) Tenta obter token via Client Credentials (sem logar valor)
	if _, err := c.creds.GetToken(cctx); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"auth": "ERROR"})
		return
	}

	// 2) Tenta montar cliente mTLS (parsing do .p12)
	if _, err := b3client.NewB3OfficialClient(c.cfg, func(_ context.Context) (string, error) {
		// Não retornamos token aqui; não é necessário para validar mTLS
		return "", nil
	}); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"mtls": "ERROR"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"auth": "OK", "mtls": "OK"})
}
