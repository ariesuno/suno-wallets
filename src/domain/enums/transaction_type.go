package enums

// TransactionType representa os tipos de transação disponíveis no sistema
type TransactionType string

const (
	// TransactionTypeDebit débito - saída de dinheiro da carteira
	TransactionTypeDebit TransactionType = "debit"

	// TransactionTypeCredit crédito - entrada de dinheiro na carteira
	TransactionTypeCredit TransactionType = "credit"

	// TransactionTypeTransfer transferência - movimentação entre carteiras
	TransactionTypeTransfer TransactionType = "transfer"

	// TransactionTypeWithdrawal saque - retirada para conta bancária
	TransactionTypeWithdrawal TransactionType = "withdrawal"

	// TransactionTypeDeposit depósito - entrada via conta bancária
	TransactionTypeDeposit TransactionType = "deposit"

	// TransactionTypeFee taxa - cobrança de taxas do sistema
	TransactionTypeFee TransactionType = "fee"

	// TransactionTypeRefund estorno - devolução de valor
	TransactionTypeRefund TransactionType = "refund"

	// TransactionTypeReward recompensa - bônus ou cashback
	TransactionTypeReward TransactionType = "reward"
)

// IsValid verifica se o tipo de transação é válido
func (tt TransactionType) IsValid() bool {
	switch tt {
	case TransactionTypeDebit, TransactionTypeCredit, TransactionTypeTransfer,
		TransactionTypeWithdrawal, TransactionTypeDeposit, TransactionTypeFee,
		TransactionTypeRefund, TransactionTypeReward:
		return true
	default:
		return false
	}
}

// String retorna a representação string do tipo de transação
func (tt TransactionType) String() string {
	return string(tt)
}

// IsOutbound verifica se a transação resulta em saída de dinheiro
func (tt TransactionType) IsOutbound() bool {
	switch tt {
	case TransactionTypeDebit, TransactionTypeWithdrawal, TransactionTypeFee:
		return true
	default:
		return false
	}
}

// IsInbound verifica se a transação resulta em entrada de dinheiro
func (tt TransactionType) IsInbound() bool {
	switch tt {
	case TransactionTypeCredit, TransactionTypeDeposit, TransactionTypeRefund, TransactionTypeReward:
		return true
	default:
		return false
	}
}

// RequiresDestination verifica se o tipo de transação requer uma carteira de destino
func (tt TransactionType) RequiresDestination() bool {
	return tt == TransactionTypeTransfer
}

// GetAllTransactionTypes retorna todos os tipos de transação válidos
func GetAllTransactionTypes() []TransactionType {
	return []TransactionType{
		TransactionTypeDebit,
		TransactionTypeCredit,
		TransactionTypeTransfer,
		TransactionTypeWithdrawal,
		TransactionTypeDeposit,
		TransactionTypeFee,
		TransactionTypeRefund,
		TransactionTypeReward,
	}
}
