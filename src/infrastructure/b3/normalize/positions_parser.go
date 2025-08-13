package normalize

import (
	"encoding/json"
	"strings"
	"time"

	hashx "suno-wallets/src/shared/hash"

	"github.com/google/uuid"
)

// Comentários em pt-BR: parser determinístico para posições v3 (flatten básico)

func NormalizePositions(tenantID string, cpf, assetType string, rawID uuid.UUID, payload []byte) ([]NormalizedPosition, error) {
	var list []map[string]any
	if err := json.Unmarshal(payload, &list); err != nil {
		// tentar dentro de "data"
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(payload, &env)
		_ = json.Unmarshal(env.Data, &list)
	}
	var out []NormalizedPosition
	for i, m := range list {
		ticker := strings.ToUpper(stringify(m["ticker"]))
		ref := parseYMD(stringify(m["referenceDate"]))
		qty := stringify(m["quantity"])
		avg := stringify(m["avgPrice"])
		val := stringify(m["positionValue"])
		isin := strptr(stringify(m["isin"]))
		cur := strptr(stringify(m["currency"]))
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
