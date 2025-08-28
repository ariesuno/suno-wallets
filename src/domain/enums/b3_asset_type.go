package enums

// B3AssetType representa os tipos de ativos específicos da API B3
type B3AssetType string

const (
	// B3AssetTypeEquities ações negociadas na B3
	B3AssetTypeEquities B3AssetType = "equities"

	// B3AssetTypeFixedIncome instrumentos de renda fixa
	B3AssetTypeFixedIncome B3AssetType = "fixed-income"

	// B3AssetTypeTreasuryBonds títulos do tesouro direto
	B3AssetTypeTreasuryBonds B3AssetType = "treasury-bonds"

	// B3AssetTypeDerivatives derivativos (opções, futuros, etc.)
	B3AssetTypeDerivatives B3AssetType = "derivatives"

	// B3AssetTypeSecuritiesLending empréstimos de valores mobiliários
	B3AssetTypeSecuritiesLending B3AssetType = "securities-lending"
)

// IsValid verifica se o tipo de ativo B3 é válido
func (bt B3AssetType) IsValid() bool {
	switch bt {
	case B3AssetTypeEquities, B3AssetTypeFixedIncome, B3AssetTypeTreasuryBonds,
		B3AssetTypeDerivatives, B3AssetTypeSecuritiesLending:
		return true
	default:
		return false
	}
}

// String retorna a representação string do tipo de ativo B3
func (bt B3AssetType) String() string {
	return string(bt)
}

// GetAPIEndpoint retorna o endpoint específico da API B3 para este tipo de ativo
func (bt B3AssetType) GetAPIEndpoint() string {
	switch bt {
	case B3AssetTypeEquities:
		return "/equities/investors"
	case B3AssetTypeFixedIncome:
		return "/fixed-income/investors"
	case B3AssetTypeTreasuryBonds:
		return "/treasury-bonds/investors"
	case B3AssetTypeDerivatives:
		return "/derivatives/investors"
	case B3AssetTypeSecuritiesLending:
		return "/securities-lending/investors"
	default:
		return ""
	}
}

// GetPositionsEndpoint retorna o endpoint específico para posições (se diferente)
func (bt B3AssetType) GetPositionsEndpoint() string {
	// Para equities, mantemos o endpoint v3 existente que já funciona
	if bt == B3AssetTypeEquities {
		return "/position/v3/equities/investors"
	}
	// Para outros tipos, usamos o mesmo endpoint base
	return bt.GetAPIEndpoint()
}

// GetDisplayName retorna o nome amigável do tipo de ativo B3
func (bt B3AssetType) GetDisplayName() string {
	switch bt {
	case B3AssetTypeEquities:
		return "Ações"
	case B3AssetTypeFixedIncome:
		return "Renda Fixa"
	case B3AssetTypeTreasuryBonds:
		return "Títulos do Tesouro"
	case B3AssetTypeDerivatives:
		return "Derivativos"
	case B3AssetTypeSecuritiesLending:
		return "Empréstimos de Valores Mobiliários"
	default:
		return string(bt)
	}
}

// RequiresSpecialHandling verifica se o tipo de ativo requer tratamento especial
func (bt B3AssetType) RequiresSpecialHandling() bool {
	switch bt {
	case B3AssetTypeSecuritiesLending:
		// Empréstimos podem ter estrutura de dados diferente
		return true
	default:
		return false
	}
}

// IsTransactionSupported verifica se o tipo suporta busca de transações
func (bt B3AssetType) IsTransactionSupported() bool {
	// Todos os tipos suportam transações na API B3
	return true
}

// IsPositionSupported verifica se o tipo suporta busca de posições
func (bt B3AssetType) IsPositionSupported() bool {
	// Nem todos os tipos podem ter posições (ex: treasury bonds são geralmente para vencimento)
	switch bt {
	case B3AssetTypeEquities, B3AssetTypeDerivatives, B3AssetTypeSecuritiesLending:
		return true
	case B3AssetTypeFixedIncome, B3AssetTypeTreasuryBonds:
		// Renda fixa e tesouro podem ter posições, mas estrutura pode ser diferente
		return true
	default:
		return false
	}
}

// GetAllB3AssetTypes retorna todos os tipos de ativos B3 válidos
func GetAllB3AssetTypes() []B3AssetType {
	return []B3AssetType{
		B3AssetTypeEquities,
		B3AssetTypeFixedIncome,
		B3AssetTypeTreasuryBonds,
		B3AssetTypeDerivatives,
		B3AssetTypeSecuritiesLending,
	}
}

// MapFromGenericAssetType mapeia um AssetType genérico para B3AssetType
func MapFromGenericAssetType(genericType AssetType) []B3AssetType {
	switch genericType {
	case AssetTypeStock:
		return []B3AssetType{B3AssetTypeEquities}
	case AssetTypeBond:
		return []B3AssetType{B3AssetTypeFixedIncome, B3AssetTypeTreasuryBonds}
	case AssetTypeDerivative:
		return []B3AssetType{B3AssetTypeDerivatives}
	case AssetTypeFund, AssetTypeETF:
		// Fundos podem estar tanto em equities quanto em renda fixa
		return []B3AssetType{B3AssetTypeEquities, B3AssetTypeFixedIncome}
	default:
		// Para tipos não mapeados, usar equities como fallback
		return []B3AssetType{B3AssetTypeEquities}
	}
}

// ParseB3AssetType converte string para B3AssetType com validação
func ParseB3AssetType(s string) (B3AssetType, bool) {
	assetType := B3AssetType(s)
	return assetType, assetType.IsValid()
}
