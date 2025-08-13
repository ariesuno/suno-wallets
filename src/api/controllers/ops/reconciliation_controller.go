package ops

import (
	"net/http"
	"suno-wallets/src/api/middlewares"
	appops "suno-wallets/src/application/ops"
	obs "suno-wallets/src/infrastructure/observability"
	"time"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: controller para resumo de reconciliação (leitura)

type ReconSummaryController struct{ svc *appops.ReconSummaryService }

func NewReconSummaryController(s *appops.ReconSummaryService) *ReconSummaryController {
	return &ReconSummaryController{svc: s}
}

// Get godoc
// @Summary Resumo de reconciliação do cliente
// @Description Retorna resumo estatístico do status de reconciliação de um cliente, incluindo contadores de inconsistências, distribuição por tipos e informações de última reconciliação.
// @Tags Operations Reconciliation Summary
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Success 200 {object} map[string]interface{} "Resumo de reconciliação do cliente"
// @Failure 400 {object} map[string]string "CPF obrigatório"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /ops/reconciliation/summary [get]
func (rc *ReconSummaryController) Get(ctx *gin.Context) {
	started := time.Now()
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	cpf := ctx.Query("cpf")
	if cpf == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cpf required"})
		return
	}
	out, err := rc.svc.Get(ctx, tenantID.String(), cpf)
	if err != nil {
		obs.ObserveReconSummary("error", started)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	obs.ObserveReconSummary("success", started)
	ctx.JSON(http.StatusOK, out)
}
