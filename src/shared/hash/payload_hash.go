package hashx

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Comentários em pt-BR: hash canônico para RAW (payload + chaves de contexto)

func ComputeRawHash(payload interface{}, endpoint, tenantID, cpf, periodStart, periodEnd string, page int) (string, error) {
	normalized, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	base := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%s", endpoint, tenantID, cpf, periodStart, periodEnd, "v1", page, string(normalized))
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:]), nil
}
