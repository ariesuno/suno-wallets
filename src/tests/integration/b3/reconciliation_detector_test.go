package b3

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	b3ctrl "suno-wallets/src/api/controllers/b3"
	"suno-wallets/src/api/middlewares"
	reconrepo "suno-wallets/src/infrastructure/b3/reconciliation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Comentários em pt-BR: integração leve do controller com SQLite em memória

func Test_Recon_Scan_List_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// criar schema mínimo usado pelo repo (tabelas normalizadas + inconsistencies)
	db.Exec("CREATE TABLE b3_normalized_positions (tenant_id text, cpf text, reference_date text, ticker text, quantity numeric)")
	db.Exec("CREATE TABLE b3_normalized_transactions (tenant_id text, cpf text, trade_date text, ticker text, side text)")
	db.Exec("CREATE TABLE b3_inconsistencies (id text, tenant_id text, cpf text, ticker text, type text, status text, severity int, updated_at text)")

	r := gin.New()
	r.Use(middlewares.TenantMiddleware())
	ctrl := b3ctrl.NewReconciliationController(reconrepo.NewRepository(db))
	r.POST("/api/v1/b3/reconciliation/scan", ctrl.Scan)
	r.GET("/api/v1/b3/reconciliation/inconsistencies", ctrl.List)

	// chamada de list vazia
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/b3/reconciliation/inconsistencies?cpf=00000000000", nil)
	req.Header.Set("X-Tenant-Id", uuid.New().String())
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
}

func setupReconRouterWithSchema(t *testing.T) (*gin.Engine, *gorm.DB, uuid.UUID) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// criar schema mínimo usado pelo repo (tabelas normalizadas + inconsistencies with unique index)
	db.Exec("CREATE TABLE b3_normalized_positions (tenant_id text, cpf text, reference_date text, ticker text, quantity numeric)")
	db.Exec("CREATE TABLE b3_normalized_transactions (tenant_id text, cpf text, trade_date text, ticker text, side text, quantity numeric)")
	db.Exec("CREATE TABLE b3_inconsistencies (id text, tenant_id text, cpf text, ticker text, type text, status text, severity int, updated_at text, first_detected_at text, last_detected_at text, affected_period_start text, affected_period_end text, sample_dates text, details text, dedupe_hash text, created_by_version text, created_by text)")
	db.Exec("CREATE UNIQUE INDEX uq_b3_inconsistencies_dedupe ON b3_inconsistencies(tenant_id, cpf, ticker, type, dedupe_hash)")

	r := gin.New()
	r.Use(middlewares.TenantMiddleware())
	ctrl := b3ctrl.NewReconciliationController(reconrepo.NewRepository(db))
	r.POST("/api/v1/b3/reconciliation/scan", ctrl.Scan)
	r.GET("/api/v1/b3/reconciliation/inconsistencies", ctrl.List)
	r.GET("/api/v1/b3/reconciliation/inconsistencies/:id", ctrl.Get)
	tenant := uuid.New()
	return r, db, tenant
}

func Test_Recon_Scan_DryRun_NoPersist(t *testing.T) {
	r, db, tenant := setupReconRouterWithSchema(t)
	// inserir dados que gerariam inconsistências
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_positions (tenant_id, cpf, reference_date, ticker, quantity) VALUES (?,?,?,?,?)`, tenant.String(), "00000000000", "2020-01-10", "ABC", 100).Error)
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_transactions (tenant_id, cpf, trade_date, ticker, side, quantity) VALUES (?,?,?,?,?,?)`, tenant.String(), "00000000000", "2020-02-01", "XYZ", "SELL", 10).Error)
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_positions (tenant_id, cpf, reference_date, ticker, quantity) VALUES (?,?,?,?,?)`, tenant.String(), "00000000000", "2020-03-01", "DEF", 150).Error)
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_transactions (tenant_id, cpf, trade_date, ticker, side, quantity) VALUES (?,?,?,?,?,?)`, tenant.String(), "00000000000", "2020-01-01", "DEF", "BUY", 100).Error)
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_transactions (tenant_id, cpf, trade_date, ticker, side, quantity) VALUES (?,?,?,?,?,?)`, tenant.String(), "00000000000", "2020-02-01", "DEF", "BUY", 20).Error)

	body := bytes.NewBufferString(`{"cpf":"00000000000","dryRun":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/b3/reconciliation/scan", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// banco não alterado
	var count int
	_ = db.Raw("SELECT COUNT(1) FROM b3_inconsistencies").Scan(&count).Error
	require.Equal(t, 0, count)
}

func Test_Recon_Scan_Persist_Idempotent(t *testing.T) {
	r, db, tenant := setupReconRouterWithSchema(t)
	// dados para 3 tipos
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_positions (tenant_id, cpf, reference_date, ticker, quantity) VALUES (?,?,?,?,?)`, tenant.String(), "00000000000", "2020-01-10", "ABC", 100).Error)
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_transactions (tenant_id, cpf, trade_date, ticker, side, quantity) VALUES (?,?,?,?,?,?)`, tenant.String(), "00000000000", "2020-02-01", "XYZ", "SELL", 10).Error)
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_positions (tenant_id, cpf, reference_date, ticker, quantity) VALUES (?,?,?,?,?)`, tenant.String(), "00000000000", "2020-03-01", "DEF", 150).Error)
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_transactions (tenant_id, cpf, trade_date, ticker, side, quantity) VALUES (?,?,?,?,?,?)`, tenant.String(), "00000000000", "2020-01-01", "DEF", "BUY", 100).Error)
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_transactions (tenant_id, cpf, trade_date, ticker, side, quantity) VALUES (?,?,?,?,?,?)`, tenant.String(), "00000000000", "2020-02-01", "DEF", "BUY", 20).Error)

	scan := func() int {
		body := bytes.NewBufferString(`{"cpf":"00000000000","dryRun":false}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/b3/reconciliation/scan", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenant.String())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	code := scan()
	require.Equal(t, 200, code)
	// deve persistir achados (>=3, pois um ticker pode gerar mais de um tipo)
	var count int
	_ = db.Raw("SELECT COUNT(1) FROM b3_inconsistencies WHERE tenant_id = ? AND cpf = ?", tenant.String(), "00000000000").Scan(&count).Error
	require.GreaterOrEqual(t, count, 3)

	// rodar novamente (idempotente)
	code2 := scan()
	require.Equal(t, 200, code2)
	var count2 int
	_ = db.Raw("SELECT COUNT(1) FROM b3_inconsistencies WHERE tenant_id = ? AND cpf = ?", tenant.String(), "00000000000").Scan(&count2).Error
	require.Equal(t, count, count2)
}

func Test_Recon_List_And_Get_Details(t *testing.T) {
	r, db, tenant := setupReconRouterWithSchema(t)
	// inserir e persistir
	_ = db.Exec(`INSERT INTO b3_normalized_positions (tenant_id, cpf, reference_date, ticker, quantity) VALUES (?,?,?,?,?)`, tenant.String(), "00000000000", "2020-01-10", "ABC", 100).Error
	body := bytes.NewBufferString(`{"cpf":"00000000000","dryRun":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/b3/reconciliation/scan", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// listar
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/b3/reconciliation/inconsistencies?cpf=00000000000&page=1&pageSize=10", nil)
	req2.Header.Set("X-Tenant-ID", tenant.String())
	r.ServeHTTP(w2, req2)
	require.Equal(t, 200, w2.Code)

	// pegar um id e detalhar
	type row struct{ ID string }
	var rid row
	_ = db.Raw("SELECT id FROM b3_inconsistencies WHERE tenant_id = ? AND cpf = ? LIMIT 1", tenant.String(), "00000000000").Scan(&rid).Error
	require.NotEmpty(t, rid.ID)
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/b3/reconciliation/inconsistencies/"+rid.ID, nil)
	req3.Header.Set("X-Tenant-ID", tenant.String())
	r.ServeHTTP(w3, req3)
	require.Equal(t, 200, w3.Code)
}
