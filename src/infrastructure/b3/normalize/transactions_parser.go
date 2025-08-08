package normalize

import (
    "encoding/json"
    "strings"
    "time"

    "github.com/google/uuid"
    hashx "suno-wallets/src/shared/hash"
)

// Comentários em pt-BR: parser determinístico para transações v2 (flatten básico)

type RawEnvelope struct {
    Data json.RawMessage `json:"data"`
}

func NormalizeTransactions(tenantID uuid.UUID, cpf, assetType string, rawID uuid.UUID, payload []byte) ([]NormalizedTransaction, error) {
    // Exemplo genérico: assume lista em data
    var env RawEnvelope
    _ = json.Unmarshal(payload, &env)
    var list []map[string]any
    if err := json.Unmarshal(env.Data, &list); err != nil { _ = json.Unmarshal(payload, &list) }
    var out []NormalizedTransaction
    for i, m := range list {
        ticker := strings.ToUpper(stringify(m["ticker"]))
        tradeDate := parseYMD(stringify(m["tradeDate"]))
        side := strings.ToUpper(stringify(m["side"]))
        if side != "BUY" && side != "SELL" { side = "OTHER" }
        qty := stringify(m["quantity"])
        price := stringify(m["price"])
        gross := stringify(m["grossValue"])
        brk := strptr(stringify(m["brokerCode"]))
        tid := strptr(stringify(m["tradeId"]))
        isin := strptr(stringify(m["isin"]))
        cur := strptr(stringify(m["currency"]))
        nHash := hashx.ComputeNormalizedHash(tenantID.String(), cpf, assetType, tradeDate.Format("2006-01-02"), ticker, side, qty, price, rawID.String(), itoa(i))
        extra, _ := json.Marshal(m)
        out = append(out, NormalizedTransaction{
            ID: uuid.New(), TenantID: tenantID, CPF: cpf, AssetType: assetType, SourceVersion: "v2", RawID: rawID,
            SequenceInRaw: i, TradeID: tid, BrokerCode: brk, TradeDate: tradeDate, SettlementDate: nil, Ticker: ticker,
            ISIN: isin, Side: side, Quantity: qty, Price: price, GrossValue: optstr(gross), Currency: cur, ExtraJSON: extra,
            NormalizedHash: nHash, NormalizedAt: time.Now(),
        })
    }
    return out, nil
}

// helpers extraídos para helpers.go


