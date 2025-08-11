package enums

// TenantType representa os tenants válidos do sistema
type TenantType string

const (
	TenantNai           TenantType = "nai"
	TenantStatusInvest  TenantType = "status_invest"
	TenantFiis          TenantType = "fiis"
	TenantFundsExplorer TenantType = "funds_explorer"
	TenantOrcana        TenantType = "orcana"
)

// ValidTenants retorna lista de tenants válidos
func ValidTenants() []TenantType {
	return []TenantType{
		TenantNai,
		TenantStatusInvest,
		TenantFiis,
		TenantFundsExplorer,
		TenantOrcana,
	}
}

// IsValid verifica se o tenant é válido
func (t TenantType) IsValid() bool {
	for _, valid := range ValidTenants() {
		if t == valid {
			return true
		}
	}
	return false
}

// String retorna a representação string do tenant
func (t TenantType) String() string {
	return string(t)
}
