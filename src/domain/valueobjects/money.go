package valueobjects

import (
	"fmt"
	"math"
	"math/big"

	"suno-wallets/src/domain/enums"
)

// Money representa um valor monetário com precisão e moeda
type Money struct {
	amount   *big.Int           // Valor em menor unidade (centavos, satoshis, etc.)
	currency enums.CurrencyType // Tipo de moeda
	scale    int                // Número de casas decimais
}

// NewMoney cria um novo value object Money
func NewMoney(amount int64, currency enums.CurrencyType) (*Money, error) {
	if !currency.IsValid() {
		return nil, NewValidationError("tipo de moeda inválido")
	}

	scale := currency.GetDecimalPlaces()

	return &Money{
		amount:   big.NewInt(amount),
		currency: currency,
		scale:    scale,
	}, nil
}

// NewMoneyFromFloat cria Money a partir de um valor float
func NewMoneyFromFloat(amount float64, currency enums.CurrencyType) (*Money, error) {
	if !currency.IsValid() {
		return nil, NewValidationError("tipo de moeda inválido")
	}

	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return nil, NewValidationError("valor monetário inválido")
	}

	scale := currency.GetDecimalPlaces()
	multiplier := math.Pow10(scale)
	intAmount := int64(math.Round(amount * multiplier))

	return &Money{
		amount:   big.NewInt(intAmount),
		currency: currency,
		scale:    scale,
	}, nil
}

// NewMoneyFromString cria Money a partir de uma string
func NewMoneyFromString(amount string, currency enums.CurrencyType) (*Money, error) {
	if !currency.IsValid() {
		return nil, NewValidationError("tipo de moeda inválido")
	}

	bigAmount := new(big.Int)
	_, ok := bigAmount.SetString(amount, 10)
	if !ok {
		return nil, NewValidationError("valor monetário inválido")
	}

	scale := currency.GetDecimalPlaces()

	return &Money{
		amount:   bigAmount,
		currency: currency,
		scale:    scale,
	}, nil
}

// MustNewMoney cria Money ou entra em pânico se inválido (para testes)
func MustNewMoney(amount int64, currency enums.CurrencyType) *Money {
	m, err := NewMoney(amount, currency)
	if err != nil {
		panic(fmt.Sprintf("money inválido: %v", err))
	}
	return m
}

// Zero retorna um Money com valor zero na moeda especificada
func Zero(currency enums.CurrencyType) *Money {
	money, _ := NewMoney(0, currency)
	return money
}

// Amount retorna o valor em menor unidade (centavos, satoshis)
func (m *Money) Amount() *big.Int {
	return new(big.Int).Set(m.amount)
}

// Currency retorna o tipo de moeda
func (m *Money) Currency() enums.CurrencyType {
	return m.currency
}

// Scale retorna o número de casas decimais
func (m *Money) Scale() int {
	return m.scale
}

// ToFloat retorna o valor como float64
func (m *Money) ToFloat() float64 {
	if m.scale == 0 {
		return float64(m.amount.Int64())
	}

	divisor := math.Pow10(m.scale)
	return float64(m.amount.Int64()) / divisor
}

// String implementa fmt.Stringer
func (m *Money) String() string {
	return m.Formatted()
}

// Formatted retorna o valor formatado com símbolo da moeda
func (m *Money) Formatted() string {
	value := m.ToFloat()
	symbol := m.currency.GetSymbol()

	if m.scale == 0 {
		return fmt.Sprintf("%s %.0f", symbol, value)
	}

	format := fmt.Sprintf("%s %%.%df", symbol, m.scale)
	return fmt.Sprintf(format, value)
}

// IsZero verifica se o valor é zero
func (m *Money) IsZero() bool {
	return m.amount.Sign() == 0
}

// IsPositive verifica se o valor é positivo
func (m *Money) IsPositive() bool {
	return m.amount.Sign() > 0
}

// IsNegative verifica se o valor é negativo
func (m *Money) IsNegative() bool {
	return m.amount.Sign() < 0
}

// Abs retorna o valor absoluto
func (m *Money) Abs() *Money {
	absAmount := new(big.Int).Abs(m.amount)
	return &Money{
		amount:   absAmount,
		currency: m.currency,
		scale:    m.scale,
	}
}

// Negate retorna o valor negativo
func (m *Money) Negate() *Money {
	negAmount := new(big.Int).Neg(m.amount)
	return &Money{
		amount:   negAmount,
		currency: m.currency,
		scale:    m.scale,
	}
}

// Add soma dois valores monetários da mesma moeda
func (m *Money) Add(other *Money) (*Money, error) {
	if err := m.validateSameCurrency(other); err != nil {
		return nil, err
	}

	result := new(big.Int).Add(m.amount, other.amount)
	return &Money{
		amount:   result,
		currency: m.currency,
		scale:    m.scale,
	}, nil
}

// Subtract subtrai dois valores monetários da mesma moeda
func (m *Money) Subtract(other *Money) (*Money, error) {
	if err := m.validateSameCurrency(other); err != nil {
		return nil, err
	}

	result := new(big.Int).Sub(m.amount, other.amount)
	return &Money{
		amount:   result,
		currency: m.currency,
		scale:    m.scale,
	}, nil
}

// Multiply multiplica o valor por um fator
func (m *Money) Multiply(factor float64) (*Money, error) {
	if math.IsNaN(factor) || math.IsInf(factor, 0) {
		return nil, NewValidationError("fator de multiplicação inválido")
	}

	// Converte factor para big.Rat para precisão
	factorRat := big.NewRat(1, 1)
	factorRat.SetFloat64(factor)

	// Multiplica amount por factor
	amountRat := new(big.Rat).SetInt(m.amount)
	result := new(big.Rat).Mul(amountRat, factorRat)

	// Converte de volta para big.Int
	resultInt, _ := result.Float64()
	finalAmount := big.NewInt(int64(math.Round(resultInt)))

	return &Money{
		amount:   finalAmount,
		currency: m.currency,
		scale:    m.scale,
	}, nil
}

// Divide divide o valor por um divisor
func (m *Money) Divide(divisor float64) (*Money, error) {
	if divisor == 0 {
		return nil, NewValidationError("divisão por zero")
	}

	return m.Multiply(1 / divisor)
}

// Compare compara dois valores monetários da mesma moeda
// Retorna: -1 se m < other, 0 se m == other, 1 se m > other
func (m *Money) Compare(other *Money) (int, error) {
	if err := m.validateSameCurrency(other); err != nil {
		return 0, err
	}

	return m.amount.Cmp(other.amount), nil
}

// Equals verifica se dois valores monetários são iguais
func (m *Money) Equals(other *Money) bool {
	if other == nil {
		return false
	}

	if m.currency != other.currency {
		return false
	}

	return m.amount.Cmp(other.amount) == 0
}

// GreaterThan verifica se este valor é maior que outro
func (m *Money) GreaterThan(other *Money) (bool, error) {
	cmp, err := m.Compare(other)
	return cmp > 0, err
}

// GreaterThanOrEqual verifica se este valor é maior ou igual a outro
func (m *Money) GreaterThanOrEqual(other *Money) (bool, error) {
	cmp, err := m.Compare(other)
	return cmp >= 0, err
}

// LessThan verifica se este valor é menor que outro
func (m *Money) LessThan(other *Money) (bool, error) {
	cmp, err := m.Compare(other)
	return cmp < 0, err
}

// LessThanOrEqual verifica se este valor é menor ou igual a outro
func (m *Money) LessThanOrEqual(other *Money) (bool, error) {
	cmp, err := m.Compare(other)
	return cmp <= 0, err
}

// validateSameCurrency verifica se duas moedas são iguais
func (m *Money) validateSameCurrency(other *Money) error {
	if other == nil {
		return NewValidationError("valor monetário não pode ser nulo")
	}

	if m.currency != other.currency {
		return NewValidationError(
			fmt.Sprintf("moedas diferentes: %s != %s", m.currency, other.currency),
		)
	}

	return nil
}

// Validate valida o valor monetário
func (m *Money) Validate() error {
	if !m.currency.IsValid() {
		return NewValidationError("tipo de moeda inválido")
	}

	if m.amount == nil {
		return NewValidationError("valor monetário não pode ser nulo")
	}

	return nil
}

// MarshalJSON implementa json.Marshaler
func (m *Money) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"amount":"%s","currency":"%s","formatted":"%s"}`,
		m.amount.String(), m.currency, m.Formatted())), nil
}

// Split divide o valor em partes iguais (para divisão de custos)
func (m *Money) Split(parts int) ([]*Money, *Money, error) {
	if parts <= 0 {
		return nil, nil, NewValidationError("número de partes deve ser positivo")
	}

	if parts == 1 {
		return []*Money{m}, Zero(m.currency), nil
	}

	partAmount := new(big.Int).Div(m.amount, big.NewInt(int64(parts)))
	remainder := new(big.Int).Mod(m.amount, big.NewInt(int64(parts)))

	results := make([]*Money, parts)
	for i := 0; i < parts; i++ {
		results[i] = &Money{
			amount:   new(big.Int).Set(partAmount),
			currency: m.currency,
			scale:    m.scale,
		}
	}

	remainderMoney := &Money{
		amount:   remainder,
		currency: m.currency,
		scale:    m.scale,
	}

	return results, remainderMoney, nil
}
