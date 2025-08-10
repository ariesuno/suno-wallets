package b3_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	b3 "suno-wallets/src/api/controllers/b3"
	"suno-wallets/src/api/middlewares"
	repsvc "suno-wallets/src/application/b3/reports"
	reprepo "suno-wallets/src/infrastructure/b3/reports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupReportsDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	exec := func(sql string) { require.NoError(t, db.Exec(sql).Error) }
	exec(`CREATE TABLE b3_raw_data_client (
        id TEXT PRIMARY KEY, tenant_id TEXT, cpf TEXT, data_type TEXT, asset_type TEXT,
        period_start TEXT, period_end TEXT, page INTEGER, payload_json TEXT, payload_hash TEXT,
        source_version TEXT, endpoint_path TEXT, http_status INTEGER, fetched_at TEXT,
        retry_count INTEGER, request_id TEXT
    );`)
	exec(`CREATE TABLE b3_normalized_transactions (
        id TEXT PRIMARY KEY, tenant_id TEXT, cpf TEXT, asset_type TEXT, source_version TEXT, raw_id TEXT, sequence_in_raw INTEGER,
        trade_id TEXT, broker_code TEXT, trade_date TEXT, settlement_date TEXT, ticker TEXT, isin TEXT, side TEXT, quantity TEXT,
        price TEXT, gross_value TEXT, currency TEXT, extra_json TEXT, normalized_hash TEXT, normalized_at TEXT
    );`)
	exec(`CREATE TABLE b3_normalized_positions (
        id TEXT PRIMARY KEY, tenant_id TEXT, cpf TEXT, asset_type TEXT, source_version TEXT, raw_id TEXT, sequence_in_raw INTEGER,
        reference_date TEXT, ticker TEXT, isin TEXT, quantity TEXT, avg_price TEXT, position_value TEXT, currency TEXT, extra_json TEXT,
        normalized_hash TEXT, normalized_at TEXT
    );`)
	return db
}

func TestReports_RawDateRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupReportsDB(t)
	repo := reprepo.NewRepository(db)
	svc := repsvc.NewService(repo)
	ctrl := b3.NewReportsController(svc)

	tenant := uuid.New()
	// inserir dois períodos no RAW
	require.NoError(t, db.Exec(`INSERT INTO b3_raw_data_client (id, tenant_id, cpf, data_type, asset_type, period_start, period_end, page, payload_json, payload_hash, source_version, endpoint_path, http_status, fetched_at, retry_count, request_id)
        VALUES ('1', ?, '00000000000', 'transactions','equity','2023-12-01','2023-12-31',1,'{}','h','v2','/x',200,'now',0,'req')`, tenant.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO b3_raw_data_client (id, tenant_id, cpf, data_type, asset_type, period_start, period_end, page, payload_json, payload_hash, source_version, endpoint_path, http_status, fetched_at, retry_count, request_id)
        VALUES ('2', ?, '00000000000', 'transactions','equity','2024-01-01','2024-01-31',1,'{}','h2','v2','/x',200,'now',0,'req')`, tenant.String()).Error)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middlewares.TenantMiddleware())
	api.GET("/b3/client/raw-date-range", ctrl.RawDateRange)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/b3/client/raw-date-range?cpf=00000000000", nil)
	req.Header.Set("X-Tenant-ID", tenant.String())
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	body := w.Body.String()
	require.Contains(t, body, `"from":"2023-12-01"`)
	require.Contains(t, body, `"to":"2024-01-31"`)
}

func TestReports_Summary_And_Tickers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupReportsDB(t)
	repo := reprepo.NewRepository(db)
	svc := repsvc.NewService(repo)
	ctrl := b3.NewReportsController(svc)

	tenant := uuid.New()
	// inserir normalized tx em 2 meses
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_transactions (id, tenant_id, cpf, asset_type, source_version, raw_id, sequence_in_raw, trade_date, ticker, side, quantity, price, normalized_hash, normalized_at)
        VALUES ('t1', ?, '00000000000', 'equity','v2','raw',1,'2024-02-10','ABCD3','BUY','10','1','h','now')`, tenant.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_transactions (id, tenant_id, cpf, asset_type, source_version, raw_id, sequence_in_raw, trade_date, ticker, side, quantity, price, normalized_hash, normalized_at)
        VALUES ('t2', ?, '00000000000', 'equity','v2','raw',2,'2024-03-05','EFGH4','BUY','5','1','h2','now')`, tenant.String()).Error)
	// inserir positions
	require.NoError(t, db.Exec(`INSERT INTO b3_normalized_positions (id, tenant_id, cpf, asset_type, source_version, raw_id, sequence_in_raw, reference_date, ticker, quantity, position_value, currency, normalized_hash, normalized_at)
        VALUES ('p1', ?, '00000000000', 'equity','v3','raw',1,'2024-03-05','ABCD3','15','150.00','BRL','hp','now')`, tenant.String()).Error)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middlewares.TenantMiddleware())
	api.GET("/b3/client/summary", ctrl.Summary)
	api.GET("/b3/client/tickers", ctrl.Tickers)

	// summary
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/b3/client/summary?cpf=00000000000&from=2024-02-01&to=2024-03-31", nil)
	req.Header.Set("X-Tenant-ID", tenant.String())
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	s := w.Body.String()
	require.Contains(t, s, `"monthsWithTransactions":2`)
	require.Contains(t, s, `"totalTransactions":2`)
	require.Contains(t, s, `"tickersCount":2`)
	require.Contains(t, s, `"positionsCount":1`)
	require.Contains(t, s, `"grossValueBRLSum":"150.00"`)

	// tickers
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/b3/client/tickers?cpf=00000000000&from=2024-02-01&to=2024-03-31", nil)
	req2.Header.Set("X-Tenant-ID", tenant.String())
	r.ServeHTTP(w2, req2)
	require.Equal(t, 200, w2.Code)
	tks := w2.Body.String()
	require.Contains(t, tks, `"ticker":"ABCD3"`)
	require.Contains(t, tks, `"quantityTotal":"15"`)
}
