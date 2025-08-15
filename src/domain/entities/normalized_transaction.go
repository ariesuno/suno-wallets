package entities

import (
	"time"

	"suno-wallets/src/domain/valueobjects"

	"github.com/google/uuid"
)

// NormalizedTransaction representa uma transação B3 normalizada no domínio
type NormalizedTransaction struct {
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
	Side          string    `json:"side"` // BUY, SELL, OTHER
	Quantity      string    `json:"quantity"`
	Price         *string   `json:"price,omitempty"`
	Amount        *string   `json:"amount,omitempty"`
	Fee           *string   `json:"fee,omitempty"`
	Currency      *string   `json:"currency,omitempty"`

	NormalizedHash string    `json:"normalized_hash"`
	NormalizedAt   time.Time `json:"normalized_at"`
}

// NewNormalizedTransaction cria nova transação normalizada
func NewNormalizedTransaction(tenantID string, cpf *valueobjects.CPF, assetType string, rawID uuid.UUID) (*NormalizedTransaction, error) {
	if tenantID == "" {
		return nil, NewValidationError("tenant ID é obrigatório")
	}
	if cpf == nil {
		return nil, NewValidationError("CPF é obrigatório")
	}
	if assetType == "" {
		return nil, NewValidationError("asset type é obrigatório")
	}

	return &NormalizedTransaction{
		ID:           uuid.New(),
		TenantID:     tenantID,
		CPF:          cpf,
		AssetType:    assetType,
		RawID:        rawID,
		NormalizedAt: time.Now(),
	}, nil
}

// Validate valida a entidade de acordo com regras de negócio
func (nt *NormalizedTransaction) Validate() error {
	if nt.TenantID == "" {
		return NewFieldValidationError("tenant_id", "é obrigatório")
	}

	if nt.CPF == nil {
		return NewFieldValidationError("cpf", "é obrigatório")
	}

	if err := nt.CPF.Validate(); err != nil {
		return NewFieldValidationError("cpf", err.Error())
	}

	if nt.Ticker == "" {
		return NewFieldValidationError("ticker", "é obrigatório")
	}

	if nt.Side != "BUY" && nt.Side != "SELL" && nt.Side != "OTHER" {
		return NewFieldValidationError("side", "deve ser BUY, SELL ou OTHER")
	}

	if nt.Quantity == "" {
		return NewFieldValidationError("quantity", "é obrigatório")
	}

	return nil
}

// GetDomainEvents retorna eventos de domínio (para Event Sourcing futuro)
func (nt *NormalizedTransaction) GetDomainEvents() []interface{} {
	return []interface{}{
		// Poderia retornar eventos como TransactionNormalized, etc.
	}
}
