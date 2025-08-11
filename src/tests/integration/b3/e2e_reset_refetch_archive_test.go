package b3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	b3 "suno-wallets/src/api/controllers/b3"
	"suno-wallets/src/api/middlewares"
	appe2e "suno-wallets/src/application/b3/e2e"
	appingest "suno-wallets/src/application/b3/ingest"
	appnorm "suno-wallets/src/application/b3/normalize"
	obs "suno-wallets/src/infrastructure/observability"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Fakes para Ingest/Normalize (evita dependência de SQL específico/cliente B3)
type fakeIngest struct{ simulateRetry bool }

func (f *fakeIngest) Ingest(ctx context.Context, p appingest.IngestParams) (*appingest.Summary, error) {
	if f.simulateRetry {
		obs.IncB3Retries("/assets-trading/v2/equity")
	}
	months := 1
	if len(p.Start) == 10 && len(p.End) == 10 {
		st, _ := time.Parse("2006-01-02", p.Start)
		en, _ := time.Parse("2006-01-02", p.End)
		months = int((en.Year()-st.Year())*12 + int(en.Month()) - int(st.Month()) + 1)
		if months < 1 {
			months = 1
		}
	}
	saved := 0
	if p.Force {
		saved = months
	}
	return &appingest.Summary{Saved: saved, Skipped: 0, Errors: 0, MonthsProcessed: months, PagesProcessed: months, DryRun: false, Force: p.Force}, nil
}

type fakeNormalize struct{}

func (f *fakeNormalize) Run(ctx context.Context, p appnorm.RunParams) (*appnorm.Summary, error) {
	return &appnorm.Summary{Inserted: 1, Updated: 0, Skipped: 0, Errors: 0, RawProcessed: 1}, nil
}

// Reset repo que move linhas reais nas tabelas sqlite de teste
type sqliteResetRepo struct{ db *gorm.DB }

func (r *sqliteResetRepo) TryAcquireLock(ctx context.Context, tenantID string, cpf string) (bool, error) {
	return true, nil
}
func (r *sqliteResetRepo) ReleaseLock(ctx context.Context, tenantID string, cpf string) error {
	return nil
}
func (r *sqliteResetRepo) Reset(ctx context.Context, tenantID string, cpf string, mode string, archivedBy string) (*appe2e.ResetResult, error) {
	res := &appe2e.ResetResult{}
	if mode == "archive" {
		_ = r.db.WithContext(ctx).Exec(`INSERT INTO b3_raw_data_client_archive
            SELECT id, tenant_id, cpf, data_type, asset_type, period_start, period_end, page, payload_json, payload_hash,
                   source_version, endpoint_path, http_status, fetched_at, retry_count, request_id, normalized_at, normalized_count,
                   ?, ? FROM b3_raw_data_client WHERE tenant_id = ? AND cpf = ?`, time.Now().Format(time.RFC3339), archivedBy, tenantID, cpf).Error
		var cnt int64
		_ = r.db.WithContext(ctx).Raw(`SELECT COUNT(1) FROM b3_raw_data_client_archive WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Scan(&cnt).Error
		res.RawMoved = int(cnt)
		_ = r.db.WithContext(ctx).Exec(`DELETE FROM b3_raw_data_client WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Error
	} else {
		tx := r.db.WithContext(ctx).Exec(`DELETE FROM b3_raw_data_client WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf)
		res.RawDeleted = int(tx.RowsAffected)
	}
	// limpar períodos
	_ = r.db.WithContext(ctx).Exec(`DELETE FROM b3_fetched_periods WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Error
	res.SyncStateReset = true
	return res, nil
}

func setupDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Tabelas mínimas
	exec := func(sql string) { require.NoError(t, db.Exec(sql).Error) }
	exec(`CREATE TABLE b3_raw_data_client (
    id TEXT PRIMARY KEY, tenant_id TEXT, cpf TEXT, data_type TEXT, asset_type TEXT,
    period_start TEXT, period_end TEXT, page INTEGER, payload_json TEXT, payload_hash TEXT,
    source_version TEXT, endpoint_path TEXT, http_status INTEGER, fetched_at TEXT,
    retry_count INTEGER, request_id TEXT, normalized_at TEXT, normalized_count INTEGER
  );`)
	exec(`CREATE TABLE b3_raw_data_client_archive (
    id TEXT, tenant_id TEXT, cpf TEXT, data_type TEXT, asset_type TEXT,
    period_start TEXT, period_end TEXT, page INTEGER, payload_json TEXT, payload_hash TEXT,
    source_version TEXT, endpoint_path TEXT, http_status INTEGER, fetched_at TEXT,
    retry_count INTEGER, request_id TEXT, normalized_at TEXT, normalized_count INTEGER,
    archived_at TEXT, archived_by TEXT
  );`)
	exec(`CREATE TABLE b3_normalized_transactions (
    id TEXT PRIMARY KEY, tenant_id TEXT, cpf TEXT, asset_type TEXT, source_version TEXT, raw_id TEXT, sequence_in_raw INTEGER,
    trade_id TEXT, broker_code TEXT, trade_date TEXT, settlement_date TEXT, ticker TEXT, isin TEXT, side TEXT, quantity TEXT,
    price TEXT, gross_value TEXT, currency TEXT, extra_json TEXT, normalized_hash TEXT, normalized_at TEXT
  );`)
	exec(`CREATE TABLE b3_normalized_transactions_archive (
    id TEXT, tenant_id TEXT, cpf TEXT, asset_type TEXT, source_version TEXT, raw_id TEXT, sequence_in_raw INTEGER,
    trade_id TEXT, broker_code TEXT, trade_date TEXT, settlement_date TEXT, ticker TEXT, isin TEXT, side TEXT, quantity TEXT,
    price TEXT, gross_value TEXT, currency TEXT, extra_json TEXT, normalized_hash TEXT, normalized_at TEXT, archived_at TEXT, archived_by TEXT
  );`)
	exec(`CREATE TABLE b3_normalized_positions (
    id TEXT PRIMARY KEY, tenant_id TEXT, cpf TEXT, asset_type TEXT, source_version TEXT, raw_id TEXT, sequence_in_raw INTEGER,
    reference_date TEXT, ticker TEXT, isin TEXT, quantity TEXT, avg_price TEXT, position_value TEXT, currency TEXT, extra_json TEXT,
    normalized_hash TEXT, normalized_at TEXT
  );`)
	exec(`CREATE TABLE b3_normalized_positions_archive (
    id TEXT, tenant_id TEXT, cpf TEXT, asset_type TEXT, source_version TEXT, raw_id TEXT, sequence_in_raw INTEGER,
    reference_date TEXT, ticker TEXT, isin TEXT, quantity TEXT, avg_price TEXT, position_value TEXT, currency TEXT, extra_json TEXT,
    normalized_hash TEXT, normalized_at TEXT, archived_at TEXT, archived_by TEXT
  );`)
	exec(`CREATE TABLE b3_fetched_periods (
    id TEXT PRIMARY KEY, tenant_id TEXT, cpf TEXT, data_type TEXT, asset_type TEXT, month_start TEXT, month_end TEXT, pages INTEGER, completed BOOLEAN, last_fetched_at TEXT
  );`)
	exec(`CREATE TABLE b3_sync_state (
    id TEXT PRIMARY KEY, tenant_id TEXT, cpf TEXT, is_active BOOLEAN, last_tx_sync_at TEXT, last_pos_sync_at TEXT,
    last_checked_at TEXT, last_result TEXT, last_error TEXT, needs_reprocess BOOLEAN, failure_count INTEGER, created_at TEXT, updated_at TEXT
  );`)
	return db
}

func TestE2E_ResetArchive_RealFlow_Idempotency_AndRetries(t *testing.T) {
	_ = os.Setenv("ADMIN_SECRET", "")
	gin.SetMode(gin.TestMode)
	db := setupDB(t)

	// Pre-insere dado antigo para ser arquivado
	tenant := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO b3_raw_data_client (id, tenant_id, cpf, data_type, asset_type, period_start, period_end, page, payload_json, payload_hash, source_version, endpoint_path, http_status, fetched_at, retry_count, request_id)
    VALUES (?, ?, ?, 'transactions','equity','2023-12-01','2023-12-31',1,'{}','h','v2','/x',200,'now',0,'req')`, uuid.New().String(), tenant.String(), "00000000000").Error)
	// serviços fake e orquestrador com reset sqlite real
	ingestSvc := &fakeIngest{simulateRetry: true}
	normSvc := &fakeNormalize{}
	orch := appe2e.NewOrchestratorPorts(&sqliteResetRepo{db: db}, ingestSvc, normSvc)
	ctrl := b3.NewAdminController(orch, nil)

	r := gin.New()
	r.Use(obs.PrometheusMiddleware())
	api := r.Group("/api/v1")
	api.Use(middlewares.TenantMiddleware())
	api.POST("/b3/admin/reset-and-refetch", ctrl.ResetAndRefetch)
	obs.RegisterMetricsRoute(r)

	// Execução real com force=true
	body := map[string]any{
		"cpf":        "00000000000",
		"assetTypes": []string{"equity"},
		"dataTypes":  []string{"transactions"},
		"force":      true,
		"dryRun":     false,
		"mode":       "archive",
		"confirm":    "RESET_AND_REFETCH",
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/b3/admin/reset-and-refetch", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	// valida movimento para _archive e limpeza da tabela base
	var archived int
	_ = db.Raw(`SELECT COUNT(1) FROM b3_raw_data_client_archive WHERE tenant_id = ? AND cpf = ?`, tenant.String(), "00000000000").Scan(&archived).Error
	require.Greater(t, archived, 0)
	var left int
	_ = db.Raw(`SELECT COUNT(1) FROM b3_raw_data_client WHERE tenant_id = ? AND cpf = ?`, tenant.String(), "00000000000").Scan(&left).Error
	require.Equal(t, 0, left)

	// Verifica métrica de retry apareceu
	mw := httptest.NewRecorder()
	mreq := httptest.NewRequest("GET", "/metrics", nil)
	r.ServeHTTP(mw, mreq)
	require.Equal(t, 200, mw.Code)
	require.Contains(t, mw.Body.String(), "b3_retries_total")

	// Segunda execução (idempotência) sem force
	body["force"] = false
	payload2, _ := json.Marshal(body)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/b3/admin/reset-and-refetch", bytes.NewReader(payload2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Tenant-ID", tenant.String())
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, 200, w2.Code)
	// idempotência: não aumenta archive na segunda execução sem force
	var archived2 int
	_ = db.Raw(`SELECT COUNT(1) FROM b3_raw_data_client_archive WHERE tenant_id = ? AND cpf = ?`, tenant.String(), "00000000000").Scan(&archived2).Error
	require.Equal(t, archived, archived2)
}
