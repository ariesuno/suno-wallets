package b3

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"suno-wallets/src/api/middlewares"
	app "suno-wallets/src/application/b3/reports"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"
	"suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: endpoints de relatórios (somente leitura), tag Swagger: B3 Reports

type ReportsController struct{ svc *app.Service }

func NewReportsController(svc *app.Service) *ReportsController { return &ReportsController{svc: svc} }

// RawDateRange godoc
// @Summary Relatório de datas dos dados brutos
// @Description Retorna as datas mínima e máxima dos dados brutos (raw) armazenados para um CPF específico. Útil para verificar o período de cobertura dos dados coletados da B3.
// @Tags B3 Reports
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Success 200 {object} map[string]interface{} "Período de cobertura dos dados"
// @Failure 400 {object} map[string]string "CPF inválido"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /b3/client/raw-date-range [get]
func (c *ReportsController) RawDateRange(ctx *gin.Context) {
	started := time.Now()
	endpoint := "raw-date-range"
	// Usar o middleware para obter o tenant ID (já validado)
	tenantID := middlewares.MustGetTenantID(ctx)
	cpf := ctx.Query("cpf")
	if err := validation.ValidateCPF(cpf); err != nil {
		observability.IncReportError(endpoint)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	helpers.LogInfo("reports raw-date-range", map[string]any{"tenantId": tenantID, "cpfMasked": maskCPF(cpf)})
	out, err := c.svc.GetRawDateRange(ctx, tenantID, cpf)
	observability.ObserveReport(endpoint, started)
	if err != nil {
		observability.IncReportError(endpoint)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	helpers.LogInfo("reports raw-date-range done", map[string]any{"tenantId": tenantID, "cpfMasked": maskCPF(cpf), "durationMs": time.Since(started).Milliseconds()})
	ctx.JSON(http.StatusOK, out)
}

// Summary godoc
// @Summary Relatório resumo de transações e posições
// @Description Retorna um resumo estatístico das transações e posições do cliente em um período específico, incluindo contadores, valores totais e distribuição por tipo de ativo.
// @Tags B3 Reports
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Param from query string true "Data inicial no formato YYYY-MM-DD" example(2024-01-01)
// @Param to query string true "Data final no formato YYYY-MM-DD" example(2024-12-31)
// @Success 200 {object} map[string]interface{} "Resumo estatístico do período"
// @Failure 400 {object} map[string]string "Parâmetros inválidos (CPF ou datas)"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /b3/client/summary [get]
func (c *ReportsController) Summary(ctx *gin.Context) {
	started := time.Now()
	endpoint := "summary"
	// Usar o middleware para obter o tenant ID (já validado)
	tenantID := middlewares.MustGetTenantID(ctx)
	cpf := ctx.Query("cpf")
	fromStr := ctx.Query("from")
	toStr := ctx.Query("to")
	if err := validation.ValidateCPF(cpf); err != nil {
		observability.IncReportError(endpoint)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateDateYMD(fromStr); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid from"})
		return
	}
	if err := validation.ValidateDateYMD(toStr); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid to"})
		return
	}
	from, _ := time.Parse("2006-01-02", fromStr)
	to, _ := time.Parse("2006-01-02", toStr)
	helpers.LogInfo("reports summary", map[string]any{"tenantId": tenantID, "cpfMasked": maskCPF(cpf), "from": fromStr, "to": toStr})
	out, err := c.svc.GetSummary(ctx, tenantID, cpf, from, to)
	observability.ObserveReport(endpoint, started)
	if err != nil {
		observability.IncReportError(endpoint)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	helpers.LogInfo("reports summary done", map[string]any{"tenantId": tenantID, "cpfMasked": maskCPF(cpf), "durationMs": time.Since(started).Milliseconds(), "months": out.MonthsWithTransactions, "tx": out.TotalTransactions, "tickers": out.TickersCount, "positions": out.PositionsCount})
	ctx.JSON(http.StatusOK, out)
}

// Tickers godoc
// @Summary Relatório de tickers (códigos de ativos)
// @Description Retorna lista paginada dos tickers (códigos de ativos) negociados pelo cliente em um período específico, com informações de volume e frequência.
// @Tags B3 Reports
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Param from query string true "Data inicial no formato YYYY-MM-DD" example(2024-01-01)
// @Param to query string true "Data final no formato YYYY-MM-DD" example(2024-12-31)
// @Param limit query int false "Limite de registros por página (1-1000)" example(100)
// @Param offset query int false "Número de registros para pular" example(0)
// @Success 200 {object} map[string]interface{} "Lista paginada de tickers"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /b3/client/tickers [get]
func (c *ReportsController) Tickers(ctx *gin.Context) {
	started := time.Now()
	endpoint := "tickers"
	// Usar o middleware para obter o tenant ID (já validado)
	tenantID := middlewares.MustGetTenantID(ctx)
	cpf := ctx.Query("cpf")
	fromStr := ctx.Query("from")
	toStr := ctx.Query("to")
	if err := validation.ValidateCPF(cpf); err != nil {
		observability.IncReportError(endpoint)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateDateYMD(fromStr); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid from"})
		return
	}
	if err := validation.ValidateDateYMD(toStr); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid to"})
		return
	}
	from, _ := time.Parse("2006-01-02", fromStr)
	to, _ := time.Parse("2006-01-02", toStr)
	// paginação
	limit := 100
	offset := 0
	if v := ctx.Query("limit"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			limit = n
		}
	}
	if v := ctx.Query("offset"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			offset = n
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	helpers.LogInfo("reports tickers", map[string]any{"tenantId": tenantID, "cpfMasked": maskCPF(cpf), "from": fromStr, "to": toStr, "limit": limit, "offset": offset})
	out, err := c.svc.GetTickers(ctx, tenantID, cpf, from, to, limit, offset)
	observability.ObserveReport(endpoint, started)
	if err != nil {
		observability.IncReportError(endpoint)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// log rows
	if arr, ok := out["tickers"].([]app.TickerRow); ok {
		helpers.LogInfo("reports tickers done", map[string]any{"tenantId": tenantID, "cpfMasked": maskCPF(cpf), "durationMs": time.Since(started).Milliseconds(), "rows": len(arr)})
	}
	ctx.JSON(http.StatusOK, out)
}

func maskCPF(cpf string) string {
	if len(cpf) != 11 {
		return "invalid"
	}
	return "*********" + cpf[9:]
}
