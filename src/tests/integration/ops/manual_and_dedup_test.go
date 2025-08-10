package ops

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	opsctl "suno-wallets/src/api/controllers/ops"
	"suno-wallets/src/api/middlewares"
	opsapp "suno-wallets/src/application/ops"
	opsinfra "suno-wallets/src/infrastructure/ops"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupOpsRouter(t *testing.T) (*gin.Engine, *gorm.DB, uuid.UUID) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// schemas necessárias
	db.Exec(`CREATE TABLE b3_operations_ledger (id text, tenant_id text, cpf text, ticker text, asset_type text, operation_date text, operation_type text, source text, quantity real, unit_price real, currency text, is_active bool, superseded_by_operation_id text, created_by text, updated_by text, updated_at text)`)
	db.Exec(`CREATE TABLE dedup_candidates (id text, tenant_id text, cpf text, primary_operation_id text, candidate_operation_id text, score real, status text, rationale text, pair_key text, dedupe_key text, resolved_at text, resolved_by text)`)
	db.Exec(`CREATE UNIQUE INDEX uq_dedup_pair ON dedup_candidates(tenant_id, cpf, primary_operation_id, candidate_operation_id)`)
	db.Exec(`CREATE TABLE dedup_policies (id text, tenant_id text, cpf text, prefer_source text, auto_merge_threshold real, alert_threshold real, date_tolerance_days int, quantity_tolerance_ratio real, gross_tolerance_ratio real, mode text)`)

	r := gin.New()
	r.Use(middlewares.TenantMiddleware())
	// wiring
	opsRepo := opsinfra.NewOperationsRepository(db)
	manualSvc := opsapp.NewManualOperationsService(opsRepo)
	manualCtl := opsctl.NewManualOperationsController(manualSvc)
	dedupRepo := opsinfra.NewDedupRepo(db)
	dedupSvc := opsapp.NewDedupService(dedupRepo)
	dedupCtl := opsctl.NewDedupController(dedupSvc)

	api := r.Group("/api/v1")
	ops := api.Group("/ops")
	ops.POST("/manual", manualCtl.Create)
	ops.PUT("/manual/:id", manualCtl.Update)
	ops.DELETE("/manual/:id", manualCtl.Delete)
	ops.POST("/dedup/scan", dedupCtl.Scan)
	ops.POST("/dedup/resolve", dedupCtl.Resolve)
	ops.GET("/dedup/candidates", dedupCtl.List)

	return r, db, uuid.New()
}

func TestManualOps_Block_ByPolicy_B3Only(t *testing.T) {
	r, db, tenant := setupOpsRouter(t)
	// set policy B3_ONLY
	_ = db.Exec(`INSERT INTO dedup_policies (id, tenant_id, cpf, prefer_source, auto_merge_threshold, alert_threshold, date_tolerance_days, quantity_tolerance_ratio, gross_tolerance_ratio, mode) VALUES (?,?,?,?,?,?,?,?,?,?)`, uuid.New().String(), tenant.String(), "00000000000", "B3_RAW", 0.92, 0.7, 2, 0.005, 0.005, "B3_ONLY").Error
	body := map[string]any{"cpf": "00000000000", "ticker": "ABC", "assetType": "EQUITY", "operationDate": "2024-01-02", "operationType": "BUY", "quantity": 10}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/manual", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestManualOps_Create_And_Dedup_Scan_Resolve(t *testing.T) {
	r, db, tenant := setupOpsRouter(t)
	// set policy HYBRID
	_ = db.Exec(`INSERT INTO dedup_policies (id, tenant_id, cpf, prefer_source, auto_merge_threshold, alert_threshold, date_tolerance_days, quantity_tolerance_ratio, gross_tolerance_ratio, mode) VALUES (?,?,?,?,?,?,?,?,?,?)`, uuid.New().String(), tenant.String(), "00000000000", "B3_RAW", 0.92, 0.7, 2, 0.005, 0.005, "HYBRID").Error
	// inserir uma operação B3_RAW
	_ = db.Exec(`INSERT INTO b3_operations_ledger (id, tenant_id, cpf, ticker, asset_type, operation_date, operation_type, source, quantity, unit_price, currency, is_active) VALUES (?,?,?,?,?,?,?,?,?,?,?,true)`, uuid.New().String(), tenant.String(), "00000000000", "XYZ", "EQUITY", "2024-02-01", "SELL", "B3_RAW", 5, 10.0, "BRL").Error
	// criar USER_MANUAL similar
	body := map[string]any{"cpf": "00000000000", "ticker": "XYZ", "assetType": "EQUITY", "operationDate": "2024-02-02", "operationType": "SELL", "quantity": 5, "unitPrice": 10.0}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/manual", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// scan dedup
	scanReq := map[string]any{"cpf": "00000000000", "scanMode": "BATCH", "dryRun": false}
	scanPayload, _ := json.Marshal(scanReq)
	r1 := httptest.NewRequest(http.MethodPost, "/api/v1/ops/dedup/scan", bytes.NewReader(scanPayload))
	r1.Header.Set("Content-Type", "application/json")
	r1.Header.Set("X-Tenant-ID", tenant.String())
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, r1)
	require.Equal(t, http.StatusOK, w1.Code)

	// listar candidatos
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/ops/dedup/candidates?cpf=00000000000", nil)
	r2.Header.Set("X-Tenant-ID", tenant.String())
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, r2)
	require.Equal(t, http.StatusOK, w2.Code)

	// resolve override (preferir USER_MANUAL)
	var list struct {
		Items []struct{ ID string } `json:"items"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &list)
	if len(list.Items) > 0 {
		ids := []string{list.Items[0].ID}
		resReq := map[string]any{"action": "OVERRIDE", "candidateIds": ids}
		resPayload, _ := json.Marshal(resReq)
		r3 := httptest.NewRequest(http.MethodPost, "/api/v1/ops/dedup/resolve", bytes.NewReader(resPayload))
		r3.Header.Set("Content-Type", "application/json")
		r3.Header.Set("X-Tenant-ID", tenant.String())
		w3 := httptest.NewRecorder()
		r.ServeHTTP(w3, r3)
		require.Equal(t, http.StatusOK, w3.Code)
	}
}
