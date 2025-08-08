package b3

import (
	"errors"
	"net/http"

	positions "suno-wallets/src/application/b3/positions"
	b3errors "suno-wallets/src/infrastructure/b3/errors"
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
// @Param cpf query string true "CPF (11 dígitos)" example(12345678901)
// @Param start query string true "Data inicial (YYYY-MM-DD)" example(2024-01-01)
// @Param end query string true "Data final (YYYY-MM-DD)" example(2024-01-31)
// @Param assetType query string false "Tipo de ativo (equity...) — somente equity implementado agora" example(equity)
// @Param page query int false "Página inicial (>= 1), padrão 1" example(1)
// @Param fetchAllPages query bool false "Se true, pagina até o fim" example(false)
// @Success 200 {object} dto.PositionsPreviewResponse "Exemplo de sucesso"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 401 {object} map[string]string "Não autorizado / Token inválido"
// @Failure 403 {object} map[string]string "Acesso negado / Sem permissão"
// @Failure 429 {object} map[string]string "Limite de requisições excedido (Retry-After)"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
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
		var b3Err *b3errors.B3Error
		if errors.As(err, &b3Err) {
			switch b3Err.Status {
			case http.StatusUnauthorized:
				ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed, please renew credentials"})
				return
			case http.StatusForbidden:
				ctx.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				return
			case http.StatusTooManyRequests:
				ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests, please try again later"})
				return
			default:
				if b3Err.IsInternal() {
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal B3 API error"})
					return
				}
			}
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}
