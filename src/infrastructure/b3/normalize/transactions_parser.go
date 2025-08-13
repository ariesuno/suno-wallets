package normalize

import (
	"encoding/json"
	"strings"
	"time"

	hashx "suno-wallets/src/shared/hash"

	"github.com/google/uuid"
)

// Comentários em pt-BR: parser determinístico para transações v2 B3 format

// B3TransactionResponse representa a estrutura completa da resposta da B3
type B3TransactionResponse struct {
	Data B3TransactionData `json:"data"`
}

type B3TransactionData struct {
	Periods B3Periods `json:"periods"`
}

type B3Periods struct {
	PeriodLists    []B3PeriodList `json:"periodLists"`
	DocumentNumber string         `json:"documentNumber"`
}

type B3PeriodList struct {
	BuyTotal         float64          `json:"buyTotal"`
	SellTotal        float64          `json:"sellTotal"`
	ReferenceDate    string           `json:"referenceDate"`
	AssetTradingList []B3AssetTrading `json:"assetTradingList"`
}

type B3AssetTrading struct {
	Side                         string  `json:"side"`
	MarketName                   string  `json:"marketName"`
	PriceValue                   float64 `json:"priceValue"`
	GrossAmount                  float64 `json:"grossAmount"`
	TickerSymbol                 string  `json:"tickerSymbol"`
	TradeDateTime                string  `json:"tradeDateTime"`
	TradeQuantity                int     `json:"tradeQuantity"`
	ExpirationDate               string  `json:"expirationDate"`
	ParticipantName              string  `json:"participantName"`
	OptionExerciseValue          float64 `json:"optionExerciseValue"`
	OriginalTradePriceValue      float64 `json:"originalTradePriceValue"`
	ParticipantDocumentNumber    string  `json:"participantDocumentNumber"`
	OriginalTotalAdjustmentValue float64 `json:"originalTotalAdjustmentValue"`
	AssetTradingObjectCode       string  `json:"assetTradingObjectCode,omitempty"`
}

func NormalizeTransactions(tenantID string, cpf, assetType string, rawID uuid.UUID, payload []byte) ([]NormalizedTransaction, error) {
	// Parse da estrutura B3 específica
	var b3Response B3TransactionResponse
	if err := json.Unmarshal(payload, &b3Response); err != nil {
		// Se não conseguir parsear, pode ser um payload vazio ou formato diferente
		return []NormalizedTransaction{}, nil
	}

	var out []NormalizedTransaction
	sequenceIndex := 0

	// Iterar sobre todos os períodos e transações
	for _, periodList := range b3Response.Data.Periods.PeriodLists {
		for _, trade := range periodList.AssetTradingList {
			// Normalizar campos para formato padrão
			side := normalizeSide(trade.Side)
			ticker := strings.ToUpper(strings.TrimSpace(trade.TickerSymbol))
			tradeDate := parseB3DateTime(trade.TradeDateTime)

			// Extrair código do broker do campo participantDocumentNumber
			brokerCode := extractBrokerCode(trade.ParticipantDocumentNumber)

			// Criar hash único para esta transação
			nHash := hashx.ComputeNormalizedHash(
				tenantID, cpf, assetType,
				tradeDate.Format("2006-01-02"), ticker, side,
				itoa(trade.TradeQuantity), ftoa(trade.PriceValue),
				rawID.String(), itoa(sequenceIndex),
			)

			normalizedTx := NormalizedTransaction{
				ID:             uuid.New(),
				TenantID:       tenantID,
				CPF:            cpf,
				AssetType:      assetType,
				SourceVersion:  "v2",
				RawID:          rawID,
				SequenceInRaw:  sequenceIndex,
				TradeID:        nil, // B3 não fornece ID único de trade
				BrokerCode:     &brokerCode,
				TradeDate:      tradeDate,
				SettlementDate: parseB3ExpirationDate(trade.ExpirationDate),
				Ticker:         ticker,
				ISIN:           nil, // B3 não fornece ISIN neste endpoint
				Side:           side,
				Quantity:       itoa(trade.TradeQuantity),
				Price:          ftoa(trade.PriceValue),
				GrossValue:     strptr(ftoa(trade.GrossAmount)),
				Currency:       strptr("BRL"), // B3 sempre em BRL

				// New structured fields (extracted from previous extra_json)
				MarketName:              strptr(trade.MarketName),
				ParticipantName:         strptr(trade.ParticipantName),
				ParticipantDocument:     strptr(trade.ParticipantDocumentNumber),
				AssetTradingCode:        strptr(trade.AssetTradingObjectCode),
				ExpirationDate:          parseB3ExpirationDate(trade.ExpirationDate),
				OptionExerciseValue:     strptr(ftoa(trade.OptionExerciseValue)),
				OriginalTradePrice:      strptr(ftoa(trade.OriginalTradePriceValue)),
				OriginalAdjustmentValue: strptr(ftoa(trade.OriginalTotalAdjustmentValue)),
				TradeDateTime:           timeptr(parseB3DateTime(trade.TradeDateTime)),

				NormalizedHash: nHash,
				NormalizedAt:   time.Now(),
			}

			out = append(out, normalizedTx)
			sequenceIndex++
		}
	}

	return out, nil
}

// normalizeSide converte o lado da transação B3 para formato padrão
func normalizeSide(b3Side string) string {
	switch strings.ToUpper(strings.TrimSpace(b3Side)) {
	case "COMPRA":
		return "BUY"
	case "VENDA":
		return "SELL"
	default:
		return "OTHER"
	}
}

// parseB3DateTime converte data/hora B3 para time.Time
func parseB3DateTime(dateTimeStr string) time.Time {
	// Formato B3: "2021-03-16T13:19:21"
	if parsed, err := time.Parse("2006-01-02T15:04:05", dateTimeStr); err == nil {
		return parsed
	}
	// Fallback para apenas data
	if parsed, err := time.Parse("2006-01-02", dateTimeStr[:10]); err == nil {
		return parsed
	}
	return time.Time{}
}

// parseB3ExpirationDate converte data de vencimento B3 para ponteiro de time.Time
func parseB3ExpirationDate(dateStr string) *time.Time {
	if dateStr == "9999-12-31" || dateStr == "" {
		return nil // Sem vencimento ou vencimento indefinido
	}
	if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
		return &parsed
	}
	return nil
}

// extractBrokerCode extrai código do broker do CNPJ
func extractBrokerCode(cnpj string) string {
	// Remove formatação e mantém apenas os primeiros 8 dígitos (código da empresa)
	cleaned := strings.ReplaceAll(cnpj, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "/", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	if len(cleaned) >= 8 {
		return cleaned[:8]
	}
	return cleaned
}

// Funções auxiliares

// timeptr retorna um ponteiro para time.Time
func timeptr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
