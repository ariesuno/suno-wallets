package b3

import (
	"net/http"
	"os"
	"time"

	"suno-wallets/src/api/middlewares"
	apprecon "suno-wallets/src/application/b3/reconciliation"
	reconrepo "suno-wallets/src/infrastructure/b3/reconciliation"
	"suno-wallets/src/shared/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Comentários em pt-BR: Controller admin/read para inconsistency detector (1.17)

type ReconciliationController struct{ svc *apprecon.Service }

func NewReconciliationController(dbRepo *reconrepo.Repository) *ReconciliationController {
	return &ReconciliationController{svc: apprecon.NewService(dbRepo)}
}

// POST /reconciliation/scan (admin)
func (rc *ReconciliationController) Scan(c *gin.Context) {
	if os.Getenv("ADMIN_SECRET") != "" && c.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
		c.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
		return
	}
	tenantId, ok := middlewares.GetTenantID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	var req apprecon.ScanRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if req.Concurrency < 0 {
		req.Concurrency = 0
	}
	start := time.Now()
	totals, err := rc.svc.Scan(c, tenantId, req)
	if err != nil {
		helpers.LogError("recon_scan_error", err, map[string]interface{}{"tenantId": tenantId.String()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"totals": totals, "durationMs": time.Since(start).Milliseconds()})
}

// GET /reconciliation/inconsistencies (read-only)
func (rc *ReconciliationController) List(c *gin.Context) {
	// Placeholder: leitura simples inicial por tenant (poderá evoluir com filtros/paginação)
	tenantId, ok := middlewares.GetTenantID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	cpf := c.Query("cpf")
	status := c.Query("status")
	typ := c.Query("type")
	ticker := c.Query("ticker")
	// params simples de paginação
	page, pageSize := 1, 50
	items, err := rc.svc.List(c, tenantId, cpf, status, typ, ticker, nil, nil, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GET /reconciliation/inconsistencies/:id (read-only)
func (rc *ReconciliationController) Get(c *gin.Context) {
	tenantId, ok := middlewares.GetTenantID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
		return
	}
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	item, err := rc.svc.Get(c, tenantId, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get failed"})
		return
	}
	c.JSON(http.StatusOK, item)
}
