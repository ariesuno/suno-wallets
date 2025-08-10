package ops

import (
	"net/http"
	"net/http/httptest"
	opsctl "suno-wallets/src/api/controllers/ops"
	"suno-wallets/src/api/middlewares"
	opsapp "suno-wallets/src/application/ops"
	opsinfra "suno-wallets/src/infrastructure/ops"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTimelineRouter(t *testing.T) (*gin.Engine, *gorm.DB, uuid.UUID) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	db.Exec(`CREATE TABLE b3_operations_ledger (id text, tenant_id text, cpf text, ticker text, asset_type text, operation_date text, operation_type text, source text, quantity real, unit_price real, currency text, is_active bool, reason_code text, price_confidence text, generated_by_inconsistency_id text, supersedes_operation_id text, superseded_by_operation_id text, created_at text, updated_at text)`)
	db.Exec(`CREATE TABLE b3_inconsistencies (id text, tenant_id text, cpf text, ticker text, type text, status text, updated_at text)`)
	db.Exec(`CREATE TABLE dedup_policies (id text, tenant_id text, cpf text, mode text)`)

	r := gin.New()
	r.Use(middlewares.TenantMiddleware())
	timelineRepo := opsinfra.NewTimelineRepo(db)
	timelineSvc := opsapp.NewTimelineService(timelineRepo)
	timelineCtl := opsctl.NewTimelineController(timelineSvc)
	reconRepo := opsinfra.NewReconRepo(db)
	reconSvc := opsapp.NewReconSummaryService(reconRepo)
	reconCtl := opsctl.NewReconSummaryController(reconSvc)

	api := r.Group("/api/v1/ops")
	api.GET("/timeline", timelineCtl.Get)
	api.GET("/reconciliation/summary", reconCtl.Get)

	return r, db, uuid.New()
}

func TestTimelineAndSummary_Basic(t *testing.T) {
	r, db, tenant := setupTimelineRouter(t)
	now := time.Now().Format("2006-01-02")
	// seed ledger
	_ = db.Exec(`INSERT INTO b3_operations_ledger (id, tenant_id, cpf, ticker, asset_type, operation_date, operation_type, source, quantity, unit_price, currency, is_active, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, uuid.New().String(), tenant.String(), "00000000000", "ITSA4", "EQUITY", now, "BUY", "B3_RAW", 100, 10.0, "BRL", true, now, now).Error
	_ = db.Exec(`INSERT INTO b3_operations_ledger (id, tenant_id, cpf, ticker, asset_type, operation_date, operation_type, source, quantity, unit_price, currency, is_active, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, uuid.New().String(), tenant.String(), "00000000000", "ITSA4", "EQUITY", now, "OPENING_BALANCE", "SYSTEM_SYNTHETIC", 100, 9.5, "BRL", true, now, now).Error
	// seed inconsistency
	_ = db.Exec(`INSERT INTO b3_inconsistencies (id, tenant_id, cpf, ticker, type, status, updated_at) VALUES (?,?,?,?,?,?,?)`, uuid.New().String(), tenant.String(), "00000000000", "ITSA4", "OPENING_BALANCE_MISSING", "OPEN", now).Error

	// timeline
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/timeline?cpf=00000000000&pageSize=10", nil)
	req.Header.Set("X-Tenant-ID", tenant.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// summary
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/ops/reconciliation/summary?cpf=00000000000", nil)
	req2.Header.Set("X-Tenant-ID", tenant.String())
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
}
