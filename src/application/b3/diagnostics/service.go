package diagnostics

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

// Comentários em pt-BR: service de diagnóstico para testar conectividade e dados da B3

type Service struct {
	b3Client *b3Client.B3OfficialClient
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
	TotalRecords   int            `json:"totalRecords"`
	TotalPages     int            `json:"totalPages"`
	SampleRecords  []interface{}  `json:"sampleRecords"`
	AssetTypes     map[string]int `json:"assetTypes"`
	OperationTypes map[string]int `json:"operationTypes"`
	DateRange      DateRangeInfo  `json:"dateRange"`
	Tickers        []string       `json:"tickers"`
	Error          string         `json:"error,omitempty"`
}

type PositionsSummary struct {
	TotalRecords  int            `json:"totalRecords"`
	TotalPages    int            `json:"totalPages"`
	SampleRecords []interface{}  `json:"sampleRecords"`
	AssetTypes    map[string]int `json:"assetTypes"`
	DateRange     DateRangeInfo  `json:"dateRange"`
	Tickers       []string       `json:"tickers"`
	Error         string         `json:"error,omitempty"`
}

type DateRangeInfo struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Earliest string `json:"earliest,omitempty"`
	Latest   string `json:"latest,omitempty"`
}

type OverallSummary struct {
	TotalTransactions int      `json:"totalTransactions"`
	TotalPositions    int      `json:"totalPositions"`
	UniqueTickers     []string `json:"uniqueTickers"`
	TestDuration      string   `json:"testDuration"`
	DataQuality       string   `json:"dataQuality"`
}

func NewService(b3Client *b3Client.B3OfficialClient) *Service {
	return &Service{b3Client: b3Client}
}

func (s *Service) TestConnectionAndReport(ctx context.Context, params TestParams) (*TestReport, error) {
	if s.b3Client == nil {
		return nil, fmt.Errorf("B3 client não configurado: defina B3_CERT_P12_PATH, B3_CERT_PASSPHRASE, B3_URL_DATA, B3_OAUTH_TOKEN_URL, B3_CLIENT_ID e B3_CLIENT_SECRET")
	}
	startTime := time.Now()

	helpers.LogInfo("Iniciando teste de conexão B3", map[string]interface{}{
		"cpf":       maskCPF(params.CPF),
		"tenantId":  params.TenantID,
		"dateRange": fmt.Sprintf("%s a %s", params.StartDate.Format("2006-01-02"), params.EndDate.Format("2006-01-02")),
	})

	report := &TestReport{}

	connectionResult := s.testConnection(ctx, params.CPF)
	report.ConnectionTest = connectionResult
	if !connectionResult.AuthenticationOK {
		return report, fmt.Errorf("falha na autenticação B3: %s", connectionResult.Error)
	}

	transactionsResult := s.testTransactions(ctx, params)
	report.Transactions = transactionsResult

	positionsResult := s.testPositions(ctx, params)
	report.Positions = positionsResult

	report.Summary = s.generateSummary(transactionsResult, positionsResult, time.Since(startTime))

	helpers.LogInfo("Teste de conexão B3 concluído", map[string]interface{}{
		"cpf":               maskCPF(params.CPF),
		"totalTransactions": report.Summary.TotalTransactions,
		"totalPositions":    report.Summary.TotalPositions,
		"uniqueTickers":     len(report.Summary.UniqueTickers),
		"duration":          report.Summary.TestDuration,
	})

	return report, nil
}

func (s *Service) testConnection(ctx context.Context, cpf string) ConnectionTestResult {
	if s.b3Client == nil {
		return ConnectionTestResult{AuthenticationOK: false, APIReachable: false, ResponseTime: "0s", Error: "B3 client não configurado"}
	}
	startTime := time.Now()
	testDate := "2024-01-01"

	_, err := s.b3Client.GetPositionsV3(ctx, cpf, testDate, testDate, 1)
	duration := time.Since(startTime)
	if err != nil {
		return ConnectionTestResult{AuthenticationOK: false, APIReachable: false, ResponseTime: duration.String(), Error: err.Error()}
	}
	return ConnectionTestResult{AuthenticationOK: true, APIReachable: true, ResponseTime: duration.String()}
}

func (s *Service) testTransactions(ctx context.Context, params TestParams) TransactionsSummary {
	startDate := params.StartDate.Format("2006-01-02")
	endDate := params.EndDate.Format("2006-01-02")
	resp, err := s.b3Client.GetTransactionsV2(ctx, params.CPF, startDate, endDate, 1)
	if err != nil {
		return TransactionsSummary{Error: err.Error(), DateRange: DateRangeInfo{Start: startDate, End: endDate}}
	}
	defer resp.Body.Close()
	// Usar estrutura B3 correta
	var transactionsData struct {
		Data struct {
			Periods struct {
				PeriodLists []struct {
					BuyTotal         float64 `json:"buyTotal"`
					SellTotal        float64 `json:"sellTotal"`
					ReferenceDate    string  `json:"referenceDate"`
					AssetTradingList []struct {
						Side                      string  `json:"side"`
						MarketName                string  `json:"marketName"`
						PriceValue                float64 `json:"priceValue"`
						GrossAmount               float64 `json:"grossAmount"`
						TickerSymbol              string  `json:"tickerSymbol"`
						TradeDateTime             string  `json:"tradeDateTime"`
						TradeQuantity             int     `json:"tradeQuantity"`
						ParticipantName           string  `json:"participantName"`
						ParticipantDocumentNumber string  `json:"participantDocumentNumber"`
					} `json:"assetTradingList"`
				} `json:"periodLists"`
			} `json:"periods"`
		} `json:"data"`
		Meta map[string]interface{} `json:"meta"`
	}
	if err := parseJSONResponse(resp, &transactionsData); err != nil {
		return TransactionsSummary{Error: fmt.Sprintf("erro ao parsear resposta: %v", err), DateRange: DateRangeInfo{Start: startDate, End: endDate}}
	}
	assetTypes := make(map[string]int)
	operationTypes := make(map[string]int)
	tickersMap := make(map[string]bool)
	var sampleRecords []interface{}
	totalRecords := 0

	// Processar estrutura B3 aninhada
	for _, periodList := range transactionsData.Data.Periods.PeriodLists {
		for _, tx := range periodList.AssetTradingList {
			totalRecords++
			if len(sampleRecords) < 5 {
				sampleRecords = append(sampleRecords, map[string]interface{}{
					"ticker":      tx.TickerSymbol,
					"side":        tx.Side,
					"quantity":    tx.TradeQuantity,
					"price":       tx.PriceValue,
					"grossAmount": tx.GrossAmount,
					"marketName":  tx.MarketName,
					"tradeDate":   tx.TradeDateTime,
					"participant": tx.ParticipantName,
				})
			}

			// Classificar por mercado como "asset type"
			assetTypes[tx.MarketName]++
			operationTypes[tx.Side]++
			tickersMap[tx.TickerSymbol] = true
		}
	}

	var tickers []string
	for t := range tickersMap {
		tickers = append(tickers, t)
	}
	totalPages := 1
	if meta := transactionsData.Meta; meta != nil {
		if total, ok := meta["totalRecords"].(float64); ok {
			totalRecords = int(total)
		}
		if pages, ok := meta["totalPages"].(float64); ok {
			totalPages = int(pages)
		}
	}
	return TransactionsSummary{TotalRecords: totalRecords, TotalPages: totalPages, SampleRecords: sampleRecords, AssetTypes: assetTypes, OperationTypes: operationTypes, Tickers: tickers, DateRange: DateRangeInfo{Start: startDate, End: endDate}}
}

func (s *Service) testPositions(ctx context.Context, params TestParams) PositionsSummary {
	startDate := params.StartDate.Format("2006-01-02")
	endDate := params.EndDate.Format("2006-01-02")
	resp, err := s.b3Client.GetPositionsV3(ctx, params.CPF, startDate, endDate, 1)
	if err != nil {
		return PositionsSummary{Error: err.Error(), DateRange: DateRangeInfo{Start: startDate, End: endDate}}
	}
	defer resp.Body.Close()
	var positionsData struct {
		Data []map[string]interface{} `json:"data"`
		Meta map[string]interface{}   `json:"meta"`
	}
	if err := parseJSONResponse(resp, &positionsData); err != nil {
		return PositionsSummary{Error: fmt.Sprintf("erro ao parsear resposta: %v", err), DateRange: DateRangeInfo{Start: startDate, End: endDate}}
	}
	assetTypes := make(map[string]int)
	tickersMap := make(map[string]bool)
	var sampleRecords []interface{}
	for i, pos := range positionsData.Data {
		if i < 5 {
			sampleRecords = append(sampleRecords, pos)
		}
		if v, ok := pos["assetType"].(string); ok {
			assetTypes[v]++
		}
		if v, ok := pos["ticker"].(string); ok {
			tickersMap[v] = true
		}
	}
	var tickers []string
	for t := range tickersMap {
		tickers = append(tickers, t)
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
	return PositionsSummary{TotalRecords: totalRecords, TotalPages: totalPages, SampleRecords: sampleRecords, AssetTypes: assetTypes, Tickers: tickers, DateRange: DateRangeInfo{Start: startDate, End: endDate}}
}

func (s *Service) generateSummary(transactions TransactionsSummary, positions PositionsSummary, duration time.Duration) OverallSummary {
	tickersMap := make(map[string]bool)
	for _, t := range transactions.Tickers {
		tickersMap[t] = true
	}
	for _, t := range positions.Tickers {
		tickersMap[t] = true
	}
	var uniqueTickers []string
	for t := range tickersMap {
		uniqueTickers = append(uniqueTickers, t)
	}
	dataQuality := "GOOD"
	if transactions.Error != "" || positions.Error != "" {
		dataQuality = "PARTIAL"
	}
	if transactions.TotalRecords == 0 && positions.TotalRecords == 0 {
		dataQuality = "NO_DATA"
	}
	return OverallSummary{TotalTransactions: transactions.TotalRecords, TotalPositions: positions.TotalRecords, UniqueTickers: uniqueTickers, TestDuration: duration.String(), DataQuality: dataQuality}
}

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

func maskCPF(cpf string) string {
	if len(cpf) != 11 {
		return "invalid"
	}
	return "*********" + cpf[9:]
}
