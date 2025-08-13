package b3

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"suno-wallets/src/api/middlewares"
	incr "suno-wallets/src/application/b3/incremental"
	"suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: utilitário para inspeção de janela incremental (somente leitura)

type UtilController struct{ svc *incr.Service }

func NewUtilController(svc *incr.Service) *UtilController { return &UtilController{svc: svc} }

// SyncWindow godoc
// @Summary Janela de sincronização incremental
// @Description Calcula e retorna informações sobre a janela de sincronização incremental para um cliente, incluindo período de cobertura e estimativas de páginas a processar.
// @Tags B3 Utilities
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Param type query string true "Tipo de dados (transactions ou positions)" example(transactions)
// @Param since query string false "Data inicial no formato YYYY-MM-DD" example(2024-01-01)
// @Param end query string false "Data final no formato YYYY-MM-DD" example(2024-12-31)
// @Success 200 {object} map[string]interface{} "Informações da janela de sincronização"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Router /b3/client/sync-window [get]
func (c *UtilController) SyncWindow(ctx *gin.Context) {
	// Usar o middleware para obter o tenant ID (já validado)
	tenantID := middlewares.MustGetTenantName(ctx)

	cpf := ctx.Query("cpf")
	typ := ctx.Query("type") // transactions|positions
	sinceStr := ctx.Query("since")
	endStr := ctx.Query("end")

	if err := validation.ValidateCPF(cpf); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if typ != "transactions" && typ != "positions" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
		return
	}

	var sincePtr, endPtr *time.Time
	if sinceStr != "" {
		if err := validation.ValidateDateYMD(sinceStr); err == nil {
			t, _ := time.Parse("2006-01-02", sinceStr)
			sincePtr = &t
		}
	}
	if endStr != "" {
		if err := validation.ValidateDateYMD(endStr); err == nil {
			t, _ := time.Parse("2006-01-02", endStr)
			endPtr = &t
		}
	}

	out, _ := c.svc.Run(ctx, incr.Params{TenantID: tenantID, CPF: cpf, DataTypes: []string{typ}, AssetTypes: []string{"equity"}, Since: sincePtr, End: endPtr, DryRun: true})
	if out == nil {
		ctx.JSON(http.StatusOK, gin.H{"from": nil, "to": nil, "months": 0, "pagesEstimate": 0})
		return
	}
	var ts *incr.TypeSummary
	if typ == "transactions" {
		ts = out.Transactions
	} else {
		ts = out.Positions
	}
	if ts == nil {
		ctx.JSON(http.StatusOK, gin.H{"from": nil, "to": nil, "months": 0, "pagesEstimate": 0})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"from": ts.From, "to": ts.To, "months": ts.Raw.MonthsProcessed, "pagesEstimate": ts.Raw.PagesProcessed})
}
