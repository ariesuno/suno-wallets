package ops

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"suno-wallets/src/api/middlewares"
	cpsvc "suno-wallets/src/application/clientpolicy"
	appops "suno-wallets/src/application/ops"
	obs "suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"
	"time"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: controller de timeline com keyset pagination e export

type TimelineController struct {
	svc      *appops.TimelineService
	enforcer *appops.ReadEnforcer
}

func NewTimelineController(s *appops.TimelineService) *TimelineController {
	return &TimelineController{svc: s}
}

func NewTimelineControllerWithPolicy(s *appops.TimelineService, pol *cpsvc.Service) *TimelineController {
	return &TimelineController{svc: s, enforcer: appops.NewReadEnforcer(pol)}
}

func (tc *TimelineController) Get(ctx *gin.Context) {
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
	tickers := []string{}
	if q := ctx.Query("tickers"); q != "" {
		tickers = strings.Split(q, ",")
	}
	sources := []string{}
	if q := ctx.Query("sources"); q != "" {
		sources = strings.Split(q, ",")
	}
	assetTypes := []string{}
	if q := ctx.Query("assetTypes"); q != "" {
		assetTypes = strings.Split(q, ",")
	}
	canonicalize := true
	if v := ctx.DefaultQuery("canonicalize", "true"); v == "false" {
		canonicalize = false
	}
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "100"))
	cur := ctx.Query("cursor")
	var cursor *string
	if cur != "" {
		cursor = &cur
	}
	var fromPtr, toPtr *time.Time
	if f := ctx.Query("from"); f != "" {
		if d, err := time.Parse("2006-01-02", f); err == nil {
			fromPtr = &d
		}
	}
	if t := ctx.Query("to"); t != "" {
		if d, err := time.Parse("2006-01-02", t); err == nil {
			toPtr = &d
		}
	}
	filters := appops.TimelineFilters{CPF: cpf, Tickers: tickers, From: fromPtr, To: toPtr, Sources: sources, AssetTypes: assetTypes, Canonicalize: canonicalize, PageSize: pageSize, Cursor: cursor}
	if tc.enforcer != nil {
		tc.enforcer.Apply(ctx, tenantID.String(), cpf, &filters)
	}
	items, next, err := tc.svc.List(ctx, tenantID.String(), filters)
	log := helpers.GetLoggerWithFields(map[string]interface{}{
		"endpoint":  "ops_timeline",
		"tenantId":  tenantID,
		"cpfMasked": maskCPF(cpf),
		"filters":   map[string]interface{}{"tickers": tickers, "sources": sources, "assetTypes": assetTypes, "from": fromPtr, "to": toPtr, "pageSize": pageSize},
	})
	if err != nil {
		obs.ObserveTimeline("error", started)
		log.Error("timeline_error", err, map[string]interface{}{"durationMs": time.Since(started).Milliseconds()})
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	obs.ObserveTimeline("success", started)
	log.Info("timeline_ok", map[string]interface{}{"items": len(items), "nextCursor": next != nil, "durationMs": time.Since(started).Milliseconds()})
	ctx.JSON(http.StatusOK, gin.H{"items": items, "nextCursor": next, "count": len(items)})
}

func (tc *TimelineController) Export(ctx *gin.Context) {
	started := time.Now()
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	if secret := os.Getenv("ADMIN_SECRET"); secret != "" && ctx.GetHeader("X-Admin-Secret") != secret {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	format := ctx.DefaultQuery("format", "csv")
	limit := 0
	if v := ctx.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit <= 0 {
		if env := os.Getenv("TIMELINE_EXPORT_MAX_ROWS"); env != "" {
			if n, err := strconv.Atoi(env); err == nil {
				limit = n
			}
		}
	}
	if limit <= 0 {
		limit = 50000
	}
	cpf := ctx.Query("cpf")
	if cpf == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cpf required"})
		return
	}
	items, _, err := tc.svc.List(ctx, tenantID.String(), appops.TimelineFilters{CPF: cpf, PageSize: limit})
	if err != nil {
		obs.ObserveTimeline("error", started)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	obs.IncTimelineExport(format)
	ctx.Header("Content-Disposition", "attachment; filename=timeline."+format)
	log := helpers.GetLoggerWithFields(map[string]interface{}{"endpoint": "ops_timeline_export", "tenantId": tenantID, "cpfMasked": maskCPF(cpf), "format": format, "limit": limit})
	if strings.ToLower(format) == "csv" {
		ctx.Header("Content-Type", "text/csv")
		w := ctx.Writer
		// cabeçalho CSV
		_, _ = w.Write([]byte("id,canonicalTicker,originalTicker,assetType,operationDate,operationType,source,quantity,unitPrice,currency,reasonCode,priceConfidence\n"))
		n := 0
		for _, it := range items {
			if n >= limit {
				break
			}
			line := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%.10f,", it.ID, it.CanonicalTicker, it.OriginalTicker, it.AssetType, it.OperationDate.Format("2006-01-02"), it.OperationType, it.Source, it.Quantity)
			if it.UnitPrice != nil {
				line += fmt.Sprintf("%.10f", *it.UnitPrice)
			}
			line += fmt.Sprintf(",%s,", it.Currency)
			if it.ReasonCode != nil {
				line += *it.ReasonCode
			}
			line += fmt.Sprintf(",%s\n", it.PriceConfidence)
			_, _ = w.Write([]byte(line))
			n++
		}
		log.Info("timeline_export_ok", map[string]interface{}{"rows": n, "durationMs": time.Since(started).Milliseconds()})
		return
	}
	// fallback JSON
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func maskCPF(cpf string) string {
	if len(cpf) < 4 {
		return "***"
	}
	return "***" + cpf[len(cpf)-4:]
}
