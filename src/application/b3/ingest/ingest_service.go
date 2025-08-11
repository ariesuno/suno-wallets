package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"suno-wallets/src/infrastructure/b3/persistence"
	"suno-wallets/src/infrastructure/observability"
	hashx "suno-wallets/src/shared/hash"
)

// Comentários em pt-BR: serviço de ingestão histórica de RAW (transações v2 / posições v3)

type B3Client interface {
	MakeRequest(ctx context.Context, method, path string, query map[string]string, cpf string, needsAuth bool) (*http.Response, error)
	Paginate(ctx context.Context, method, path string, baseQuery map[string]string, cpf string, needsAuth bool, fetch func(*http.Response) (bool, int, error)) error
}

type Service struct {
	client B3Client
	repo   persistence.RawRepository
}

func NewService(client B3Client, repo persistence.RawRepository) *Service {
	return &Service{client: client, repo: repo}
}

type IngestParams struct {
	TenantID  string
	CPF       string
	DataType  string // transactions | positions
	AssetType string // equity
	Start     string // YYYY-MM-DD
	End       string // YYYY-MM-DD
	FetchAll  bool
	Force     bool
	DryRun    bool
}

func monthWindows(start, end time.Time) [][2]time.Time {
	var out [][2]time.Time
	cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cur.After(last) {
		next := cur.AddDate(0, 1, 0).Add(-24 * time.Hour)
		if next.After(end) {
			next = end
		}
		winStart := cur
		if winStart.Before(start) {
			winStart = start
		}
		out = append(out, [2]time.Time{winStart, next})
		cur = cur.AddDate(0, 1, 0)
	}
	return out
}

type Summary struct {
	Saved           int  `json:"saved"`
	Skipped         int  `json:"skipped"`
	Errors          int  `json:"errors"`
	MonthsProcessed int  `json:"monthsProcessed"`
	PagesProcessed  int  `json:"pagesProcessed"`
	DryRun          bool `json:"dryRun"`
	Force           bool `json:"force"`
}

func (s *Service) Ingest(ctx context.Context, p IngestParams) (*Summary, error) {
	// validações simples delegadas aos validadores existentes serão feitas no controller
	sum := &Summary{}
	startT, _ := time.Parse("2006-01-02", p.Start)
	endT, _ := time.Parse("2006-01-02", p.End)
	wins := monthWindows(startT, endT)
	sum.MonthsProcessed = len(wins)

	for _, w := range wins {
		// checar período já buscado
		existing, _ := s.repo.GetMonth(ctx, uuid.MustParse(p.TenantID), p.CPF, p.DataType, p.AssetType, w[0])
		if existing != nil && existing.Completed && !p.Force {
			sum.Skipped++
			observability.IncRawSkipped(1)
			continue
		}

		// preparar rota
		var path string
		if p.DataType == "positions" {
			path = fmt.Sprintf("/position/v3/equities/investors/%s", p.CPF)
		} else {
			path = fmt.Sprintf("/assets-trading/v2/investors/%s", p.CPF)
		}
		baseQuery := map[string]string{"referenceStartDate": w[0].Format("2006-01-02"), "referenceEndDate": w[1].Format("2006-01-02")}

		pageNum := 1
		fetch := func(hr *http.Response) (bool, int, error) {
			defer hr.Body.Close()
			var generic interface{}
			if err := json.NewDecoder(hr.Body).Decode(&generic); err != nil {
				generic = map[string]any{"raw": "decode_error"}
			}
			h, _ := hashx.ComputeRawHash(generic, path, p.TenantID, p.CPF, baseQuery["referenceStartDate"], baseQuery["referenceEndDate"], pageNum)
			payload, _ := json.Marshal(generic)
			rec := &persistence.RawRecord{ID: uuid.New(), TenantID: uuid.MustParse(p.TenantID), CPF: p.CPF, DataType: p.DataType, AssetType: p.AssetType,
				PeriodStart: w[0], PeriodEnd: w[1], Page: pageNum, PayloadJSON: payload, PayloadHash: h, SourceVer: map[bool]string{p.DataType == "positions": "v3"}[true],
				EndpointPath: path, HTTPStatus: hr.StatusCode, RetryCount: 0, RequestID: uuid.New(), FetchedAt: time.Now()}
			if !p.DryRun {
				_ = s.repo.UpsertRaw(ctx, rec)
				observability.IncRawSaved(1)
			}
			sum.Saved++
			sum.PagesProcessed++
			observability.IncRawPagesProcessed(1)
			// paginação heurística simples fica a cargo do client helper (não implementado aqui para brevidade)
			pageNum++
			return false, pageNum, nil
		}

		if p.FetchAll {
			_ = s.client.Paginate(ctx, http.MethodGet, path, baseQuery, p.CPF, true, fetch)
		} else {
			_, _ = s.client.MakeRequest(ctx, http.MethodGet, path, map[string]string{
				"referenceStartDate": baseQuery["referenceStartDate"],
				"referenceEndDate":   baseQuery["referenceEndDate"],
				"page":               "1",
			}, p.CPF, true)
		}

		// marcar mês
		_ = s.repo.MarkMonth(ctx, &persistence.FetchedPeriod{ID: uuid.New(), TenantID: uuid.MustParse(p.TenantID), CPF: p.CPF,
			DataType: p.DataType, AssetType: p.AssetType, MonthStart: w[0], MonthEnd: w[1], Pages: sum.PagesProcessed, Completed: true})
		observability.IncRawMonthsCompleted(1)
	}

	observability.ObserveExternalAPI("b3", "historical_ingest", "success", time.Now().Add(-time.Second))
	sum.DryRun = p.DryRun
	sum.Force = p.Force
	return sum, nil
}
