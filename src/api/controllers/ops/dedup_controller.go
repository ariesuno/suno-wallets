package ops

import (
    "net/http"
    "github.com/gin-gonic/gin"
    appops "suno-wallets/src/application/ops"
    "suno-wallets/src/api/middlewares"
)

// Comentários em pt-BR: controller para scan/resolve/list de dedup

type DedupController struct{ svc *appops.DedupService }

func NewDedupController(svc *appops.DedupService) *DedupController { return &DedupController{svc: svc} }

func (c2 *DedupController) Scan(ctx *gin.Context) {
    tenantID, ok := middlewares.GetTenantID(ctx)
    if !ok { ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"}); return }
    var body appops.ScanRequest
    if err := ctx.ShouldBindJSON(&body); err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"}); return }
    created, err := c2.svc.Scan(ctx, tenantID, body)
    if err != nil { ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    ctx.JSON(http.StatusOK, gin.H{"created": created})
}

func (c2 *DedupController) Resolve(ctx *gin.Context) {
    tenantID, ok := middlewares.GetTenantID(ctx)
    if !ok { ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"}); return }
    var body appops.ResolveRequest
    if err := ctx.ShouldBindJSON(&body); err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"}); return }
    if err := c2.svc.Resolve(ctx, tenantID, body); err != nil { ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}


