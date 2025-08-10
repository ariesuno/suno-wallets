package observability_test

import (
	"net/http/httptest"
	"testing"

	obsctl "suno-wallets/src/api/controllers/observability"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// helper para router de health
func routerWithHealth(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := obsctl.NewHealthController(db, nil)
	r.GET("/health/live", h.Live)
	r.GET("/health/ready", h.Ready)
	r.GET("/health/details", h.Details)
	return r
}

func TestHealthLive_OK(t *testing.T) {
	r := routerWithHealth(nil)

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"version"`)
	require.Contains(t, w.Body.String(), `"commit"`)
	require.Contains(t, w.Body.String(), `"uptimeSec"`)
}

func TestHealthReady_503_WhenMigrationsMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	r := routerWithHealth(db)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 503, w.Code)
	require.Contains(t, w.Body.String(), `"migrations"`)
}

func TestHealthReady_200_WhenDBAndMigrationsOK(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// cria schema mínimo (wallets) para CheckMigrations
	db.Exec(`CREATE TABLE wallets (id TEXT PRIMARY KEY);`)

	r := routerWithHealth(db)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"status":"UP"`)
	require.Contains(t, w.Body.String(), `"migrations"`)
}

func TestHealthDetails_ReturnsComponentsAndBuild(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	db.Exec(`CREATE TABLE wallets (id TEXT PRIMARY KEY);`)

	r := routerWithHealth(db)

	req := httptest.NewRequest("GET", "/health/details", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"components"`)
	require.Contains(t, w.Body.String(), `"build"`)
}
