package b3

import (
	"net/http"

	positions "suno-wallets/src/application/b3/positions"
	"suno-wallets/src/shared/dto"
	"suno-wallets/src/shared/validation"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: controller de preview de posições v3 (equities)

type PositionsController struct{ service positions.Service }

func NewPositionsController(service positions.Service) *PositionsController {
	return &PositionsController{service: service}
}

// FetchPositionsPreview godoc
// @Summary Preview de posições v3 (sem persistência)
// @Description Busca posições por período e tipo de ativo na B3 e retorna o payload bruto (consolidado), sem persistir.
// @Tags B3 Data
// @Produce json
// @Param cpf query string true "CPF (11 dígitos)"
// @Param start query string true "Data inicial (YYYY-MM-DD)"
// @Param end query string true "Data final (YYYY-MM-DD)"
// @Param assetType query string false "Tipo de ativo (equity...) — somente equity implementado agora"
// @Param page query int false "Página inicial (>= 1), padrão 1"
// @Param fetchAllPages query bool false "Se true, pagina até o fim"
// @Success 200 {object} dto.PositionsPreviewResponse
// @Failure 400 {object} map[string]string
// @Router /b3/fetch/positions/preview [get]
func (c *PositionsController) FetchPositionsPreview(ctx *gin.Context) {
	var req dto.PositionsPreviewRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateCPF(req.CPF); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateDateYMD(req.Start); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateDateYMD(req.End); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := c.service.PreviewPositions(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}
