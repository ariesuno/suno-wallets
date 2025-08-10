package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	b3Client "suno-wallets/src/infrastructure/b3/client"
	"suno-wallets/src/shared/helpers"
)

// Comentários em pt-BR: service para testar conexão direta com B3

type Service struct {
	b3Client *b3Client.B3OfficialClient
	// Usar helper functions ao invés de logger struct
}

type TestParams struct {
	TenantID  uuid.UUID
	CPF       string
	StartDate time.Time
	EndDate   time.Time
}

type TestReport struct {
	ConnectionTest ConnectionTestResult `json:"connectionTest"`
	Transactions   TransactionsSummary  `json:"transactions"`
	Positions      PositionsSummary     `json:"positions"`
	Summary        OverallSummary       `json:"summary"`
}

type ConnectionTestResult struct {
	AuthenticationOK bool   `json:"authenticationOK"`
	APIReachable     bool   `json:"apiReachable"`
	ResponseTime     string `json:"responseTime"`
	Error            string `json:"error,omitempty"`
}

type TransactionsSummary struct {
	TotalRecords    int                    `json:"totalRecords"`
	TotalPages      int                    `json:"totalPages"`
	SampleRecords   []interface{}          `json:"sampleRecords"`
	AssetTypes      map[string]int         `json:"assetTypes"`
	OperationTypes  map[string]int         `json:"operationTypes"`
	DateRange       DateRangeInfo          `json:"dateRange"`
	Tickers         []string               `json:"tickers"`
	Error           string                 `json:"error,omitempty"`
}

type PositionsSummary struct {
	TotalRecords  int                    `json:"totalRecords"`
	TotalPages    int                    `json:"totalPages"`
	SampleRecords []interface{}          `json:"sampleRecords"`
	AssetTypes    map[string]int         `json:"assetTypes"`
	DateRange     DateRangeInfo          `json:"dateRange"`
	Tickers       []string               `json:"tickers"`
	Error         string                 `json:"error,omitempty"`
}

type DateRangeInfo struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Earliest  string `json:"earliest,omitempty"`
	Latest    string `json:"latest,omitempty"`
}

type OverallSummary struct {
	TotalTransactions int      `json:"totalTransactions"`
	TotalPositions    int      `json:"totalPositions"`
	UniqueTickers     []string `json:"uniqueTickers"`
	TestDuration      string   `json:"testDuration"`
	DataQuality       string   `json:"dataQuality"`
}

func NewService(b3Client *b3Client.B3OfficialClient) *Service {
	return &Service{
		b3Client: b3Client,
	}
}

func (s *Service) TestConnectionAndReport(ctx context.Context, params TestParams) (*TestReport, error) {
	startTime := time.Now()
	
	helpers.LogInfo("Iniciando teste de conexão B3", map[string]interface{}{
		"cpf":       maskCPF(params.CPF),
		"tenantId":  params.TenantID,
		"dateRange": fmt.Sprintf("%s a %s", params.StartDate.Format("2006-01-02"), params.EndDate.Format("2006-01-02")),
	})

	report := &TestReport{}

	// 1. Teste de conexão e autenticação
	connectionResult := s.testConnection(ctx)
	report.ConnectionTest = connectionResult

	if !connectionResult.AuthenticationOK {
		return report, fmt.Errorf("falha na autenticação B3: %s", connectionResult.Error)
	}

	// 2. Teste de transações
	transactionsResult := s.testTransactions(ctx, params)
	report.Transactions = transactionsResult

	// 3. Teste de posições
	positionsResult := s.testPositions(ctx, params)
	report.Positions = positionsResult

	// 4. Resumo geral
	report.Summary = s.generateSummary(transactionsResult, positionsResult, time.Since(startTime))

	helpers.LogInfo("Teste de conexão B3 concluído", map[string]interface{}{
		"cpf":                maskCPF(params.CPF),
		"totalTransactions":  report.Summary.TotalTransactions,
		"totalPositions":     report.Summary.TotalPositions,
		"uniqueTickers":      len(report.Summary.UniqueTickers),
		"duration":           report.Summary.TestDuration,
	})

	return report, nil
}

func (s *Service) testConnection(ctx context.Context) ConnectionTestResult {
	startTime := time.Now()

	// Tenta fazer uma chamada simples para testar a conectividade
	// Usamos uma consulta mínima de posições como teste de conexão
	testCPF := "00000000000" // CPF de teste para validar apenas conectividade
	testDate := "2024-01-01"
	
	_, err := s.b3Client.GetPositionsV3(ctx, testCPF, testDate, testDate, 1)
	duration := time.Since(startTime)

	if err != nil {
		return ConnectionTestResult{
			AuthenticationOK: false,
			APIReachable:     false,
			ResponseTime:     duration.String(),
			Error:            err.Error(),
		}
	}

	return ConnectionTestResult{
		AuthenticationOK: true,
		APIReachable:     true,
		ResponseTime:     duration.String(),
	}
}

func (s *Service) testTransactions(ctx context.Context, params TestParams) TransactionsSummary {
	// Busca transações do período especificado usando GetTransactionsV2
	startDate := params.StartDate.Format("2006-01-02")
	endDate := params.EndDate.Format("2006-01-02")
	
	resp, err := s.b3Client.GetTransactionsV2(ctx, params.CPF, startDate, endDate, 1)
	if err != nil {
		return TransactionsSummary{
			Error: err.Error(),
			DateRange: DateRangeInfo{
				Start: startDate,
				End:   endDate,
			},
		}
	}
	defer resp.Body.Close()

	// Parse da resposta JSON
	var transactionsData struct {
		Data []map[string]interface{} `json:"data"`
		Meta map[string]interface{}   `json:"meta"`
	}

	if err := parseJSONResponse(resp, &transactionsData); err != nil {
		return TransactionsSummary{
			Error: fmt.Sprintf("erro ao parsear resposta: %v", err),
			DateRange: DateRangeInfo{
				Start: startDate,
				End:   endDate,
			},
		}
	}

	// Analisa os dados
	assetTypes := make(map[string]int)
	operationTypes := make(map[string]int)
	tickersMap := make(map[string]bool)
	var sampleRecords []interface{}

	for i, tx := range transactionsData.Data {
		if i < 5 { // Mostra apenas 5 registros como amostra
			sampleRecords = append(sampleRecords, tx)
		}

		// Extrai informações para análise
		if assetType, exists := tx["assetType"]; exists {
			if assetTypeStr, ok := assetType.(string); ok {
				assetTypes[assetTypeStr]++
			}
		}
		if operation, exists := tx["operation"]; exists {
			if operationStr, ok := operation.(string); ok {
				operationTypes[operationStr]++
			}
		}
		if ticker, exists := tx["ticker"]; exists {
			if tickerStr, ok := ticker.(string); ok {
				tickersMap[tickerStr] = true
			}
		}
	}

	// Converte map de tickers para slice
	var tickers []string
	for ticker := range tickersMap {
		tickers = append(tickers, ticker)
	}

	totalRecords := len(transactionsData.Data)
	totalPages := 1
	if meta := transactionsData.Meta; meta != nil {
		if total, ok := meta["totalRecords"].(float64); ok {
			totalRecords = int(total)
		}
		if pages, ok := meta["totalPages"].(float64); ok {
			totalPages = int(pages)
		}
	}

	return TransactionsSummary{
		TotalRecords:   totalRecords,
		TotalPages:     totalPages,
		SampleRecords:  sampleRecords,
		AssetTypes:     assetTypes,
		OperationTypes: operationTypes,
		Tickers:        tickers,
		DateRange: DateRangeInfo{
			Start: startDate,
			End:   endDate,
		},
	}
}

func (s *Service) testPositions(ctx context.Context, params TestParams) PositionsSummary {
	// Busca posições do período especificado usando GetPositionsV3
	startDate := params.StartDate.Format("2006-01-02")
	endDate := params.EndDate.Format("2006-01-02")
	
	resp, err := s.b3Client.GetPositionsV3(ctx, params.CPF, startDate, endDate, 1)
	if err != nil {
		return PositionsSummary{
			Error: err.Error(),
			DateRange: DateRangeInfo{
				Start: startDate,
				End:   endDate,
			},
		}
	}
	defer resp.Body.Close()

	// Parse da resposta JSON
	var positionsData struct {
		Data []map[string]interface{} `json:"data"`
		Meta map[string]interface{}   `json:"meta"`
	}

	if err := parseJSONResponse(resp, &positionsData); err != nil {
		return PositionsSummary{
			Error: fmt.Sprintf("erro ao parsear resposta: %v", err),
			DateRange: DateRangeInfo{
				Start: startDate,
				End:   endDate,
			},
		}
	}

	// Analisa os dados
	assetTypes := make(map[string]int)
	tickersMap := make(map[string]bool)
	var sampleRecords []interface{}

	for i, pos := range positionsData.Data {
		if i < 5 { // Mostra apenas 5 registros como amostra
			sampleRecords = append(sampleRecords, pos)
		}

		// Extrai informações para análise
		if assetType, exists := pos["assetType"]; exists {
			if assetTypeStr, ok := assetType.(string); ok {
				assetTypes[assetTypeStr]++
			}
		}
		if ticker, exists := pos["ticker"]; exists {
			if tickerStr, ok := ticker.(string); ok {
				tickersMap[tickerStr] = true
			}
		}
	}

	// Converte map de tickers para slice
	var tickers []string
	for ticker := range tickersMap {
		tickers = append(tickers, ticker)
	}

	totalRecords := len(positionsData.Data)
	totalPages := 1
	if meta := positionsData.Meta; meta != nil {
		if total, ok := meta["totalRecords"].(float64); ok {
			totalRecords = int(total)
		}
		if pages, ok := meta["totalPages"].(float64); ok {
			totalPages = int(pages)
		}
	}

	return PositionsSummary{
		TotalRecords:  totalRecords,
		TotalPages:    totalPages,
		SampleRecords: sampleRecords,
		AssetTypes:    assetTypes,
		Tickers:       tickers,
		DateRange: DateRangeInfo{
			Start: startDate,
			End:   endDate,
		},
	}
}

func (s *Service) generateSummary(transactions TransactionsSummary, positions PositionsSummary, duration time.Duration) OverallSummary {
	// Combina tickers únicos de transações e posições
	tickersMap := make(map[string]bool)
	for _, ticker := range transactions.Tickers {
		tickersMap[ticker] = true
	}
	for _, ticker := range positions.Tickers {
		tickersMap[ticker] = true
	}

	var uniqueTickers []string
	for ticker := range tickersMap {
		uniqueTickers = append(uniqueTickers, ticker)
	}

	// Avalia qualidade dos dados
	dataQuality := "GOOD"
	if transactions.Error != "" || positions.Error != "" {
		dataQuality = "PARTIAL"
	}
	if transactions.TotalRecords == 0 && positions.TotalRecords == 0 {
		dataQuality = "NO_DATA"
	}

	return OverallSummary{
		TotalTransactions: transactions.TotalRecords,
		TotalPositions:    positions.TotalRecords,
		UniqueTickers:     uniqueTickers,
		TestDuration:      duration.String(),
		DataQuality:       dataQuality,
	}
}

// parseJSONResponse helper para parsear resposta HTTP em struct
func parseJSONResponse(resp *http.Response, target interface{}) error {
	if resp == nil {
		return fmt.Errorf("resposta HTTP é nil")
	}
	
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("erro ao decodificar JSON: %w", err)
	}
	
	return nil
}

// maskCPF mascara CPF mantendo apenas últimos 2 dígitos
func maskCPF(cpf string) string {
	if len(cpf) != 11 {
		return "invalid"
	}
	return "*********" + cpf[9:]
}
