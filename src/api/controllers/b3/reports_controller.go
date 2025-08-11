package b3

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	app "suno-wallets/src/application/b3/reports"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"
	"suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: endpoints de relatórios (somente leitura), tag Swagger: B3 Reports

type ReportsController struct{ svc *app.Service }

func NewReportsController(svc *app.Service) *ReportsController { return &ReportsController{svc: svc} }

// GET /b3/client/raw-date-range?cpf=...
func (c *ReportsController) RawDateRange(ctx *gin.Context) {
	started := time.Now()
	endpoint := "raw-date-range"
	tenantIDStr := ctx.GetHeader("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_TENANT_ID"})
		return
	}
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

// GET /b3/client/summary?cpf=...&from=...&to=...
func (c *ReportsController) Summary(ctx *gin.Context) {
	started := time.Now()
	endpoint := "summary"
	tenantID, err := uuid.Parse(ctx.GetHeader("X-Tenant-ID"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_TENANT_ID"})
		return
	}
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

// GET /b3/client/tickers?cpf=...&from=...&to=...
func (c *ReportsController) Tickers(ctx *gin.Context) {
	started := time.Now()
	endpoint := "tickers"
	tenantID, err := uuid.Parse(ctx.GetHeader("X-Tenant-ID"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_TENANT_ID"})
		return
	}
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
