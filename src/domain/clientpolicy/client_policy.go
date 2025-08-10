package clientpolicy

import "time"

// Comentários em pt-BR: entidade e contratos para política de fonte de dados por cliente

type Mode string

const (
	ModeB3Only     Mode = "B3_ONLY"
	ModeManualOnly      = "MANUAL_ONLY"
	ModeHybrid          = "HYBRID"
)

type ClientPolicy struct {
	TenantID  string
	CPF       string
	Mode      Mode
	Reason    string
	UpdatedAt time.Time
}

type Repository interface {
	Get(tenantID, cpf string) (*ClientPolicy, error)
	Upsert(tenantID, cpf string, mode Mode, reason, actor string) error
	ListAudit(tenantID, cpf string, limit int) ([]map[string]interface{}, error)
}

type Cache interface {
	Get(tenantID, cpf string) (*ClientPolicy, bool)
	Set(tenantID, cpf string, pol *ClientPolicy)
	Invalidate(tenantID, cpf string)
}
