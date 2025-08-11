package b3

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"suno-wallets/src/api/middlewares"
	"suno-wallets/src/application/b3/ingest"
	"suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: controller para ingestão histórica de RAW da B3

type IngestController struct{ svc *ingest.Service }

func NewIngestController(svc *ingest.Service) *IngestController { return &IngestController{svc: svc} }

type ingestRequest struct {
	CPF       string `json:"cpf" binding:"required"`
	DataType  string `json:"dataType" binding:"required"` // transactions|positions
	AssetType string `json:"assetType"`                   // default: equity
	Start     string `json:"start" binding:"required"`
	End       string `json:"end" binding:"required"`
	FetchAll  bool   `json:"fetchAllPages"`
	Force     bool   `json:"force"`
	DryRun    bool   `json:"dryRun"`
}

// PostHistorical godoc
// @Summary Ingestão histórica de RAW (transactions v2 / positions v3)
// @Description Busca e persiste RAW por janelas mensais, evitando duplicidade por hash
// @Tags B3 Data (Persistence)
// @Accept json
// @Produce json
// @Param request body ingestRequest true "Parâmetros"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /b3/fetch/historical [post]
func (c *IngestController) PostHistorical(ctx *gin.Context) {
	// tenant obrigatório
	tenantUUID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		return
	}
	var req ingestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateCPF(req.CPF); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.DataType != "transactions" && req.DataType != "positions" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "dataType must be transactions or positions"})
		return
	}
	if req.AssetType == "" {
		req.AssetType = "equity"
	}
	if err := validation.ValidateDateYMD(req.Start); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateDateYMD(req.End); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := time.Parse("2006-01-02", req.Start); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid start"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.End); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid end"})
		return
	}

	params := ingest.IngestParams{
		TenantID:  tenantUUID,
		CPF:       req.CPF,
		DataType:  req.DataType,
		AssetType: req.AssetType,
		Start:     req.Start,
		End:       req.End,
		FetchAll:  req.FetchAll,
		Force:     req.Force,
		DryRun:    req.DryRun,
	}

	sum, err := c.svc.Ingest(ctx, params)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"saved":           sum.Saved,
		"skipped":         sum.Skipped,
		"errors":          sum.Errors,
		"monthsProcessed": sum.MonthsProcessed,
		"pagesProcessed":  sum.PagesProcessed,
		"dryRun":          sum.DryRun,
		"force":           sum.Force,
		"tenantId":        tenantUUID,
	})
}
