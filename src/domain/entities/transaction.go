package entities

import (
	"time"

	"suno-wallets/src/domain/enums"
	"suno-wallets/src/domain/valueobjects"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Transaction representa uma transação financeira no sistema
type Transaction struct {
	// Identificação primária
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`

	// Informações básicas da transação
	Type        enums.TransactionType   `json:"type" gorm:"type:varchar(50);not null"`
	Status      enums.TransactionStatus `json:"status" gorm:"type:varchar(50);not null;default:'pending'"`
	Description string                  `json:"description" gorm:"type:text"`
	Reference   string                  `json:"reference,omitempty" gorm:"type:varchar(255);index"`

	// Valores e moeda
	Amount       *valueobjects.Money `json:"amount" gorm:"embedded;embeddedPrefix:amount_"`
	Fee          *valueobjects.Money `json:"fee,omitempty" gorm:"embedded;embeddedPrefix:fee_"`
	NetAmount    *valueobjects.Money `json:"net_amount" gorm:"embedded;embeddedPrefix:net_amount_"`
	ExchangeRate float64             `json:"exchange_rate,omitempty" gorm:"type:decimal(20,8);default:1"`

	// Relacionamentos principais
	WalletID            uuid.UUID  `json:"wallet_id" gorm:"type:uuid;not null;index"`
	DestinationWalletID *uuid.UUID `json:"destination_wallet_id,omitempty" gorm:"type:uuid;index"`
	AssetID             uuid.UUID  `json:"asset_id" gorm:"type:uuid;not null;index"`
	UserID              uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`

	// Informações de processamento
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty"`
	FailedAt        *time.Time `json:"failed_at,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	ProcessingNotes string     `json:"processing_notes,omitempty" gorm:"type:text"`
	FailureReason   string     `json:"failure_reason,omitempty" gorm:"type:text"`

	// Informações de blockchain (para criptomoedas)
	BlockchainTxHash string `json:"blockchain_tx_hash,omitempty" gorm:"type:varchar(255);index"`
	BlockNumber      *int64 `json:"block_number,omitempty" gorm:"type:bigint"`
	Confirmations    int    `json:"confirmations" gorm:"type:int;default:0"`
	GasUsed          *int64 `json:"gas_used,omitempty" gorm:"type:bigint"`
	GasFee           *int64 `json:"gas_fee,omitempty" gorm:"type:bigint"`

	// Informações bancárias (para transferências fiat)
	BankAccount   string `json:"bank_account,omitempty" gorm:"type:varchar(100)"`
	BankCode      string `json:"bank_code,omitempty" gorm:"type:varchar(10)"`
	BankReference string `json:"bank_reference,omitempty" gorm:"type:varchar(255)"`

	// Informações de compliance e auditoria
	RiskScore        float64    `json:"risk_score" gorm:"type:decimal(5,2);default:0"`
	IsHighRisk       bool       `json:"is_high_risk" gorm:"type:boolean;default:false"`
	ComplianceStatus string     `json:"compliance_status" gorm:"type:varchar(50);default:'approved'"` // approved, under_review, rejected
	ComplianceNotes  string     `json:"compliance_notes,omitempty" gorm:"type:text"`
	ReviewedBy       *uuid.UUID `json:"reviewed_by,omitempty" gorm:"type:uuid"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`

	// Informações da sessão/origem
	IPAddress string `json:"ip_address,omitempty" gorm:"type:inet"`
	UserAgent string `json:"user_agent,omitempty" gorm:"type:text"`
	DeviceID  string `json:"device_id,omitempty" gorm:"type:varchar(255)"`
	SessionID string `json:"session_id,omitempty" gorm:"type:varchar(255)"`
	Channel   string `json:"channel" gorm:"type:varchar(50);default:'web'"` // web, mobile, api

	// Transação relacionada (para estornos/correções)
	ParentTransactionID *uuid.UUID    `json:"parent_transaction_id,omitempty" gorm:"type:uuid;index"`
	ChildTransactions   []Transaction `json:"child_transactions,omitempty" gorm:"foreignKey:ParentTransactionID"`

	// Metadados flexíveis
	Tags     []string               `json:"tags,omitempty" gorm:"type:text[]"`
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`

	// Campos de auditoria obrigatórios
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	CreatedBy uuid.UUID      `json:"created_by" gorm:"type:uuid"`
	UpdatedBy *uuid.UUID     `json:"updated_by,omitempty" gorm:"type:uuid"`
}

// TableName define o nome da tabela no banco de dados
func (Transaction) TableName() string {
	return "transactions"
}

// BeforeCreate hook executado antes da criação
func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}

	// Calcular valor líquido se não definido
	if t.NetAmount == nil && t.Amount != nil {
		netAmount := t.Amount
		if t.Fee != nil {
			var err error
			netAmount, err = t.Amount.Subtract(t.Fee)
			if err != nil {
				return err
			}
		}
		t.NetAmount = netAmount
	}

	// Definir expiração padrão se não definida
	if t.ExpiresAt == nil {
		expires := time.Now().Add(24 * time.Hour) // 24 horas por padrão
		t.ExpiresAt = &expires
	}

	return nil
}

// BeforeUpdate hook executado antes da atualização
func (t *Transaction) BeforeUpdate(tx *gorm.DB) error {
	// Atualizar timestamps de processamento baseado no status
	now := time.Now()

	switch t.Status {
	case enums.TransactionStatusProcessing:
		if t.ProcessedAt == nil {
			t.ProcessedAt = &now
		}
	case enums.TransactionStatusCompleted:
		if t.ConfirmedAt == nil {
			t.ConfirmedAt = &now
		}
	case enums.TransactionStatusFailed:
		if t.FailedAt == nil {
			t.FailedAt = &now
		}
	}

	return nil
}

// IsPending verifica se a transação está pendente
func (t *Transaction) IsPending() bool {
	return t.Status == enums.TransactionStatusPending
}

// IsProcessing verifica se a transação está sendo processada
func (t *Transaction) IsProcessing() bool {
	return t.Status == enums.TransactionStatusProcessing
}

// IsCompleted verifica se a transação foi concluída
func (t *Transaction) IsCompleted() bool {
	return t.Status.IsSuccessful()
}

// IsFailed verifica se a transação falhou
func (t *Transaction) IsFailed() bool {
	return t.Status == enums.TransactionStatusFailed
}

// IsFinal verifica se a transação está em estado final
func (t *Transaction) IsFinal() bool {
	return t.Status.IsFinal()
}

// CanBeCancelled verifica se a transação pode ser cancelada
func (t *Transaction) CanBeCancelled() bool {
	return t.Status.CanBeCancelled()
}

// CanBeReversed verifica se a transação pode ser revertida
func (t *Transaction) CanBeReversed() bool {
	return t.Status.CanBeReversed()
}

// IsExpired verifica se a transação expirou
func (t *Transaction) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.ExpiresAt)
}

// IsInbound verifica se é uma transação de entrada
func (t *Transaction) IsInbound() bool {
	return t.Type.IsInbound()
}

// IsOutbound verifica se é uma transação de saída
func (t *Transaction) IsOutbound() bool {
	return t.Type.IsOutbound()
}

// IsTransfer verifica se é uma transferência
func (t *Transaction) IsTransfer() bool {
	return t.Type == enums.TransactionTypeTransfer
}

// RequiresDestination verifica se requer carteira de destino
func (t *Transaction) RequiresDestination() bool {
	return t.Type.RequiresDestination()
}

// GetAmountInReais retorna o valor em reais
func (t *Transaction) GetAmountInReais() float64 {
	if t.Amount == nil {
		return 0
	}
	return t.Amount.ToFloat()
}

// GetFeeInReais retorna a taxa em reais
func (t *Transaction) GetFeeInReais() float64 {
	if t.Fee == nil {
		return 0
	}
	return t.Fee.ToFloat()
}

// GetNetAmountInReais retorna o valor líquido em reais
func (t *Transaction) GetNetAmountInReais() float64 {
	if t.NetAmount == nil {
		return 0
	}
	return t.NetAmount.ToFloat()
}

// GetBlockchainExplorerURL retorna URL do explorador de blockchain
func (t *Transaction) GetBlockchainExplorerURL() string {
	if t.BlockchainTxHash == "" {
		return ""
	}
	// Implementar baseado na blockchain específica
	// Por exemplo, para Bitcoin: https://blockstream.info/tx/
	// Para Ethereum: https://etherscan.io/tx/
	return ""
}

// IsHighValue verifica se é uma transação de alto valor
func (t *Transaction) IsHighValue(threshold *valueobjects.Money) bool {
	if t.Amount == nil || threshold == nil {
		return false
	}

	greater, err := t.Amount.GreaterThan(threshold)
	if err != nil {
		return false
	}

	return greater
}

// CalculateEffectiveAmount calcula o valor efetivo baseado no tipo de transação
func (t *Transaction) CalculateEffectiveAmount() *valueobjects.Money {
	if t.Amount == nil {
		return nil
	}

	// Para transações de entrada, retorna valor positivo
	if t.IsInbound() {
		return t.Amount
	}

	// Para transações de saída, retorna valor negativo
	if t.IsOutbound() {
		return t.Amount.Negate()
	}

	// Para transferências, depende do contexto da carteira
	return t.Amount
}

// AddTag adiciona uma tag à transação
func (t *Transaction) AddTag(tag string) {
	for _, existingTag := range t.Tags {
		if existingTag == tag {
			return // Tag já existe
		}
	}
	t.Tags = append(t.Tags, tag)
}

// RemoveTag remove uma tag da transação
func (t *Transaction) RemoveTag(tag string) {
	for i, existingTag := range t.Tags {
		if existingTag == tag {
			t.Tags = append(t.Tags[:i], t.Tags[i+1:]...)
			break
		}
	}
}

// HasTag verifica se a transação tem uma tag específica
func (t *Transaction) HasTag(tag string) bool {
	for _, existingTag := range t.Tags {
		if existingTag == tag {
			return true
		}
	}
	return false
}

// MarkAsHighRisk marca a transação como alto risco
func (t *Transaction) MarkAsHighRisk(reason string) {
	t.IsHighRisk = true
	t.ComplianceStatus = "under_review"
	if t.ComplianceNotes == "" {
		t.ComplianceNotes = reason
	} else {
		t.ComplianceNotes += "; " + reason
	}
	t.AddTag("high_risk")
}

// SetFailure define a transação como falhada com motivo
func (t *Transaction) SetFailure(reason string) {
	t.Status = enums.TransactionStatusFailed
	t.FailureReason = reason
	now := time.Now()
	t.FailedAt = &now
}

// Validate valida os dados da transação
func (t *Transaction) Validate() error {
	if t.TenantID == uuid.Nil {
		return NewValidationError("tenant_id é obrigatório")
	}

	if !t.Type.IsValid() {
		return NewValidationError("tipo de transação inválido")
	}

	if !t.Status.IsValid() {
		return NewValidationError("status de transação inválido")
	}

	if t.Amount == nil {
		return NewValidationError("valor da transação é obrigatório")
	}

	if err := t.Amount.Validate(); err != nil {
		return err
	}

	if !t.Amount.IsPositive() {
		return NewValidationError("valor da transação deve ser positivo")
	}

	if t.WalletID == uuid.Nil {
		return NewValidationError("wallet_id é obrigatório")
	}

	if t.AssetID == uuid.Nil {
		return NewValidationError("asset_id é obrigatório")
	}

	if t.UserID == uuid.Nil {
		return NewValidationError("user_id é obrigatório")
	}

	// Validar se transação de transferência tem carteira de destino
	if t.RequiresDestination() && t.DestinationWalletID == nil {
		return NewValidationError("transferências requerem carteira de destino")
	}

	// Validar se carteira de origem e destino são diferentes
	if t.DestinationWalletID != nil && t.WalletID == *t.DestinationWalletID {
		return NewValidationError("carteira de origem e destino devem ser diferentes")
	}

	// Validar taxa se fornecida
	if t.Fee != nil {
		if err := t.Fee.Validate(); err != nil {
			return err
		}

		if t.Fee.IsNegative() {
			return NewValidationError("taxa não pode ser negativa")
		}

		// Verificar se taxa e valor estão na mesma moeda
		if t.Fee.Currency() != t.Amount.Currency() {
			return NewValidationError("taxa e valor devem estar na mesma moeda")
		}
	}

	// Validar valor líquido se fornecido
	if t.NetAmount != nil {
		if err := t.NetAmount.Validate(); err != nil {
			return err
		}

		if t.NetAmount.Currency() != t.Amount.Currency() {
			return NewValidationError("valor líquido e valor devem estar na mesma moeda")
		}
	}

	// Validar taxa de câmbio
	if t.ExchangeRate <= 0 {
		return NewValidationError("taxa de câmbio deve ser positiva")
	}

	// Validar score de risco
	if t.RiskScore < 0 || t.RiskScore > 100 {
		return NewValidationError("score de risco deve estar entre 0 e 100")
	}

	return nil
}
