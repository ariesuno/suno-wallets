package enums

// UserStatus representa os status possíveis de um usuário no sistema
type UserStatus string

const (
	// UserStatusPending usuário criado mas ainda não verificado
	UserStatusPending UserStatus = "pending"

	// UserStatusActive usuário ativo e verificado
	UserStatusActive UserStatus = "active"

	// UserStatusInactive usuário temporariamente inativo
	UserStatusInactive UserStatus = "inactive"

	// UserStatusSuspended usuário suspenso por violação de termos
	UserStatusSuspended UserStatus = "suspended"

	// UserStatusBlocked usuário bloqueado por segurança
	UserStatusBlocked UserStatus = "blocked"

	// UserStatusDeleted usuário excluído (soft delete)
	UserStatusDeleted UserStatus = "deleted"

	// UserStatusKYCRequired usuário que precisa completar KYC
	UserStatusKYCRequired UserStatus = "kyc_required"

	// UserStatusKYCPending usuário com KYC em análise
	UserStatusKYCPending UserStatus = "kyc_pending"

	// UserStatusKYCRejected usuário com KYC rejeitado
	UserStatusKYCRejected UserStatus = "kyc_rejected"
)

// IsValid verifica se o status do usuário é válido
func (us UserStatus) IsValid() bool {
	switch us {
	case UserStatusPending, UserStatusActive, UserStatusInactive, UserStatusSuspended,
		UserStatusBlocked, UserStatusDeleted, UserStatusKYCRequired, UserStatusKYCPending,
		UserStatusKYCRejected:
		return true
	default:
		return false
	}
}

// String retorna a representação string do status do usuário
func (us UserStatus) String() string {
	return string(us)
}

// CanAccess verifica se o usuário pode acessar o sistema
func (us UserStatus) CanAccess() bool {
	switch us {
	case UserStatusActive, UserStatusKYCRequired:
		return true
	default:
		return false
	}
}

// CanTrade verifica se o usuário pode realizar transações
func (us UserStatus) CanTrade() bool {
	return us == UserStatusActive
}

// RequiresAction verifica se o status requer ação do usuário
func (us UserStatus) RequiresAction() bool {
	switch us {
	case UserStatusPending, UserStatusKYCRequired, UserStatusKYCRejected:
		return true
	default:
		return false
	}
}

// IsBlocked verifica se o usuário está bloqueado
func (us UserStatus) IsBlocked() bool {
	switch us {
	case UserStatusSuspended, UserStatusBlocked, UserStatusDeleted:
		return true
	default:
		return false
	}
}

// GetDisplayName retorna o nome amigável do status
func (us UserStatus) GetDisplayName() string {
	switch us {
	case UserStatusPending:
		return "Aguardando Verificação"
	case UserStatusActive:
		return "Ativo"
	case UserStatusInactive:
		return "Inativo"
	case UserStatusSuspended:
		return "Suspenso"
	case UserStatusBlocked:
		return "Bloqueado"
	case UserStatusDeleted:
		return "Excluído"
	case UserStatusKYCRequired:
		return "KYC Necessário"
	case UserStatusKYCPending:
		return "KYC em Análise"
	case UserStatusKYCRejected:
		return "KYC Rejeitado"
	default:
		return string(us)
	}
}

// GetDescription retorna a descrição do status
func (us UserStatus) GetDescription() string {
	switch us {
	case UserStatusPending:
		return "Usuário criado, aguardando verificação de email"
	case UserStatusActive:
		return "Usuário verificado e com acesso completo"
	case UserStatusInactive:
		return "Usuário temporariamente inativo"
	case UserStatusSuspended:
		return "Usuário suspenso por violação de termos de uso"
	case UserStatusBlocked:
		return "Usuário bloqueado por questões de segurança"
	case UserStatusDeleted:
		return "Conta excluída pelo usuário"
	case UserStatusKYCRequired:
		return "Verificação KYC necessária para continuar"
	case UserStatusKYCPending:
		return "Documentos em análise pela equipe de compliance"
	case UserStatusKYCRejected:
		return "Documentos rejeitados, necessário reenvio"
	default:
		return string(us)
	}
}
