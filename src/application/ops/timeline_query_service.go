package ops

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// Comentários em pt-BR: service para montar consultas de timeline com keyset pagination

type TimelineItem struct {
	ID              string    `json:"id"`
	CanonicalTicker string    `json:"canonicalTicker"`
	OriginalTicker  string    `json:"originalTicker"`
	AssetType       string    `json:"assetType"`
	OperationDate   time.Time `json:"operationDate"`
	OperationType   string    `json:"operationType"`
	Source          string    `json:"source"`
	Quantity        float64   `json:"quantity"`
	UnitPrice       *float64  `json:"unitPrice"`
	Currency        string    `json:"currency"`
	ReasonCode      *string   `json:"reasonCode"`
	PriceConfidence string    `json:"priceConfidence"`
	Links           struct {
		InconsistencyID         *string `json:"inconsistencyId"`
		SupersedesOperationID   *string `json:"supersedesOperationId"`
		SupersededByOperationID *string `json:"supersededByOperationId"`
		CAID                    *string `json:"caId"`
	} `json:"links"`
	Meta struct {
		CreatedAt        time.Time `json:"createdAt"`
		UpdatedAt        time.Time `json:"updatedAt"`
		CAVersionApplied *int      `json:"caVersionApplied"`
	} `json:"meta"`
}

type TimelineFilters struct {
	CPF          string
	Tickers      []string
	From         *time.Time
	To           *time.Time
	Sources      []string
	AssetTypes   []string
	Canonicalize bool
	PageSize     int
	Cursor       *string
}

type TimelineRepository interface {
	ListTimeline(ctx context.Context, tenantID string, f TimelineFilters) ([]TimelineItem, *string, error)
}

type TimelineService struct{ repo TimelineRepository }

func NewTimelineService(repo TimelineRepository) *TimelineService {
	return &TimelineService{repo: repo}
}

func (s *TimelineService) List(ctx context.Context, tenantID string, f TimelineFilters) ([]TimelineItem, *string, error) {
	// validação básica
	if f.PageSize <= 0 {
		f.PageSize = 100
	}
	if f.PageSize > 1000 {
		f.PageSize = 1000
	}
	return s.repo.ListTimeline(ctx, tenantID, f)
}

// Helpers de cursor (operation_date|id)
func EncodeCursor(d time.Time, id string) string {
	raw := fmt.Sprintf("%s|%s", d.Format("2006-01-02"), id)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}
func DecodeCursor(cur string) (time.Time, string, error) {
	b, err := base64.StdEncoding.DecodeString(cur)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	d, err := time.Parse("2006-01-02", parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return d, parts[1], nil
}
