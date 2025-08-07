package enums

// CurrencyType representa os tipos de moeda suportados pelo sistema
type CurrencyType string

const (
	// Moedas Fiduciárias
	CurrencyBRL CurrencyType = "BRL" // Real Brasileiro
	CurrencyUSD CurrencyType = "USD" // Dólar Americano
	CurrencyEUR CurrencyType = "EUR" // Euro
	CurrencyGBP CurrencyType = "GBP" // Libra Esterlina
	CurrencyJPY CurrencyType = "JPY" // Iene Japonês
	CurrencyCAD CurrencyType = "CAD" // Dólar Canadense
	CurrencyAUD CurrencyType = "AUD" // Dólar Australiano
	CurrencyCHF CurrencyType = "CHF" // Franco Suíço
	CurrencyCNY CurrencyType = "CNY" // Yuan Chinês
	CurrencyARS CurrencyType = "ARS" // Peso Argentino

	// Criptomoedas Principais
	CurrencyBTC  CurrencyType = "BTC"  // Bitcoin
	CurrencyETH  CurrencyType = "ETH"  // Ethereum
	CurrencyBNB  CurrencyType = "BNB"  // Binance Coin
	CurrencyADA  CurrencyType = "ADA"  // Cardano
	CurrencySOL  CurrencyType = "SOL"  // Solana
	CurrencyDOT  CurrencyType = "DOT"  // Polkadot
	CurrencyLINK CurrencyType = "LINK" // Chainlink

	// Stablecoins
	CurrencyUSDT CurrencyType = "USDT" // Tether
	CurrencyUSDC CurrencyType = "USDC" // USD Coin
	CurrencyBUSD CurrencyType = "BUSD" // Binance USD
	CurrencyDAI  CurrencyType = "DAI"  // Dai Stablecoin
)

// IsValid verifica se o tipo de moeda é válido
func (ct CurrencyType) IsValid() bool {
	switch ct {
	case CurrencyBRL, CurrencyUSD, CurrencyEUR, CurrencyGBP, CurrencyJPY,
		CurrencyCAD, CurrencyAUD, CurrencyCHF, CurrencyCNY, CurrencyARS,
		CurrencyBTC, CurrencyETH, CurrencyBNB, CurrencyADA, CurrencySOL,
		CurrencyDOT, CurrencyLINK, CurrencyUSDT, CurrencyUSDC, CurrencyBUSD,
		CurrencyDAI:
		return true
	default:
		return false
	}
}

// String retorna a representação string do tipo de moeda
func (ct CurrencyType) String() string {
	return string(ct)
}

// IsFiat verifica se é uma moeda fiduciária
func (ct CurrencyType) IsFiat() bool {
	switch ct {
	case CurrencyBRL, CurrencyUSD, CurrencyEUR, CurrencyGBP, CurrencyJPY,
		CurrencyCAD, CurrencyAUD, CurrencyCHF, CurrencyCNY, CurrencyARS:
		return true
	default:
		return false
	}
}

// IsCryptocurrency verifica se é uma criptomoeda
func (ct CurrencyType) IsCryptocurrency() bool {
	return !ct.IsFiat()
}

// IsStablecoin verifica se é uma stablecoin
func (ct CurrencyType) IsStablecoin() bool {
	switch ct {
	case CurrencyUSDT, CurrencyUSDC, CurrencyBUSD, CurrencyDAI:
		return true
	default:
		return false
	}
}

// GetSymbol retorna o símbolo da moeda
func (ct CurrencyType) GetSymbol() string {
	switch ct {
	case CurrencyBRL:
		return "R$"
	case CurrencyUSD:
		return "$"
	case CurrencyEUR:
		return "€"
	case CurrencyGBP:
		return "£"
	case CurrencyJPY:
		return "¥"
	case CurrencyCAD:
		return "C$"
	case CurrencyAUD:
		return "A$"
	case CurrencyCHF:
		return "CHF"
	case CurrencyCNY:
		return "¥"
	case CurrencyARS:
		return "$"
	default:
		return string(ct)
	}
}

// GetDisplayName retorna o nome completo da moeda
func (ct CurrencyType) GetDisplayName() string {
	switch ct {
	case CurrencyBRL:
		return "Real Brasileiro"
	case CurrencyUSD:
		return "Dólar Americano"
	case CurrencyEUR:
		return "Euro"
	case CurrencyGBP:
		return "Libra Esterlina"
	case CurrencyJPY:
		return "Iene Japonês"
	case CurrencyCAD:
		return "Dólar Canadense"
	case CurrencyAUD:
		return "Dólar Australiano"
	case CurrencyCHF:
		return "Franco Suíço"
	case CurrencyCNY:
		return "Yuan Chinês"
	case CurrencyARS:
		return "Peso Argentino"
	case CurrencyBTC:
		return "Bitcoin"
	case CurrencyETH:
		return "Ethereum"
	case CurrencyBNB:
		return "Binance Coin"
	case CurrencyADA:
		return "Cardano"
	case CurrencySOL:
		return "Solana"
	case CurrencyDOT:
		return "Polkadot"
	case CurrencyLINK:
		return "Chainlink"
	case CurrencyUSDT:
		return "Tether"
	case CurrencyUSDC:
		return "USD Coin"
	case CurrencyBUSD:
		return "Binance USD"
	case CurrencyDAI:
		return "Dai Stablecoin"
	default:
		return string(ct)
	}
}

// GetDecimalPlaces retorna o número de casas decimais para a moeda
func (ct CurrencyType) GetDecimalPlaces() int {
	switch ct {
	case CurrencyJPY:
		return 0 // Iene não usa decimais
	case CurrencyBTC:
		return 8 // Bitcoin usa 8 casas decimais
	case CurrencyETH:
		return 18 // Ethereum usa 18 casas decimais (wei)
	default:
		if ct.IsCryptocurrency() {
			return 8 // Padrão para criptomoedas
		}
		return 2 // Padrão para moedas fiduciárias
	}
}

// RequiresKYC verifica se a moeda requer verificação KYC
func (ct CurrencyType) RequiresKYC() bool {
	// Geralmente criptomoedas requerem KYC mais rigoroso
	return ct.IsCryptocurrency()
}

// GetFiatCurrencies retorna todas as moedas fiduciárias
func GetFiatCurrencies() []CurrencyType {
	return []CurrencyType{
		CurrencyBRL, CurrencyUSD, CurrencyEUR, CurrencyGBP, CurrencyJPY,
		CurrencyCAD, CurrencyAUD, CurrencyCHF, CurrencyCNY, CurrencyARS,
	}
}

// GetCryptocurrencies retorna todas as criptomoedas
func GetCryptocurrencies() []CurrencyType {
	return []CurrencyType{
		CurrencyBTC, CurrencyETH, CurrencyBNB, CurrencyADA, CurrencySOL,
		CurrencyDOT, CurrencyLINK, CurrencyUSDT, CurrencyUSDC, CurrencyBUSD,
		CurrencyDAI,
	}
}

// GetStablecoins retorna todas as stablecoins
func GetStablecoins() []CurrencyType {
	return []CurrencyType{
		CurrencyUSDT, CurrencyUSDC, CurrencyBUSD, CurrencyDAI,
	}
}

// GetAllCurrencies retorna todos os tipos de moeda válidos
func GetAllCurrencies() []CurrencyType {
	fiat := GetFiatCurrencies()
	crypto := GetCryptocurrencies()
	all := make([]CurrencyType, 0, len(fiat)+len(crypto))
	all = append(all, fiat...)
	all = append(all, crypto...)
	return all
}
