package b3

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	incr "suno-wallets/src/application/b3/incremental"
	"suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: utilitário para inspeção de janela incremental (somente leitura)

type UtilController struct{ svc *incr.Service }

func NewUtilController(svc *incr.Service) *UtilController { return &UtilController{svc: svc} }

// GET /b3/client/sync-window
func (c *UtilController) SyncWindow(ctx *gin.Context) {
	tenantIDStr := ctx.GetHeader("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_TENANT_ID"})
		return
	}

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
