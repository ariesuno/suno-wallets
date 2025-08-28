package admin_test

import (
	"context"
	"testing"
	"time"

	appadmin "suno-wallets/src/application/admin"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock do ActionsRepository para testes unitários
type MockActionsRepository struct {
	mock.Mock
}

func (m *MockActionsRepository) CreateRequest(ctx context.Context, ar appadmin.ActionRequest) (uuid.UUID, error) {
	args := m.Called(ctx, ar)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockActionsRepository) ConfirmAndRun(ctx context.Context, id uuid.UUID, confirmToken string) error {
	args := m.Called(ctx, id, confirmToken)
	return args.Error(0)
}

func TestActionsService_Request(t *testing.T) {
	// Arrange
	mockRepo := new(MockActionsRepository)
	service := appadmin.NewActionsService(mockRepo)

	payload := map[string]interface{}{
		"cpf":    "12345678901",
		"from":   "2020-01",
		"to":     "2020-12",
		"dryRun": false,
	}

	mockRepo.On("CreateRequest", mock.Anything, mock.MatchedBy(func(ar appadmin.ActionRequest) bool {
		return ar.TenantID == "tenant1" &&
			ar.CPF == "12345678901" &&
			ar.Action == "B3_FULL_FETCH" &&
			ar.RequestedBy == "admin@test.com" &&
			len(ar.ConfirmToken) == 8 &&
			ar.ConfirmDeadline.After(time.Now())
	})).Return(uuid.New(), nil)

	// Act
	id, token, err := service.Request(
		context.Background(),
		"tenant1",
		"12345678901",
		"B3_FULL_FETCH",
		"admin@test.com",
		600,
		payload,
	)

	// Assert
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id) // Apenas verifica se não é nil
	assert.Len(t, token, 8)
	assert.NotEmpty(t, token)

	mockRepo.AssertExpectations(t)
}

func TestActionsService_Request_GeneratesUniqueTokens(t *testing.T) {
	// Arrange
	mockRepo := new(MockActionsRepository)
	service := appadmin.NewActionsService(mockRepo)

	id1, id2 := uuid.New(), uuid.New()
	payload := map[string]interface{}{"cpf": "12345678901"}

	mockRepo.On("CreateRequest", mock.Anything, mock.AnythingOfType("admin.ActionRequest")).Return(id1, nil).Once()
	mockRepo.On("CreateRequest", mock.Anything, mock.AnythingOfType("admin.ActionRequest")).Return(id2, nil).Once()

	// Act - criar duas requisições
	_, token1, err1 := service.Request(context.Background(), "tenant1", "12345678901", "B3_FULL_FETCH", "admin@test.com", 600, payload)
	_, token2, err2 := service.Request(context.Background(), "tenant1", "12345678901", "B3_INCREMENTAL_FETCH", "admin@test.com", 600, payload)

	// Assert - tokens devem ser diferentes
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, token1, token2)
	assert.Len(t, token1, 8)
	assert.Len(t, token2, 8)

	mockRepo.AssertExpectations(t)
}

func TestActionsService_Request_CustomTTL(t *testing.T) {
	// Arrange
	mockRepo := new(MockActionsRepository)
	service := appadmin.NewActionsService(mockRepo)

	customTTL := 300 // 5 minutes
	payload := map[string]interface{}{"cpf": "12345678901"}

	mockRepo.On("CreateRequest", mock.Anything, mock.MatchedBy(func(ar appadmin.ActionRequest) bool {
		// Verifica que o deadline está aproximadamente 5 minutos no futuro
		expectedDeadline := time.Now().Add(time.Duration(customTTL) * time.Second)
		deadlineDiff := ar.ConfirmDeadline.Sub(expectedDeadline)
		return deadlineDiff < time.Second && deadlineDiff > -time.Second // tolerância de 1 segundo
	})).Return(uuid.New(), nil)

	// Act
	id, token, err := service.Request(
		context.Background(),
		"tenant1",
		"12345678901",
		"RECON_SCAN",
		"admin@test.com",
		customTTL,
		payload,
	)

	// Assert
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id) // Apenas verifica se não é nil
	assert.NotEmpty(t, token)

	mockRepo.AssertExpectations(t)
}

func TestActionsService_Confirm(t *testing.T) {
	// Arrange
	mockRepo := new(MockActionsRepository)
	service := appadmin.NewActionsService(mockRepo)

	actionID := uuid.New()
	token := "abc12345"

	mockRepo.On("ConfirmAndRun", mock.Anything, actionID, token).Return(nil)

	// Act
	err := service.Confirm(context.Background(), actionID, token)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestActionsService_Confirm_InvalidToken(t *testing.T) {
	// Arrange
	mockRepo := new(MockActionsRepository)
	service := appadmin.NewActionsService(mockRepo)

	actionID := uuid.New()
	token := "invalid"
	expectedError := assert.AnError

	mockRepo.On("ConfirmAndRun", mock.Anything, actionID, token).Return(expectedError)

	// Act
	err := service.Confirm(context.Background(), actionID, token)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)

	mockRepo.AssertExpectations(t)
}

func TestActionsService_Request_PayloadPreservation(t *testing.T) {
	// Arrange
	mockRepo := new(MockActionsRepository)
	service := appadmin.NewActionsService(mockRepo)

	complexPayload := map[string]interface{}{
		"cpf":    "12345678901",
		"from":   "2020-01",
		"to":     "2020-12",
		"dryRun": true,
		"types":  []string{"OPENING_BALANCE_MISSING", "SELL_WITHOUT_BUY"},
		"metadata": map[string]interface{}{
			"source":    "admin-panel",
			"priority":  "high",
			"batchSize": 1000,
		},
	}

	mockRepo.On("CreateRequest", mock.Anything, mock.MatchedBy(func(ar appadmin.ActionRequest) bool {
		// Verifica que o payload foi preservado corretamente
		return ar.Payload != nil &&
			ar.Payload["cpf"] == "12345678901" &&
			ar.Payload["dryRun"] == true &&
			len(ar.Payload["types"].([]string)) == 2
	})).Return(uuid.New(), nil)

	// Act
	id, token, err := service.Request(
		context.Background(),
		"tenant1",
		"12345678901",
		"AUTO_FIX",
		"admin@test.com",
		600,
		complexPayload,
	)

	// Assert
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id) // Apenas verifica se não é nil
	assert.NotEmpty(t, token)

	mockRepo.AssertExpectations(t)
}
