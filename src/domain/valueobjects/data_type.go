package valueobjects

import (
	"fmt"
	"strings"
)

// DataType representa um tipo de dados B3 como value object
type DataType struct {
	value string
}

const (
	TransactionsDataType = "transactions"
	PositionsDataType    = "positions"
)

// NewDataType cria novo value object DataType
func NewDataType(dataType string) (*DataType, error) {
	normalized := strings.ToLower(strings.TrimSpace(dataType))

	if normalized == "" {
		return nil, NewValidationError("data type não pode ser vazio")
	}

	if normalized != TransactionsDataType && normalized != PositionsDataType {
		return nil, NewValidationError(fmt.Sprintf("data type deve ser '%s' ou '%s'", TransactionsDataType, PositionsDataType))
	}

	return &DataType{value: normalized}, nil
}

// MustNewDataType cria DataType ou entra em pânico
func MustNewDataType(dataType string) *DataType {
	dt, err := NewDataType(dataType)
	if err != nil {
		panic(fmt.Sprintf("DataType inválido: %v", err))
	}
	return dt
}

// Value retorna o valor string do data type
func (dt *DataType) Value() string {
	return dt.value
}

// String implementa fmt.Stringer
func (dt *DataType) String() string {
	return dt.value
}

// IsTransactions verifica se é tipo transactions
func (dt *DataType) IsTransactions() bool {
	return dt.value == TransactionsDataType
}

// IsPositions verifica se é tipo positions
func (dt *DataType) IsPositions() bool {
	return dt.value == PositionsDataType
}

// Equals verifica igualdade
func (dt *DataType) Equals(other *DataType) bool {
	if other == nil {
		return false
	}
	return dt.value == other.value
}

// Validate valida o value object
func (dt *DataType) Validate() error {
	if dt.value == "" {
		return NewValidationError("data type está vazio")
	}

	if dt.value != TransactionsDataType && dt.value != PositionsDataType {
		return NewValidationError("data type inválido")
	}

	return nil
}
