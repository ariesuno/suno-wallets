package normalize

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	hashx "suno-wallets/src/shared/hash"

	"github.com/google/uuid"
)

// Comentários em pt-BR: parser determinístico para posições v3 (flatten básico)

func NormalizePositions(tenantID string, cpf, assetType string, rawID uuid.UUID, payload []byte) ([]NormalizedPosition, error) {
	var list []map[string]any

	// Primeiro tentar estrutura simples (array direto)
	if err := json.Unmarshal(payload, &list); err != nil {
		// Tentar estrutura v3 da B3: {"data": {"equitiesPositions": [...]}}
		var env struct {
			Data struct {
				EquitiesPositions []map[string]any `json:"equitiesPositions"`
			} `json:"data"`
		}
		if err := json.Unmarshal(payload, &env); err == nil {
			list = env.Data.EquitiesPositions
		} else {
			// Fallback para estrutura genérica com "data"
			var fallback struct {
				Data json.RawMessage `json:"data"`
			}
			_ = json.Unmarshal(payload, &fallback)
			_ = json.Unmarshal(fallback.Data, &list)
		}
	}

	var out []NormalizedPosition
	if len(list) == 0 {
		return out, nil // Return early se não há dados para processar
	}

	for i, m := range list {
		// Mapear campos da estrutura v3 da B3
		ticker := strings.ToUpper(stringify(m["tickerSymbol"]))
		ref := parseYMD(stringify(m["referenceDate"]))
		qty := stringify(m["equitiesQuantity"])
		avg := stringify(m["closingPrice"]) // Usar closingPrice como proxy para avgPrice
		val := stringify(m["updateValue"])  // updateValue é o valor da posição
		isin := strptr(stringify(m["isin"]))
		cur := strptr("BRL") // Currency não vem no payload, assumir BRL

		// VALIDAÇÕES DE CAMPOS OBRIGATÓRIOS
		// Validar se ticker não está vazio
		if ticker == "" {
			continue // Skip posições sem ticker
		}

		// Validar se quantity não está vazio e é um número válido
		if qty == "" {
			continue // Skip posições sem quantity
		}

		// Validar se quantity é um número válido (aceita int ou float)
		if _, err := strconv.ParseFloat(qty, 64); err != nil {
			continue // Skip posições com quantity inválida
		}

		// Validar se referenceDate não está vazio
		if stringify(m["referenceDate"]) == "" {
			continue // Skip posições sem data de referência
		}

		// Validar se a data de referência foi parseada corretamente (não zero)
		if ref.IsZero() {
			continue // Skip posições com data inválida
		}

		nHash := hashx.ComputeNormalizedHash(tenantID, cpf, assetType, ref.Format("2006-01-02"), ticker, qty, rawID.String(), itoa(i))

		out = append(out, NormalizedPosition{
			ID: uuid.New(), TenantID: tenantID, CPF: cpf, AssetType: assetType, SourceVersion: "v3", RawID: rawID,
			SequenceInRaw: i, ReferenceDate: ref, Ticker: ticker, ISIN: isin, Quantity: qty, AvgPrice: optstr(avg), PositionValue: optstr(val),
			Currency: cur, NormalizedHash: nHash, NormalizedAt: time.Now(),
		})
	}

	return out, nil
}

// helpers extraídos para helpers.go
