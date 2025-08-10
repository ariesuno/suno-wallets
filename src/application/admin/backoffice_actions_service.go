package admin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Comentários em pt-BR: service de ações com REQUESTED → CONFIRMED, confirmToken e TTL

type ActionRequest struct {
	ID              uuid.UUID
	TenantID        string
	CPF             string
	Action          string
	RequestedBy     string
	ConfirmToken    string
	ConfirmDeadline time.Time
	Payload         map[string]interface{}
}

type ActionsRepository interface {
	CreateRequest(ctx context.Context, ar ActionRequest) (uuid.UUID, error)
	ConfirmAndRun(ctx context.Context, id uuid.UUID, confirmToken string) error
}

type ActionsService struct{ repo ActionsRepository }

func NewActionsService(r ActionsRepository) *ActionsService { return &ActionsService{repo: r} }

func (s *ActionsService) Request(ctx context.Context, tenantID, cpf, action, requestedBy string, ttlSeconds int, payload map[string]interface{}) (uuid.UUID, string, error) {
	id := uuid.New()
	token := uuid.New().String()[0:8]
	ar := ActionRequest{ID: id, TenantID: tenantID, CPF: cpf, Action: action, RequestedBy: requestedBy, ConfirmToken: token, ConfirmDeadline: time.Now().Add(time.Duration(ttlSeconds) * time.Second), Payload: payload}
	_, err := s.repo.CreateRequest(ctx, ar)
	return id, token, err
}

func (s *ActionsService) Confirm(ctx context.Context, id uuid.UUID, token string) error {
	return s.repo.ConfirmAndRun(ctx, id, token)
}

func toJSON(m map[string]interface{}) string { b, _ := json.Marshal(m); return string(b) }
