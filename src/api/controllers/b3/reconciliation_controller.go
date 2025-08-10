package b3

import (
	"net/http"
	"os"
	"strconv"
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

// Auto-fix wiring simplificado (1.18)
type AutoFixController struct{ svc *apprecon.SystemOperationsService }

func NewAutoFixController(sysRepo *reconrepo.SysOpsRepository, price apprecon.PriceLookupPort) *AutoFixController {
    return &AutoFixController{svc: apprecon.NewSystemOperationsService(sysRepo, price)}
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
	result, err := rc.svc.Scan(c, tenantId, req)
	if err != nil {
		helpers.LogError("recon_scan_error", err, map[string]interface{}{"tenantId": tenantId.String()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
		return
	}
	// anexa duration também no controller
	result["durationMsController"] = time.Since(start).Milliseconds()
	c.JSON(http.StatusOK, result)
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
	// filtros adicionais e paginação
	var fromPtr, toPtr *time.Time
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			fromPtr = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			toPtr = &t
		}
	}
	page, pageSize := 1, 50
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			pageSize = n
		}
	}
	items, err := rc.svc.List(c, tenantId, cpf, status, typ, ticker, fromPtr, toPtr, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "pageSize": pageSize})
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

// POST /reconciliation/auto-fix (admin)
func (ac *AutoFixController) AutoFix(c *gin.Context) {
    if os.Getenv("ADMIN_SECRET") != "" && c.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
        c.JSON(http.StatusForbidden, gin.H{"error": "FORBIDDEN"})
        return
    }
    tenantId, ok := middlewares.GetTenantID(c)
    if !ok {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant"})
        return
    }
    var req apprecon.AutoFixRequest
    if err := c.BindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
        return
    }
    out, err := ac.svc.AutoFix(c, tenantId, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, out)
}
