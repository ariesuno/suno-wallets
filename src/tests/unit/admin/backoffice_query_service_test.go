package admin_test

import (
	"context"
	"testing"

	appadmin "suno-wallets/src/application/admin"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock do Repository para testes unitários
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) LoadProfile(ctx context.Context, tenantID, cpf string) (*appadmin.Profile, error) {
	args := m.Called(ctx, tenantID, cpf)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*appadmin.Profile), args.Error(1)
}

func (m *MockRepository) Search(ctx context.Context, tenantID, query string, limit int) ([]map[string]interface{}, error) {
	args := m.Called(ctx, tenantID, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockRepository) ListActions(ctx context.Context, tenantID, cpf, action, status string, page, pageSize int) ([]map[string]interface{}, error) {
	args := m.Called(ctx, tenantID, cpf, action, status, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockRepository) GetAction(ctx context.Context, tenantID string, id uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockRepository) ExportLedger(ctx context.Context, tenantID, cpf string, excludeB3 bool, limit int) ([]map[string]interface{}, error) {
	args := m.Called(ctx, tenantID, cpf, excludeB3, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockRepository) ExportLedgerStream(ctx context.Context, tenantID, cpf string, excludeB3 bool, limit int, callback func([]map[string]interface{}) error) error {
	args := m.Called(ctx, tenantID, cpf, excludeB3, limit, callback)
	return args.Error(0)
}

func TestQueryService_Profile(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := appadmin.NewQueryService(mockRepo)

	expectedProfile := &appadmin.Profile{
		CPFMasked:      "***1234",
		DataSourceMode: "HYBRID",
		B3:             map[string]interface{}{"lastFullFetchAt": nil},
		Reconciliation: map[string]interface{}{"inconsistencies": map[string]interface{}{"open": 0}},
		Performance:    map[string]interface{}{"avgFullFetchSec": nil},
		LastActions:    []map[string]interface{}{},
	}

	mockRepo.On("LoadProfile", mock.Anything, "tenant1", "12345678901").Return(expectedProfile, nil)

	// Act
	result, err := service.Profile(context.Background(), "tenant1", "12345678901")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedProfile, result)
	assert.Equal(t, "***1234", result.CPFMasked)
	assert.Equal(t, "HYBRID", result.DataSourceMode)

	mockRepo.AssertExpectations(t)
}

func TestQueryService_Search(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := appadmin.NewQueryService(mockRepo)

	expectedResults := []map[string]interface{}{
		{"cpfMasked": "***1234"},
		{"cpfMasked": "***5678"},
	}

	mockRepo.On("Search", mock.Anything, "tenant1", "123", 20).Return(expectedResults, nil)

	// Act
	result, err := service.Search(context.Background(), "tenant1", "123", 20)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedResults, result)
	assert.Len(t, result, 2)

	mockRepo.AssertExpectations(t)
}

func TestQueryService_ListActions(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := appadmin.NewQueryService(mockRepo)

	actionID := uuid.New()
	expectedActions := []map[string]interface{}{
		{
			"id":          actionID,
			"action":      "B3_FULL_FETCH",
			"status":      "SUCCESS",
			"requestedBy": "admin@test.com",
			"createdAt":   "2023-01-01T10:00:00Z",
		},
	}

	mockRepo.On("ListActions", mock.Anything, "tenant1", "12345678901", "", "", 1, 50).Return(expectedActions, nil)

	// Act
	result, err := service.ListActions(context.Background(), "tenant1", "12345678901", "", "", 1, 50)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedActions, result)
	assert.Len(t, result, 1)

	mockRepo.AssertExpectations(t)
}

func TestQueryService_GetAction(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := appadmin.NewQueryService(mockRepo)

	actionID := uuid.New()
	expectedAction := map[string]interface{}{
		"id":             actionID,
		"action":         "B3_FULL_FETCH",
		"status":         "SUCCESS",
		"requestedBy":    "admin@test.com",
		"requestPayload": `{"cpf":"12345678901","from":"2020-01","to":"2020-12"}`,
		"resultPayload":  `{"result":"SUCCESS","recordsProcessed":1500}`,
	}

	mockRepo.On("GetAction", mock.Anything, "tenant1", actionID).Return(expectedAction, nil)

	// Act
	result, err := service.GetAction(context.Background(), "tenant1", actionID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedAction, result)
	assert.Equal(t, "B3_FULL_FETCH", result["action"])
	assert.Equal(t, "SUCCESS", result["status"])

	mockRepo.AssertExpectations(t)
}

func TestQueryService_ExportLedger(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := appadmin.NewQueryService(mockRepo)

	expectedLedger := []map[string]interface{}{
		{
			"id":            "op1",
			"ticker":        "PETR4",
			"assetType":     "STOCK",
			"operationType": "BUY",
			"quantity":      100.0,
			"unitPrice":     25.50,
		},
		{
			"id":            "op2",
			"ticker":        "VALE3",
			"assetType":     "STOCK",
			"operationType": "SELL",
			"quantity":      50.0,
			"unitPrice":     85.75,
		},
	}

	mockRepo.On("ExportLedger", mock.Anything, "tenant1", "12345678901", false, 50000).Return(expectedLedger, nil)

	// Act
	result, err := service.ExportLedger(context.Background(), "tenant1", "12345678901", false, 50000)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedLedger, result)
	assert.Len(t, result, 2)

	// Verifica estrutura dos dados
	assert.Equal(t, "PETR4", result[0]["ticker"])
	assert.Equal(t, "BUY", result[0]["operationType"])
	assert.Equal(t, 100.0, result[0]["quantity"])

	mockRepo.AssertExpectations(t)
}
