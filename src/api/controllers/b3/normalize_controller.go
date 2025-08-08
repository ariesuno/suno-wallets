package b3

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"suno-wallets/src/api/middlewares"
	appnorm "suno-wallets/src/application/b3/normalize"
	"suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: controller para normalização de RAW

type NormalizeController struct{ svc *appnorm.Service }

func NewNormalizeController(svc *appnorm.Service) *NormalizeController {
	return &NormalizeController{svc: svc}
}

type normalizeRequest struct {
	CPF       string `json:"cpf" binding:"required"`
	DataType  string `json:"dataType" binding:"required"`
	AssetType string `json:"assetType"`
	Start     string `json:"start" binding:"required"`
	End       string `json:"end" binding:"required"`
	Force     bool   `json:"force"`
	DryRun    bool   `json:"dryRun"`
}

// Run godoc
// @Summary Normalização de RAW (transactions v2 / positions v3)
// @Tags B3 Data (Normalization)
// @Accept json
// @Produce json
// @Param request body normalizeRequest true "Parâmetros"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /b3/normalize/run [post]
func (c *NormalizeController) Run(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		return
	}

	var req normalizeRequest
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

	// valida datas
	if err := validation.ValidateDateYMD(req.Start); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateDateYMD(req.End); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	startT, _ := time.Parse("2006-01-02", req.Start)
	endT, _ := time.Parse("2006-01-02", req.End)

	sum, err := c.svc.Run(ctx, appnorm.RunParams{TenantID: tenantID, CPF: req.CPF, DataType: req.DataType, AssetType: req.AssetType, Start: startT, End: endT, Force: req.Force, DryRun: req.DryRun})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"inserted": sum.Inserted, "updated": sum.Updated, "skipped": sum.Skipped, "errors": sum.Errors, "rawProcessed": sum.RawProcessed})
}
