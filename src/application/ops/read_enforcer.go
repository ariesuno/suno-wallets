package ops

import (
	"context"
	pol "suno-wallets/src/application/clientpolicy"
	dom "suno-wallets/src/domain/clientpolicy"
)

// Comentários em pt-BR: enforcer de leitura — ajusta filtros conforme policy

type ReadEnforcer struct{ pol *pol.Service }

func NewReadEnforcer(s *pol.Service) *ReadEnforcer { return &ReadEnforcer{pol: s} }

func (e *ReadEnforcer) Apply(ctx context.Context, tenantID, cpf string, f *TimelineFilters) {
	mode := e.pol.GetMode(ctx, tenantID, cpf)
	if mode == dom.ModeManualOnly {
		// ignorar B3_RAW no read-side por padrão
		filtered := make([]string, 0, len(f.Sources))
		if len(f.Sources) == 0 {
			f.Sources = []string{"USER_MANUAL", "SYSTEM_SYNTHETIC"}
			return
		}
		for _, s := range f.Sources {
			if s != "B3_RAW" {
				filtered = append(filtered, s)
			}
		}
		f.Sources = filtered
	}
}
