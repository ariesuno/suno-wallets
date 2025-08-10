package ops

import (
	"net/http"
	"os"
	"suno-wallets/src/api/middlewares"
	appops "suno-wallets/src/application/ops"

	"github.com/gin-gonic/gin"
)

// Comentários em pt-BR: controller para scan/resolve/list de dedup

type DedupController struct{ svc *appops.DedupService }

func NewDedupController(svc *appops.DedupService) *DedupController { return &DedupController{svc: svc} }

func (c2 *DedupController) Scan(ctx *gin.Context) {
	if os.Getenv("ADMIN_SECRET") != "" && ctx.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	var body appops.ScanRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	created, err := c2.svc.Scan(ctx, tenantID, body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"created": created})
}

func (c2 *DedupController) Resolve(ctx *gin.Context) {
	if os.Getenv("ADMIN_SECRET") != "" && ctx.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	var body appops.ResolveRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := c2.svc.Resolve(ctx, tenantID, body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GET /ops/dedup/candidates
func (c2 *DedupController) List(ctx *gin.Context) {
	tenantID, ok := middlewares.GetTenantID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	cpf := ctx.Query("cpf")
	status := ctx.Query("status")
	page, pageSize := 1, 50
	items, err := c2.svc.ListCandidates(ctx, tenantID, cpf, status, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}
