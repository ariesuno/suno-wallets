package enums

// AssetType representa os tipos de ativos disponíveis no sistema
type AssetType string

const (
	// AssetTypeCash dinheiro em espécie (moeda fiduciária)
	AssetTypeCash AssetType = "cash"

	// AssetTypeCryptocurrency criptomoeda
	AssetTypeCryptocurrency AssetType = "cryptocurrency"

	// AssetTypeStock ação de empresa
	AssetTypeStock AssetType = "stock"

	// AssetTypeBond título de renda fixa
	AssetTypeBond AssetType = "bond"

	// AssetTypeFund fundo de investimento
	AssetTypeFund AssetType = "fund"

	// AssetTypeETF exchange-traded fund
	AssetTypeETF AssetType = "etf"

	// AssetTypeCommodity commodity (ouro, prata, petróleo, etc.)
	AssetTypeCommodity AssetType = "commodity"

	// AssetTypeRealEstate investimento imobiliário
	AssetTypeRealEstate AssetType = "real_estate"

	// AssetTypeDerivative derivativo financeiro
	AssetTypeDerivative AssetType = "derivative"

	// AssetTypeStablecoin criptomoeda estável
	AssetTypeStablecoin AssetType = "stablecoin"
)

// IsValid verifica se o tipo de ativo é válido
func (at AssetType) IsValid() bool {
	switch at {
	case AssetTypeCash, AssetTypeCryptocurrency, AssetTypeStock, AssetTypeBond,
		AssetTypeFund, AssetTypeETF, AssetTypeCommodity, AssetTypeRealEstate,
		AssetTypeDerivative, AssetTypeStablecoin:
		return true
	default:
		return false
	}
}

// String retorna a representação string do tipo de ativo
func (at AssetType) String() string {
	return string(at)
}

// IsDigital verifica se o ativo é digital
func (at AssetType) IsDigital() bool {
	switch at {
	case AssetTypeCryptocurrency, AssetTypeStablecoin:
		return true
	default:
		return false
	}
}

// IsTraditional verifica se é um ativo tradicional do mercado financeiro
func (at AssetType) IsTraditional() bool {
	switch at {
	case AssetTypeStock, AssetTypeBond, AssetTypeFund, AssetTypeETF, AssetTypeDerivative:
		return true
	default:
		return false
	}
}

// IsPhysical verifica se é um ativo físico
func (at AssetType) IsPhysical() bool {
	switch at {
	case AssetTypeCommodity, AssetTypeRealEstate:
		return true
	default:
		return false
	}
}

// RequiresKYC verifica se o tipo de ativo requer verificação KYC
func (at AssetType) RequiresKYC() bool {
	// Cash geralmente não requer KYC para pequenos valores
	return at != AssetTypeCash
}

// GetDisplayName retorna o nome amigável do tipo de ativo
func (at AssetType) GetDisplayName() string {
	switch at {
	case AssetTypeCash:
		return "Dinheiro"
	case AssetTypeCryptocurrency:
		return "Criptomoeda"
	case AssetTypeStock:
		return "Ação"
	case AssetTypeBond:
		return "Título"
	case AssetTypeFund:
		return "Fundo"
	case AssetTypeETF:
		return "ETF"
	case AssetTypeCommodity:
		return "Commodity"
	case AssetTypeRealEstate:
		return "Imóvel"
	case AssetTypeDerivative:
		return "Derivativo"
	case AssetTypeStablecoin:
		return "Stablecoin"
	default:
		return string(at)
	}
}

// GetAllAssetTypes retorna todos os tipos de ativos válidos
func GetAllAssetTypes() []AssetType {
	return []AssetType{
		AssetTypeCash,
		AssetTypeCryptocurrency,
		AssetTypeStock,
		AssetTypeBond,
		AssetTypeFund,
		AssetTypeETF,
		AssetTypeCommodity,
		AssetTypeRealEstate,
		AssetTypeDerivative,
		AssetTypeStablecoin,
	}
}
