package entities

import (
	"time"

	"suno-wallets/src/domain/valueobjects"

	"github.com/google/uuid"
)

// NormalizedPosition representa uma posição B3 normalizada no domínio
type NormalizedPosition struct {
	ID            uuid.UUID         `json:"id"`
	TenantID      string            `json:"tenant_id"`
	CPF           *valueobjects.CPF `json:"cpf"`
	AssetType     string            `json:"asset_type"`
	SourceVersion string            `json:"source_version"`
	RawID         uuid.UUID         `json:"raw_id"`
	SequenceInRaw int               `json:"sequence_in_raw"`

	ReferenceDate time.Time `json:"reference_date"`
	Ticker        string    `json:"ticker"`
	ISIN          *string   `json:"isin,omitempty"`
	Quantity      string    `json:"quantity"`
	AvgPrice      *string   `json:"avg_price,omitempty"`
	PositionValue *string   `json:"position_value,omitempty"`
	Currency      *string   `json:"currency,omitempty"`

	NormalizedHash string    `json:"normalized_hash"`
	NormalizedAt   time.Time `json:"normalized_at"`
}

// NewNormalizedPosition cria nova posição normalizada
func NewNormalizedPosition(tenantID string, cpf *valueobjects.CPF, assetType string, rawID uuid.UUID) (*NormalizedPosition, error) {
	if tenantID == "" {
		return nil, NewValidationError("tenant ID é obrigatório")
	}
	if cpf == nil {
		return nil, NewValidationError("CPF é obrigatório")
	}
	if assetType == "" {
		return nil, NewValidationError("asset type é obrigatório")
	}

	return &NormalizedPosition{
		ID:           uuid.New(),
		TenantID:     tenantID,
		CPF:          cpf,
		AssetType:    assetType,
		RawID:        rawID,
		NormalizedAt: time.Now(),
	}, nil
}

// Validate valida a entidade de acordo com regras de negócio
func (np *NormalizedPosition) Validate() error {
	if np.TenantID == "" {
		return NewFieldValidationError("tenant_id", "é obrigatório")
	}

	if np.CPF == nil {
		return NewFieldValidationError("cpf", "é obrigatório")
	}

	if err := np.CPF.Validate(); err != nil {
		return NewFieldValidationError("cpf", err.Error())
	}

	if np.Ticker == "" {
		return NewFieldValidationError("ticker", "é obrigatório")
	}

	if np.Quantity == "" {
		return NewFieldValidationError("quantity", "é obrigatório")
	}

	if np.ReferenceDate.IsZero() {
		return NewFieldValidationError("reference_date", "é obrigatório")
	}

	return nil
}

// IsPositive verifica se a posição tem quantidade positiva
func (np *NormalizedPosition) IsPositive() bool {
	return np.Quantity != "0" && np.Quantity != ""
}

// GetDomainEvents retorna eventos de domínio
func (np *NormalizedPosition) GetDomainEvents() []interface{} {
	return []interface{}{
		// Poderia retornar eventos como PositionNormalized, etc.
	}
}
