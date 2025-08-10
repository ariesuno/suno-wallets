package b3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	b3 "suno-wallets/src/api/controllers/b3"
	"suno-wallets/src/api/middlewares"
	appe2e "suno-wallets/src/application/b3/e2e"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fakeResetRepo implementa appe2e.ResetRepository para o caminho dryRun (apenas lock)
type fakeResetRepoDry struct{}

func (r *fakeResetRepoDry) TryAcquireLock(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return true, nil
}
func (r *fakeResetRepoDry) ReleaseLock(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *fakeResetRepoDry) Reset(_ context.Context, _ uuid.UUID, _ string, _ string, _ string) (*appe2e.ResetResult, error) {
	return &appe2e.ResetResult{}, nil
}

func TestAdminResetAndRefetch_DryRun_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// garantir que não há exigência de X-Admin-Secret no teste
	_ = os.Setenv("ADMIN_SECRET", "")

	// orquestrador com fake repo (lock ok) e sem dependências downstream
	repo := &fakeResetRepoDry{}
	orch := appe2e.NewOrchestratorPorts(repo, nil, nil)
	ctrl := b3.NewAdminController(orch, nil)

	r := gin.New()
	r.Use(middlewares.TenantMiddleware())
	r.POST("/api/v1/b3/admin/reset-and-refetch", ctrl.ResetAndRefetch)

	tenant := uuid.New()
	body := map[string]any{
		"cpf":     "00000000000",
		"dryRun":  true,
		"confirm": "RESET_AND_REFETCH",
		"mode":    "archive",
		// assetTypes/dataTypes podem ficar vazios para defaults
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/b3/admin/reset-and-refetch", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant.String())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "\"dryRun\":true")
	require.Contains(t, w.Body.String(), "\"monthsProcessed\"")
}
