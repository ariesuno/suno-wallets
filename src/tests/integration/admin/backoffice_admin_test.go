package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	adminController "suno-wallets/src/api/controllers/admin"
	appadmin "suno-wallets/src/application/admin"
	infradmin "suno-wallets/src/infrastructure/admin"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test Suite para testes de integração do Backoffice Admin
type BackofficeAdminIntegrationTestSuite struct {
	suite.Suite
	db         *gorm.DB
	router     *gin.Engine
	controller *adminController.Controller
	mockB3Orch *MockB3SyncOrchestrator
	tenantID   uuid.UUID
}

// Mocks dos orchestrators para testes de integração
type MockB3SyncOrchestrator struct {
	mock.Mock
}

func (m *MockB3SyncOrchestrator) ProcessFullHistorical(ctx context.Context, tenantID, cpf, fromMonth, toMonth string, dryRun bool) error {
	args := m.Called(ctx, tenantID, cpf, fromMonth, toMonth, dryRun)
	return args.Error(0)
}

func (m *MockB3SyncOrchestrator) ProcessIncremental(ctx context.Context, tenantID, cpf, date string, dryRun bool) error {
	args := m.Called(ctx, tenantID, cpf, date, dryRun)
	return args.Error(0)
}

type MockReconciliationOrchestrator struct {
	mock.Mock
}

func (m *MockReconciliationOrchestrator) ScanInconsistencies(ctx context.Context, tenantID, cpf string, types []string, dryRun bool) error {
	args := m.Called(ctx, tenantID, cpf, types, dryRun)
	return args.Error(0)
}

func (m *MockReconciliationOrchestrator) AutoFix(ctx context.Context, tenantID, cpf string, types []string, dryRun bool) error {
	args := m.Called(ctx, tenantID, cpf, types, dryRun)
	return args.Error(0)
}

type MockDedupOrchestrator struct {
	mock.Mock
}

func (m *MockDedupOrchestrator) ScanDuplicates(ctx context.Context, tenantID, cpf string, scanMode string, dryRun bool) error {
	args := m.Called(ctx, tenantID, cpf, scanMode, dryRun)
	return args.Error(0)
}

func (m *MockDedupOrchestrator) ResolveDuplicates(ctx context.Context, tenantID string, action string, candidateIds []string) error {
	args := m.Called(ctx, tenantID, action, candidateIds)
	return args.Error(0)
}

type MockPolicyOrchestrator struct {
	mock.Mock
}

func (m *MockPolicyOrchestrator) SetPolicy(ctx context.Context, tenantID, cpf, mode, reason string) error {
	args := m.Called(ctx, tenantID, cpf, mode, reason)
	return args.Error(0)
}

type MockClientDataOrchestrator struct {
	mock.Mock
}

func (m *MockClientDataOrchestrator) ResetClient(ctx context.Context, tenantID, cpf, what string, dryRun bool) error {
	args := m.Called(ctx, tenantID, cpf, what, dryRun)
	return args.Error(0)
}

func (m *MockClientDataOrchestrator) ZeroAndRefetch(ctx context.Context, tenantID, cpf, fromDate string, dryRun bool) error {
	args := m.Called(ctx, tenantID, cpf, fromDate, dryRun)
	return args.Error(0)
}

func (suite *BackofficeAdminIntegrationTestSuite) SetupSuite() {
	// Setup do banco de dados em memória
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	suite.Require().NoError(err)
	suite.db = db

	// Criação das tabelas necessárias para os testes
	err = suite.db.Exec(`
		CREATE TYPE admin_action_type AS TEXT CHECK(admin_action_type IN (
			'B3_FULL_FETCH','B3_INCREMENTAL_FETCH','RECON_SCAN','AUTO_FIX','DEDUPE_SCAN','DEDUPE_RESOLVE',
			'POLICY_UPDATE','LEDGER_EXPORT','CACHE_INVALIDATE','CLIENT_RESET','CLIENT_ZERO_AND_REFETCH'
		));
	`).Error
	if err != nil {
		// SQLite não suporta ENUM, usar CHECK constraint
		suite.db.Exec(`DROP TYPE IF EXISTS admin_action_type`)
	}

	err = suite.db.Exec(`
		CREATE TABLE IF NOT EXISTS admin_action_audit (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			cpf VARCHAR(11) NOT NULL,
			action TEXT NOT NULL,
			status TEXT NOT NULL,
			requested_by TEXT NOT NULL,
			confirmed_by TEXT,
			confirm_token TEXT,
			confirm_deadline DATETIME,
			request_payload TEXT NOT NULL,
			result_payload TEXT,
			error_message TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`).Error
	suite.Require().NoError(err)

	err = suite.db.Exec(`
		CREATE TABLE IF NOT EXISTS b3_sync_state (
			tenant_id TEXT NOT NULL,
			cpf VARCHAR(11) NOT NULL,
			last_pos_sync_at DATETIME,
			last_tx_sync_at DATETIME,
			PRIMARY KEY (tenant_id, cpf)
		);
	`).Error
	suite.Require().NoError(err)

	err = suite.db.Exec(`
		CREATE TABLE IF NOT EXISTS b3_operations_ledger (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			cpf VARCHAR(11) NOT NULL,
			ticker TEXT NOT NULL,
			asset_type TEXT NOT NULL,
			operation_date TEXT NOT NULL,
			operation_type TEXT NOT NULL,
			source TEXT NOT NULL,
			quantity REAL NOT NULL,
			unit_price REAL,
			currency TEXT NOT NULL,
			reason_code TEXT NOT NULL,
			price_confidence TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE
		);
	`).Error
	suite.Require().NoError(err)

	// Setup dos mocks
	suite.mockB3Orch = new(MockB3SyncOrchestrator)
	mockReconOrch := new(MockReconciliationOrchestrator)
	mockDedupOrch := new(MockDedupOrchestrator)
	mockPolicyOrch := new(MockPolicyOrchestrator)
	mockClientDataOrch := new(MockClientDataOrchestrator)

	// Setup dos services e controllers
	repo := infradmin.NewRepository(suite.db, suite.mockB3Orch, mockReconOrch, mockDedupOrch, mockPolicyOrch, mockClientDataOrch)
	queryService := appadmin.NewQueryService(repo)
	actionsService := appadmin.NewActionsService(repo)
	suite.controller = adminController.NewController(queryService, actionsService)

	// Setup do router Gin
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.router.Use(func(c *gin.Context) {
		// Mock do TenantMiddleware para testes
		c.Set("tenant_id", suite.tenantID)
		c.Next()
	})

	// Configuração das rotas
	admin := suite.router.Group("/admin")
	admin.GET("/backoffice/profile", suite.controller.Profile)
	admin.GET("/backoffice/search", suite.controller.Search)
	admin.GET("/backoffice/actions", suite.controller.ListActions)
	admin.GET("/backoffice/actions/:id", suite.controller.GetAction)
	admin.GET("/backoffice/export/ledger", suite.controller.ExportLedger)
	admin.POST("/backoffice/actions/b3-full-fetch", suite.controller.B3FullFetch)
	admin.POST("/backoffice/actions/b3-full-fetch/confirm", suite.controller.ConfirmB3FullFetch)

	suite.tenantID = uuid.New()
}

func (suite *BackofficeAdminIntegrationTestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

func (suite *BackofficeAdminIntegrationTestSuite) SetupTest() {
	// Limpar dados entre testes
	suite.db.Exec("DELETE FROM admin_action_audit")
	suite.db.Exec("DELETE FROM b3_sync_state")
	suite.db.Exec("DELETE FROM b3_operations_ledger")

	// Reset dos mocks
	suite.mockB3Orch.ExpectedCalls = nil
	suite.mockB3Orch.Calls = nil
}

func (suite *BackofficeAdminIntegrationTestSuite) TestProfile_ReturnsCorrectStructure() {
	// Arrange - inserir dados de teste
	suite.db.Exec(`
		INSERT INTO b3_sync_state (tenant_id, cpf, last_pos_sync_at, last_tx_sync_at)
		VALUES (?, ?, ?, ?)
	`, suite.tenantID.String(), "12345678901", "2023-01-01 10:00:00", "2023-01-02 11:00:00")

	req := httptest.NewRequest("GET", "/admin/backoffice/profile?cpf=12345678901", nil)
	req.Header.Set("X-Tenant-ID", suite.tenantID.String())
	req.Header.Set("X-Role", "ADMIN")
	w := httptest.NewRecorder()

	// Act
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response appadmin.Profile
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), "***8901", response.CPFMasked)
	assert.Equal(suite.T(), "HYBRID", response.DataSourceMode)
	assert.NotNil(suite.T(), response.B3)
	assert.NotNil(suite.T(), response.Reconciliation)
	assert.NotNil(suite.T(), response.Performance)
	assert.NotNil(suite.T(), response.LastActions)
}

func (suite *BackofficeAdminIntegrationTestSuite) TestSearch_ReturnsMaskedCPFs() {
	// Arrange - inserir dados de teste
	suite.db.Exec(`
		INSERT INTO b3_operations_ledger (id, tenant_id, cpf, ticker, asset_type, operation_date, operation_type, source, quantity, unit_price, currency, reason_code, price_confidence)
		VALUES (?, ?, ?, 'PETR4', 'STOCK', '2023-01-01', 'BUY', 'B3_RAW', 100, 25.50, 'BRL', 'NORMAL', 'HIGH')
	`, uuid.New().String(), suite.tenantID.String(), "12345678901")

	req := httptest.NewRequest("GET", "/admin/backoffice/search?query=123&limit=10", nil)
	req.Header.Set("X-Tenant-ID", suite.tenantID.String())
	req.Header.Set("X-Role", "ADMIN")
	w := httptest.NewRecorder()

	// Act
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response struct {
		Items []map[string]interface{} `json:"items"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Len(suite.T(), response.Items, 1)
	assert.Equal(suite.T(), "***8901", response.Items[0]["cpfMasked"])
}

func (suite *BackofficeAdminIntegrationTestSuite) TestB3FullFetch_TwoStepFlow() {
	// Step 1: Request Action
	suite.mockB3Orch.On("ProcessFullHistorical", mock.Anything, suite.tenantID.String(), "12345678901", "2020-01", "2020-12", false).Return(nil)

	requestBody := map[string]interface{}{
		"cpf":    "12345678901",
		"from":   "2020-01",
		"to":     "2020-12",
		"dryRun": false,
	}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/admin/backoffice/actions/b3-full-fetch", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", suite.tenantID.String())
	req.Header.Set("X-Role", "ADMIN")
	req.Header.Set("X-User-ID", "admin@test.com")
	w := httptest.NewRecorder()

	// Act - Request
	suite.router.ServeHTTP(w, req)

	// Assert - Request
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var requestResponse struct {
		ID           string `json:"id"`
		ConfirmToken string `json:"confirmToken"`
		Action       string `json:"action"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &requestResponse)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), requestResponse.ID)
	assert.Len(suite.T(), requestResponse.ConfirmToken, 8)
	assert.Equal(suite.T(), "B3_FULL_FETCH", requestResponse.Action)

	// Aguardar um pouco para evitar condição de corrida
	time.Sleep(100 * time.Millisecond)

	// Step 2: Confirm Action
	confirmBody := map[string]interface{}{
		"id":           requestResponse.ID,
		"confirmToken": requestResponse.ConfirmToken,
	}
	jsonConfirmBody, _ := json.Marshal(confirmBody)

	confirmReq := httptest.NewRequest("POST", "/admin/backoffice/actions/b3-full-fetch/confirm", bytes.NewBuffer(jsonConfirmBody))
	confirmReq.Header.Set("Content-Type", "application/json")
	confirmReq.Header.Set("X-Tenant-ID", suite.tenantID.String())
	confirmReq.Header.Set("X-Role", "ADMIN")
	confirmW := httptest.NewRecorder()

	// Act - Confirm
	suite.router.ServeHTTP(confirmW, confirmReq)

	// Assert - Confirm
	assert.Equal(suite.T(), http.StatusOK, confirmW.Code)

	var confirmResponse struct {
		Status string `json:"status"`
		Action string `json:"action"`
	}
	err = json.Unmarshal(confirmW.Body.Bytes(), &confirmResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "confirmed", confirmResponse.Status)
	assert.Equal(suite.T(), "B3_FULL_FETCH", confirmResponse.Action)

	// Aguardar execução assíncrona
	time.Sleep(200 * time.Millisecond)

	// Verificar que o orchestrator foi chamado
	suite.mockB3Orch.AssertExpectations(suite.T())
}

func (suite *BackofficeAdminIntegrationTestSuite) TestRBACProtection() {
	requestBody := map[string]interface{}{
		"cpf":    "12345678901",
		"from":   "2020-01",
		"to":     "2020-12",
		"dryRun": false,
	}
	jsonBody, _ := json.Marshal(requestBody)

	// Test SUPPORT role (should be allowed for request)
	req := httptest.NewRequest("POST", "/admin/backoffice/actions/b3-full-fetch", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", suite.tenantID.String())
	req.Header.Set("X-Role", "SUPPORT")
	req.Header.Set("X-User-ID", "support@test.com")
	w := httptest.NewRecorder()

	// Act
	suite.router.ServeHTTP(w, req)

	// Assert - SUPPORT can request
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Test confirmation with SUPPORT role (should be forbidden)
	var requestResponse struct {
		ID           string `json:"id"`
		ConfirmToken string `json:"confirmToken"`
	}
	json.Unmarshal(w.Body.Bytes(), &requestResponse)

	confirmBody := map[string]interface{}{
		"id":           requestResponse.ID,
		"confirmToken": requestResponse.ConfirmToken,
	}
	jsonConfirmBody, _ := json.Marshal(confirmBody)

	confirmReq := httptest.NewRequest("POST", "/admin/backoffice/actions/b3-full-fetch/confirm", bytes.NewBuffer(jsonConfirmBody))
	confirmReq.Header.Set("Content-Type", "application/json")
	confirmReq.Header.Set("X-Tenant-ID", suite.tenantID.String())
	confirmReq.Header.Set("X-Role", "SUPPORT")
	confirmW := httptest.NewRecorder()

	// Act
	suite.router.ServeHTTP(confirmW, confirmReq)

	// Assert - SUPPORT cannot confirm
	assert.Equal(suite.T(), http.StatusForbidden, confirmW.Code)
}

func (suite *BackofficeAdminIntegrationTestSuite) TestIdempotency() {
	suite.mockB3Orch.On("ProcessFullHistorical", mock.Anything, suite.tenantID.String(), "12345678901", "2020-01", "2020-12", false).Return(nil).Once()

	// Step 1: Request Action
	requestBody := map[string]interface{}{
		"cpf":    "12345678901",
		"from":   "2020-01",
		"to":     "2020-12",
		"dryRun": false,
	}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/admin/backoffice/actions/b3-full-fetch", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", suite.tenantID.String())
	req.Header.Set("X-Role", "ADMIN")
	req.Header.Set("X-User-ID", "admin@test.com")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var requestResponse struct {
		ID           string `json:"id"`
		ConfirmToken string `json:"confirmToken"`
	}
	json.Unmarshal(w.Body.Bytes(), &requestResponse)

	time.Sleep(100 * time.Millisecond)

	// Step 2: Confirm primeira vez
	confirmBody := map[string]interface{}{
		"id":           requestResponse.ID,
		"confirmToken": requestResponse.ConfirmToken,
	}
	jsonConfirmBody, _ := json.Marshal(confirmBody)

	confirmReq := httptest.NewRequest("POST", "/admin/backoffice/actions/b3-full-fetch/confirm", bytes.NewBuffer(jsonConfirmBody))
	confirmReq.Header.Set("Content-Type", "application/json")
	confirmReq.Header.Set("X-Tenant-ID", suite.tenantID.String())
	confirmReq.Header.Set("X-Role", "ADMIN")
	confirmW := httptest.NewRecorder()

	suite.router.ServeHTTP(confirmW, confirmReq)
	assert.Equal(suite.T(), http.StatusOK, confirmW.Code)

	time.Sleep(200 * time.Millisecond)

	// Step 3: Confirm segunda vez (idempotente)
	confirmReq2 := httptest.NewRequest("POST", "/admin/backoffice/actions/b3-full-fetch/confirm", bytes.NewBuffer(jsonConfirmBody))
	confirmReq2.Header.Set("Content-Type", "application/json")
	confirmReq2.Header.Set("X-Tenant-ID", suite.tenantID.String())
	confirmReq2.Header.Set("X-Role", "ADMIN")
	confirmW2 := httptest.NewRecorder()

	suite.router.ServeHTTP(confirmW2, confirmReq2)
	assert.Equal(suite.T(), http.StatusOK, confirmW2.Code)

	// Aguardar e verificar que orchestrator foi chamado apenas uma vez
	time.Sleep(200 * time.Millisecond)
	suite.mockB3Orch.AssertExpectations(suite.T())
}

func (suite *BackofficeAdminIntegrationTestSuite) TestDryRunDoesNotAlterState() {
	// Mock deve ser chamado com dryRun=true
	suite.mockB3Orch.On("ProcessFullHistorical", mock.Anything, suite.tenantID.String(), "12345678901", "2020-01", "2020-12", true).Return(nil)

	// Step 1: Request com dry run
	requestBody := map[string]interface{}{
		"cpf":    "12345678901",
		"from":   "2020-01",
		"to":     "2020-12",
		"dryRun": true, // DRY RUN MODE
	}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/admin/backoffice/actions/b3-full-fetch", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", suite.tenantID.String())
	req.Header.Set("X-Role", "ADMIN")
	req.Header.Set("X-User-ID", "admin@test.com")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var requestResponse struct {
		ID           string `json:"id"`
		ConfirmToken string `json:"confirmToken"`
	}
	json.Unmarshal(w.Body.Bytes(), &requestResponse)

	time.Sleep(100 * time.Millisecond)

	// Step 2: Confirm
	confirmBody := map[string]interface{}{
		"id":           requestResponse.ID,
		"confirmToken": requestResponse.ConfirmToken,
	}
	jsonConfirmBody, _ := json.Marshal(confirmBody)

	confirmReq := httptest.NewRequest("POST", "/admin/backoffice/actions/b3-full-fetch/confirm", bytes.NewBuffer(jsonConfirmBody))
	confirmReq.Header.Set("Content-Type", "application/json")
	confirmReq.Header.Set("X-Tenant-ID", suite.tenantID.String())
	confirmReq.Header.Set("X-Role", "ADMIN")
	confirmW := httptest.NewRecorder()

	suite.router.ServeHTTP(confirmW, confirmReq)
	assert.Equal(suite.T(), http.StatusOK, confirmW.Code)

	time.Sleep(200 * time.Millisecond)

	// Verificar que orchestrator foi chamado com dryRun=true
	suite.mockB3Orch.AssertExpectations(suite.T())
}

// Executar a suite de testes
func TestBackofficeAdminIntegrationSuite(t *testing.T) {
	suite.Run(t, new(BackofficeAdminIntegrationTestSuite))
}
