package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"suno-wallets/src/domain/enums"
	b3err "suno-wallets/src/infrastructure/b3/errors"
	"suno-wallets/src/infrastructure/b3/persistence"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared"
	hashx "suno-wallets/src/shared/hash"
	"suno-wallets/src/shared/helpers"
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

// parseAssetType converte string para B3AssetType, com fallback para equity
func (s *Service) parseAssetType(assetTypeStr string) enums.B3AssetType {
	// Normalizar entrada
	normalized := strings.ToLower(strings.TrimSpace(assetTypeStr))

	// Mapear strings conhecidas para B3AssetType
	switch normalized {
	case "equity", "equities", "stock", "stocks":
		return enums.B3AssetTypeEquities
	case "fixed-income", "fixed_income", "fixedincome", "bond", "bonds":
		return enums.B3AssetTypeFixedIncome
	case "treasury-bonds", "treasury_bonds", "treasurybonds", "tesouro", "treasury":
		return enums.B3AssetTypeTreasuryBonds
	case "derivatives", "derivative", "options", "futures":
		return enums.B3AssetTypeDerivatives
	case "securities-lending", "securities_lending", "securitieslending", "emprestimo":
		return enums.B3AssetTypeSecuritiesLending
	default:
		// Tentar parse direto
		if b3Type, valid := enums.ParseB3AssetType(normalized); valid {
			return b3Type
		}
		// Fallback para equities (compatibilidade)
		return enums.B3AssetTypeEquities
	}
}

type IngestParams struct {
	TenantID  string
	CPF       string
	DataType  string // transactions | positions
	AssetType string // equity (legacy), ou B3AssetType específico
	Start     string // YYYY-MM-DD
	End       string // YYYY-MM-DD
	FetchAll  bool
	Force     bool
	DryRun    bool
}

// monthWindows gera janelas mensais usando a nova lógica B3 com timezone e mês corrente
func monthWindows(start, end time.Time) [][2]time.Time {
	// Usar nova lógica de janelas B3 com timezone America/Sao_Paulo
	windows := shared.GenerateB3MonthlyWindows(start, end)
	return shared.ConvertToTimeWindows(windows)
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

	// Debug crítico para investigação de endpoints
	helpers.LogInfo("🔍 INGEST DEBUG: Starting ingest", map[string]interface{}{
		"asset_type": p.AssetType,
		"data_type":  p.DataType,
		"cpf_masked": p.CPF[:3] + "***",
		"start":      p.Start,
		"end":        p.End,
	})
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

		// preparar rota baseada no tipo de ativo
		var path string
		b3AssetType := s.parseAssetType(p.AssetType)

		// Debug crítico para verificar parsing e endpoint
		helpers.LogInfo("🔍 INGEST DEBUG: Asset type parsing", map[string]interface{}{
			"original_asset_type": p.AssetType,
			"parsed_b3_type":      string(b3AssetType),
			"data_type":           p.DataType,
		})

		if p.DataType == "positions" {
			path = b3AssetType.GetPositionsEndpoint() + "/" + p.CPF
		} else {
			// CORREÇÃO: Usar endpoints específicos para TODOS os tipos
			// Não mais fallback para v2 - usar sempre endpoints específicos
			path = b3AssetType.GetAPIEndpoint() + "/" + p.CPF
		}

		// Debug crítico para verificar endpoint gerado
		helpers.LogInfo("🔍 INGEST DEBUG: Generated endpoint", map[string]interface{}{
			"generated_path": path,
			"asset_type":     string(b3AssetType),
			"data_type":      p.DataType,
			"cpf_masked":     p.CPF[:3] + "***",
		})
		baseQuery := map[string]string{"referenceStartDate": w[0].Format("2006-01-02"), "referenceEndDate": w[1].Format("2006-01-02")}

		pageNum := 1
		fetch := func(hr *http.Response) (bool, int, error) {
			defer func() { _ = hr.Body.Close() }()
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

		// marcar mês com period_start e period_end precisos
		if w[0].Year() == 2025 && w[0].Month() == 8 {
			fmt.Printf("DEBUG: About to mark August period [%s, %s]\n", w[0].Format("2006-01-02"), w[1].Format("2006-01-02"))
		}

		// Converter para datas para month_start/month_end (compatibilidade)
		monthStart := time.Date(w[0].Year(), w[0].Month(), 1, 0, 0, 0, 0, time.UTC)
		monthEnd := time.Date(w[1].Year(), w[1].Month(), w[1].Day(), 0, 0, 0, 0, time.UTC)

		_ = s.repo.MarkMonth(ctx, &persistence.FetchedPeriod{
			ID:          uuid.New(),
			TenantID:    p.TenantID,
			CPF:         p.CPF,
			DataType:    p.DataType,
			AssetType:   p.AssetType,
			MonthStart:  monthStart,
			MonthEnd:    monthEnd,
			PeriodStart: &w[0], // timestamp preciso início
			PeriodEnd:   &w[1], // timestamp preciso fim
			Pages:       sum.PagesProcessed,
			Completed:   true,
		})
		observability.IncRawMonthsCompleted(1)
	}

	observability.ObserveExternalAPI("b3", "historical_ingest", "success", time.Now().Add(-time.Second))
	sum.DryRun = p.DryRun
	sum.Force = p.Force
	return sum, nil
}
