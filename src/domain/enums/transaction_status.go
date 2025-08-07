package enums

// TransactionStatus representa os status possíveis de uma transação
type TransactionStatus string

const (
	// TransactionStatusPending transação criada mas não processada
	TransactionStatusPending TransactionStatus = "pending"

	// TransactionStatusProcessing transação em processamento
	TransactionStatusProcessing TransactionStatus = "processing"

	// TransactionStatusCompleted transação concluída com sucesso
	TransactionStatusCompleted TransactionStatus = "completed"

	// TransactionStatusFailed transação falhou
	TransactionStatusFailed TransactionStatus = "failed"

	// TransactionStatusCancelled transação cancelada pelo usuário
	TransactionStatusCancelled TransactionStatus = "cancelled"

	// TransactionStatusReversed transação revertida/estornada
	TransactionStatusReversed TransactionStatus = "reversed"

	// TransactionStatusOnHold transação em espera (precisa aprovação)
	TransactionStatusOnHold TransactionStatus = "on_hold"

	// TransactionStatusExpired transação expirou
	TransactionStatusExpired TransactionStatus = "expired"

	// TransactionStatusRejected transação rejeitada por compliance
	TransactionStatusRejected TransactionStatus = "rejected"
)

// IsValid verifica se o status da transação é válido
func (ts TransactionStatus) IsValid() bool {
	switch ts {
	case TransactionStatusPending, TransactionStatusProcessing, TransactionStatusCompleted,
		TransactionStatusFailed, TransactionStatusCancelled, TransactionStatusReversed,
		TransactionStatusOnHold, TransactionStatusExpired, TransactionStatusRejected:
		return true
	default:
		return false
	}
}

// String retorna a representação string do status da transação
func (ts TransactionStatus) String() string {
	return string(ts)
}

// IsFinal verifica se o status é final (não pode mais ser alterado)
func (ts TransactionStatus) IsFinal() bool {
	switch ts {
	case TransactionStatusCompleted, TransactionStatusFailed, TransactionStatusCancelled,
		TransactionStatusReversed, TransactionStatusExpired, TransactionStatusRejected:
		return true
	default:
		return false
	}
}

// IsSuccessful verifica se a transação foi bem-sucedida
func (ts TransactionStatus) IsSuccessful() bool {
	return ts == TransactionStatusCompleted
}

// CanBeCancelled verifica se a transação pode ser cancelada
func (ts TransactionStatus) CanBeCancelled() bool {
	switch ts {
	case TransactionStatusPending, TransactionStatusOnHold:
		return true
	default:
		return false
	}
}

// CanBeReversed verifica se a transação pode ser revertida
func (ts TransactionStatus) CanBeReversed() bool {
	return ts == TransactionStatusCompleted
}

// RequiresAction verifica se o status requer ação
func (ts TransactionStatus) RequiresAction() bool {
	switch ts {
	case TransactionStatusOnHold, TransactionStatusFailed:
		return true
	default:
		return false
	}
}

// GetDisplayName retorna o nome amigável do status
func (ts TransactionStatus) GetDisplayName() string {
	switch ts {
	case TransactionStatusPending:
		return "Pendente"
	case TransactionStatusProcessing:
		return "Processando"
	case TransactionStatusCompleted:
		return "Concluída"
	case TransactionStatusFailed:
		return "Falhou"
	case TransactionStatusCancelled:
		return "Cancelada"
	case TransactionStatusReversed:
		return "Estornada"
	case TransactionStatusOnHold:
		return "Em Espera"
	case TransactionStatusExpired:
		return "Expirada"
	case TransactionStatusRejected:
		return "Rejeitada"
	default:
		return string(ts)
	}
}

// GetDescription retorna a descrição do status
func (ts TransactionStatus) GetDescription() string {
	switch ts {
	case TransactionStatusPending:
		return "Transação aguardando processamento"
	case TransactionStatusProcessing:
		return "Transação sendo processada no momento"
	case TransactionStatusCompleted:
		return "Transação concluída com sucesso"
	case TransactionStatusFailed:
		return "Transação falhou durante o processamento"
	case TransactionStatusCancelled:
		return "Transação cancelada pelo usuário"
	case TransactionStatusReversed:
		return "Transação foi estornada"
	case TransactionStatusOnHold:
		return "Transação aguardando aprovação manual"
	case TransactionStatusExpired:
		return "Transação expirou por timeout"
	case TransactionStatusRejected:
		return "Transação rejeitada por política de compliance"
	default:
		return string(ts)
	}
}

// GetColor retorna a cor associada ao status (para UI)
func (ts TransactionStatus) GetColor() string {
	switch ts {
	case TransactionStatusPending, TransactionStatusProcessing:
		return "orange"
	case TransactionStatusCompleted:
		return "green"
	case TransactionStatusFailed, TransactionStatusRejected:
		return "red"
	case TransactionStatusCancelled, TransactionStatusExpired:
		return "gray"
	case TransactionStatusReversed:
		return "blue"
	case TransactionStatusOnHold:
		return "yellow"
	default:
		return "gray"
	}
}
