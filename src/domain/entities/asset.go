package entities

import (
	"time"

	"suno-wallets/src/domain/enums"
	"suno-wallets/src/domain/valueobjects"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Asset representa um ativo financeiro no sistema
type Asset struct {
	// Identificação primária
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`

	// Informações básicas do ativo
	Symbol      string             `json:"symbol" gorm:"type:varchar(20);not null;uniqueIndex:idx_tenant_symbol"`
	Name        string             `json:"name" gorm:"type:varchar(255);not null"`
	Description string             `json:"description" gorm:"type:text"`
	Type        enums.AssetType    `json:"type" gorm:"type:varchar(50);not null"`
	Currency    enums.CurrencyType `json:"currency" gorm:"type:varchar(10);not null"`

	// Informações de preço e mercado
	CurrentPrice    *valueobjects.Money `json:"current_price,omitempty" gorm:"embedded;embeddedPrefix:current_price_"`
	MarketCap       *valueobjects.Money `json:"market_cap,omitempty" gorm:"embedded;embeddedPrefix:market_cap_"`
	Volume24h       *valueobjects.Money `json:"volume_24h,omitempty" gorm:"embedded;embeddedPrefix:volume_24h_"`
	PriceChange24h  float64             `json:"price_change_24h" gorm:"type:decimal(10,4);default:0"`
	PriceChangePerc float64             `json:"price_change_percentage" gorm:"type:decimal(8,4);default:0"`
	LastPriceUpdate *time.Time          `json:"last_price_update,omitempty"`

	// Informações técnicas
	Decimals       int     `json:"decimals" gorm:"type:int;default:18"`
	MinTradeAmount *int64  `json:"min_trade_amount,omitempty" gorm:"type:bigint"` // Menor unidade
	MaxTradeAmount *int64  `json:"max_trade_amount,omitempty" gorm:"type:bigint"`
	TradingFee     float64 `json:"trading_fee" gorm:"type:decimal(8,6);default:0"` // Taxa em percentual
	WithdrawalFee  *int64  `json:"withdrawal_fee,omitempty" gorm:"type:bigint"`

	// Status e configurações
	IsActive       bool `json:"is_active" gorm:"type:boolean;not null;default:true"`
	IsTradeable    bool `json:"is_tradeable" gorm:"type:boolean;not null;default:true"`
	IsWithdrawable bool `json:"is_withdrawable" gorm:"type:boolean;not null;default:true"`
	IsDepositable  bool `json:"is_depositable" gorm:"type:boolean;not null;default:true"`
	RequiresKYC    bool `json:"requires_kyc" gorm:"type:boolean;not null;default:false"`

	// Informações específicas por tipo
	// Para criptomoedas
	ContractAddress string `json:"contract_address,omitempty" gorm:"type:varchar(255)"`
	Blockchain      string `json:"blockchain,omitempty" gorm:"type:varchar(50)"`

	// Para ações e fundos
	ISIN     string `json:"isin,omitempty" gorm:"type:varchar(12)"` // International Securities Identification Number
	Exchange string `json:"exchange,omitempty" gorm:"type:varchar(50)"`
	Sector   string `json:"sector,omitempty" gorm:"type:varchar(100)"`

	// Para commodities
	Unit  string `json:"unit,omitempty" gorm:"type:varchar(20)"` // kg, oz, barrel, etc.
	Grade string `json:"grade,omitempty" gorm:"type:varchar(50)"`

	// Metadados e configurações avançadas
	LogoURL       string                 `json:"logo_url,omitempty" gorm:"type:varchar(500)"`
	WebsiteURL    string                 `json:"website_url,omitempty" gorm:"type:varchar(500)"`
	WhitepaperURL string                 `json:"whitepaper_url,omitempty" gorm:"type:varchar(500)"`
	ExplorerURL   string                 `json:"explorer_url,omitempty" gorm:"type:varchar(500)"`
	Tags          []string               `json:"tags,omitempty" gorm:"type:text[]"`
	RiskRating    string                 `json:"risk_rating" gorm:"type:varchar(20);default:'medium'"` // low, medium, high
	Metadata      map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`

	// Campos de auditoria obrigatórios
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	CreatedBy uuid.UUID      `json:"created_by" gorm:"type:uuid"`
	UpdatedBy *uuid.UUID     `json:"updated_by,omitempty" gorm:"type:uuid"`
}

// TableName define o nome da tabela no banco de dados
func (Asset) TableName() string {
	return "assets"
}

// BeforeCreate hook executado antes da criação
func (a *Asset) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}

	// Definir decimais baseado no tipo de ativo
	if a.Decimals == 0 {
		a.Decimals = a.getDefaultDecimals()
	}

	return nil
}

// IsDigitalAsset verifica se é um ativo digital
func (a *Asset) IsDigitalAsset() bool {
	return a.Type.IsDigital()
}

// IsTraditionalAsset verifica se é um ativo tradicional
func (a *Asset) IsTraditionalAsset() bool {
	return a.Type.IsTraditional()
}

// IsPhysicalAsset verifica se é um ativo físico
func (a *Asset) IsPhysicalAsset() bool {
	return a.Type.IsPhysical()
}

// CanTrade verifica se o ativo pode ser negociado
func (a *Asset) CanTrade() bool {
	return a.IsActive && a.IsTradeable
}

// CanWithdraw verifica se o ativo pode ser sacado
func (a *Asset) CanWithdraw() bool {
	return a.IsActive && a.IsWithdrawable
}

// CanDeposit verifica se o ativo pode ser depositado
func (a *Asset) CanDeposit() bool {
	return a.IsActive && a.IsDepositable
}

// GetTradingFeeForAmount calcula a taxa de negociação para um valor
func (a *Asset) GetTradingFeeForAmount(amount int64) int64 {
	if a.TradingFee == 0 {
		return 0
	}
	return int64(float64(amount) * a.TradingFee / 100)
}

// GetWithdrawalFeeAmount retorna a taxa de saque
func (a *Asset) GetWithdrawalFeeAmount() int64 {
	if a.WithdrawalFee == nil {
		return 0
	}
	return *a.WithdrawalFee
}

// IsMinTradeAmountMet verifica se o valor mínimo de negociação foi atendido
func (a *Asset) IsMinTradeAmountMet(amount int64) bool {
	if a.MinTradeAmount == nil {
		return true
	}
	return amount >= *a.MinTradeAmount
}

// IsMaxTradeAmountExceeded verifica se o valor máximo de negociação foi excedido
func (a *Asset) IsMaxTradeAmountExceeded(amount int64) bool {
	if a.MaxTradeAmount == nil {
		return false
	}
	return amount > *a.MaxTradeAmount
}

// GetCurrentPriceInReais retorna o preço atual em reais
func (a *Asset) GetCurrentPriceInReais() float64 {
	if a.CurrentPrice == nil {
		return 0
	}
	return a.CurrentPrice.ToFloat()
}

// IsVolatile verifica se o ativo é considerado volátil
func (a *Asset) IsVolatile() bool {
	// Considera volátil se mudança percentual > 5%
	return a.PriceChangePerc > 5.0 || a.PriceChangePerc < -5.0
}

// GetMarketCapInReais retorna o market cap em reais
func (a *Asset) GetMarketCapInReais() float64 {
	if a.MarketCap == nil {
		return 0
	}
	return a.MarketCap.ToFloat()
}

// GetVolume24hInReais retorna o volume 24h em reais
func (a *Asset) GetVolume24hInReais() float64 {
	if a.Volume24h == nil {
		return 0
	}
	return a.Volume24h.ToFloat()
}

// IsStablecoin verifica se é uma stablecoin (para criptomoedas)
func (a *Asset) IsStablecoin() bool {
	if !a.IsDigitalAsset() {
		return false
	}
	return a.Currency.IsStablecoin()
}

// HasTag verifica se o ativo tem uma tag específica
func (a *Asset) HasTag(tag string) bool {
	for _, t := range a.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adiciona uma tag ao ativo
func (a *Asset) AddTag(tag string) {
	if !a.HasTag(tag) {
		a.Tags = append(a.Tags, tag)
	}
}

// RemoveTag remove uma tag do ativo
func (a *Asset) RemoveTag(tag string) {
	for i, t := range a.Tags {
		if t == tag {
			a.Tags = append(a.Tags[:i], a.Tags[i+1:]...)
			break
		}
	}
}

// IsHighRisk verifica se o ativo é de alto risco
func (a *Asset) IsHighRisk() bool {
	return a.RiskRating == "high"
}

// UpdatePrice atualiza o preço atual do ativo
func (a *Asset) UpdatePrice(newPrice *valueobjects.Money, change24h, changePerc float64) error {
	if newPrice == nil {
		return NewValidationError("novo preço não pode ser nulo")
	}

	if newPrice.Currency() != a.Currency {
		return NewValidationError("moeda do preço deve corresponder à moeda do ativo")
	}

	a.CurrentPrice = newPrice
	a.PriceChange24h = change24h
	a.PriceChangePerc = changePerc
	now := time.Now()
	a.LastPriceUpdate = &now

	return nil
}

// getDefaultDecimals retorna o número padrão de decimais baseado no tipo de ativo
func (a *Asset) getDefaultDecimals() int {
	switch a.Type {
	case enums.AssetTypeCash:
		return 2 // Centavos
	case enums.AssetTypeCryptocurrency:
		if a.Currency.IsStablecoin() {
			return 6 // Stablecoins geralmente 6 decimais
		}
		return 8 // Bitcoin padrão
	case enums.AssetTypeStock, enums.AssetTypeFund, enums.AssetTypeETF:
		return 2 // Preços em reais/centavos
	case enums.AssetTypeCommodity:
		return 4 // Commodities com mais precisão
	default:
		return 8 // Padrão seguro
	}
}

// Validate valida os dados do ativo
func (a *Asset) Validate() error {
	if a.TenantID == uuid.Nil {
		return NewValidationError("tenant_id é obrigatório")
	}

	if a.Symbol == "" {
		return NewValidationError("símbolo é obrigatório")
	}

	if len(a.Symbol) > 20 {
		return NewValidationError("símbolo não pode ter mais de 20 caracteres")
	}

	if a.Name == "" {
		return NewValidationError("nome é obrigatório")
	}

	if len(a.Name) > 255 {
		return NewValidationError("nome não pode ter mais de 255 caracteres")
	}

	if !a.Type.IsValid() {
		return NewValidationError("tipo de ativo inválido")
	}

	if !a.Currency.IsValid() {
		return NewValidationError("moeda inválida")
	}

	if a.Decimals < 0 || a.Decimals > 18 {
		return NewValidationError("decimais deve estar entre 0 e 18")
	}

	if a.TradingFee < 0 || a.TradingFee > 100 {
		return NewValidationError("taxa de negociação deve estar entre 0 e 100%")
	}

	// Validar valores mínimo e máximo de negociação
	if a.MinTradeAmount != nil && *a.MinTradeAmount < 0 {
		return NewValidationError("valor mínimo de negociação deve ser positivo")
	}

	if a.MaxTradeAmount != nil && *a.MaxTradeAmount < 0 {
		return NewValidationError("valor máximo de negociação deve ser positivo")
	}

	if a.MinTradeAmount != nil && a.MaxTradeAmount != nil && *a.MinTradeAmount > *a.MaxTradeAmount {
		return NewValidationError("valor mínimo não pode ser maior que o máximo")
	}

	// Validações específicas por tipo
	if a.IsDigitalAsset() && a.ContractAddress != "" && len(a.ContractAddress) > 255 {
		return NewValidationError("endereço do contrato não pode ter mais de 255 caracteres")
	}

	if a.ISIN != "" && len(a.ISIN) != 12 {
		return NewValidationError("ISIN deve ter exatamente 12 caracteres")
	}

	return nil
}
