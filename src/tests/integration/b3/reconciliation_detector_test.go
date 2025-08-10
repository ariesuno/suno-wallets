package b3

import (
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
