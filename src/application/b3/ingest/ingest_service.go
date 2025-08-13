package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	b3err "suno-wallets/src/infrastructure/b3/errors"
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

	var wins [][2]time.Time

	// Para posições, buscar apenas o dia anterior (otimização)
	if p.DataType == "positions" {
		// Para posições, usar apenas o dia anterior
		yesterday := endT.AddDate(0, 0, -1)
		wins = [][2]time.Time{{yesterday, yesterday}}
	} else {
		// Para transações, usar histórico completo mês a mês
		fmt.Printf("DEBUG: monthWindows input - start=%s, end=%s\n", startT.Format("2006-01-02"), endT.Format("2006-01-02"))
		wins = monthWindows(startT, endT)
		fmt.Printf("DEBUG: monthWindows output - generated %d windows\n", len(wins))
		if len(wins) > 0 {
			fmt.Printf("DEBUG: First window: [%s, %s]\n", wins[0][0].Format("2006-01-02"), wins[0][1].Format("2006-01-02"))
			fmt.Printf("DEBUG: Last window: [%s, %s]\n", wins[len(wins)-1][0].Format("2006-01-02"), wins[len(wins)-1][1].Format("2006-01-02"))
		}
	}

	sum.MonthsProcessed = len(wins)

	for _, w := range wins {
		// Debug para verificar todas as janelas
		if w[0].Year() == 2025 && w[0].Month() == 8 {
			fmt.Printf("DEBUG: Found August window [%s, %s]\n", w[0].Format("2006-01-02"), w[1].Format("2006-01-02"))
		}

		// checar período já buscado
		existing, _ := s.repo.GetMonth(ctx, p.TenantID, p.CPF, p.DataType, p.AssetType, w[0])
		if existing != nil && existing.Completed && !p.Force {
			if w[0].Year() == 2025 && w[0].Month() == 8 {
				fmt.Printf("DEBUG: August period already exists - skipping\n")
			}
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
			rec := &persistence.RawRecord{ID: uuid.New(), TenantID: p.TenantID, CPF: p.CPF, DataType: p.DataType, AssetType: p.AssetType,
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
			if err := s.client.Paginate(ctx, http.MethodGet, path, baseQuery, p.CPF, true, fetch); err != nil {
				// Se HTTP 422, cliente não tem dados neste período - continuar e marcar período
				if b3err.IsStatusCode(err, 422) {
					observability.IncRawSkipped(1)
					// Continuar para marcar período como consultado
				} else {
					// Outros erros são falhas reais de conectividade/autenticação
					sum.Errors++
					observability.IncRawErrors(1)
					return sum, fmt.Errorf("failed to fetch data for period %s: %w", w[0].Format("2006-01-02"), err)
				}
			}
		} else {
			if _, err := s.client.MakeRequest(ctx, http.MethodGet, path, map[string]string{
				"referenceStartDate": baseQuery["referenceStartDate"],
				"referenceEndDate":   baseQuery["referenceEndDate"],
				"page":               "1",
			}, p.CPF, true); err != nil {
				// Se HTTP 422, cliente não tem dados neste período - continuar e marcar período
				if b3err.IsStatusCode(err, 422) {
					observability.IncRawSkipped(1)
					// Continuar para marcar período como consultado
				} else {
					// Outros erros são falhas reais de conectividade/autenticação
					sum.Errors++
					observability.IncRawErrors(1)
					return sum, fmt.Errorf("failed to fetch data for period %s: %w", w[0].Format("2006-01-02"), err)
				}
			}
		}

		// marcar mês
		if w[0].Year() == 2025 && w[0].Month() == 8 {
			fmt.Printf("DEBUG: About to mark August period [%s, %s]\n", w[0].Format("2006-01-02"), w[1].Format("2006-01-02"))
		}
		_ = s.repo.MarkMonth(ctx, &persistence.FetchedPeriod{ID: uuid.New(), TenantID: p.TenantID, CPF: p.CPF,
			DataType: p.DataType, AssetType: p.AssetType, MonthStart: w[0], MonthEnd: w[1], Pages: sum.PagesProcessed, Completed: true})
		observability.IncRawMonthsCompleted(1)
	}

	observability.ObserveExternalAPI("b3", "historical_ingest", "success", time.Now().Add(-time.Second))
	sum.DryRun = p.DryRun
	sum.Force = p.Force
	return sum, nil
}

// maskCPF mascara CPF mantendo apenas os 3 primeiros dígitos
func maskCPF(cpf string) string {
	if len(cpf) < 3 {
		return "***"
	}
	return cpf[:3] + "********"
}
