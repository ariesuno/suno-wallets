package hashx

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Comentários em pt-BR: hash determinístico para registros normalizados

func ComputeNormalizedHash(fields ...string) string {
	b := strings.Builder{}
	for i, f := range fields {
		if i > 0 {
			b.WriteString("|")
		}
		b.WriteString(f)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
