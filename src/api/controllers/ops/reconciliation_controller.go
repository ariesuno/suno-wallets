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
	out, err := rc.svc.Get(ctx, tenantID, cpf)
	if err != nil {
		obs.ObserveReconSummary("error", started)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	obs.ObserveReconSummary("success", started)
	ctx.JSON(http.StatusOK, out)
}
