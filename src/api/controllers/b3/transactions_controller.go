package b3

import (
	"net/http"

	"suno-wallets/src/application/b3/transactions"
	"suno-wallets/src/shared/dto"
	"suno-wallets/src/shared/validation"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: controller de preview de transações v2

type TransactionsController struct {
	service transactions.TransactionsService
}

func NewTransactionsController(service transactions.TransactionsService) *TransactionsController {
	return &TransactionsController{service: service}
}

// FetchTransactionsPreview godoc
// @Summary Preview de transações v2 (sem persistência)
// @Description Busca transações por período e tipo de ativo na B3 e retorna o payload bruto (consolidado), sem persistir.
// @Tags B3 Data
// @Produce json
// @Param cpf query string true "CPF (11 dígitos)"
// @Param start query string true "Data inicial (YYYY-MM-DD)"
// @Param end query string true "Data final (YYYY-MM-DD)"
// @Param assetType query string false "Tipo de ativo (equity|fii|bdr|...) — somente equity implementado agora"
// @Param page query int false "Página inicial (>= 1), padrão 1"
// @Param fetchAllPages query bool false "Se true, pagina até o fim"
// @Success 200 {object} dto.TransactionsPreviewResponse
// @Failure 400 {object} map[string]string
// @Router /b3/fetch/transactions/preview [get]
func (c *TransactionsController) FetchTransactionsPreview(ctx *gin.Context) {
	var req dto.TransactionsPreviewRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// validações rápidas
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

	resp, err := c.service.PreviewTransactions(ctx, &req)
	if err != nil {
		// mapeamento simples de status
		code := http.StatusBadRequest
		switch err.Error() {
		case "invalid cpf: must be 11 digits":
			code = http.StatusBadRequest
		default:
			// manter 400 por padrão
		}
		ctx.JSON(code, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}
