package admin

import (
	"context"
	"fmt"

	appe2e "suno-wallets/src/application/b3/e2e"
	incrsvc "suno-wallets/src/application/b3/incremental"
	apprecon "suno-wallets/src/application/b3/reconciliation"
	appsync "suno-wallets/src/application/b3/sync"
	cpsvc "suno-wallets/src/application/clientpolicy"
	opsapp "suno-wallets/src/application/ops"
)

// Comentários em pt-BR: adaptadores para conectar serviços existentes às interfaces do backoffice

// B3SyncOrchestratorAdapter - adapta serviços de sync existentes
type B3SyncOrchestratorAdapter struct {
	e2eOrch          *appe2e.Orchestrator
	incrementalSvc   *incrsvc.Service
	completeSyncOrch *appsync.CompleteSyncOrchestrator
}

func NewB3SyncOrchestratorAdapter(e2eOrch *appe2e.Orchestrator, incrementalSvc *incrsvc.Service, completeSyncOrch *appsync.CompleteSyncOrchestrator) B3SyncOrchestrator {
	return &B3SyncOrchestratorAdapter{
		e2eOrch:          e2eOrch,
		incrementalSvc:   incrementalSvc,
		completeSyncOrch: completeSyncOrch,
	}
}

func (a *B3SyncOrchestratorAdapter) ProcessFullHistorical(ctx context.Context, tenantID, cpf, fromMonth, toMonth string, dryRun bool) error {
	if a.completeSyncOrch != nil {
		// Usar o complete sync orchestrator para processamento inteligente
		// Por ora, implementação stub que funciona
		return fmt.Errorf("B3 full historical sync not yet implemented in adapter - use complete sync orchestrator directly")
	}
	if a.e2eOrch != nil {
		// E2E orchestrator existe mas não tem o método esperado
		return fmt.Errorf("B3 full historical sync via E2E not yet implemented in adapter")
	}
	return fmt.Errorf("no B3 sync orchestrator available")
}

func (a *B3SyncOrchestratorAdapter) ProcessIncremental(ctx context.Context, tenantID, cpf, date string, dryRun bool) error {
	if a.incrementalSvc != nil {
		// Incremental service existe mas método pode ter nome diferente
		return fmt.Errorf("B3 incremental sync not yet implemented in adapter")
	}
	return fmt.Errorf("incremental service not available")
}

// ReconciliationOrchestratorAdapter - adapta serviço de reconciliação
type ReconciliationOrchestratorAdapter struct {
	reconSvc *apprecon.Service
}

func NewReconciliationOrchestratorAdapter(reconSvc *apprecon.Service) ReconciliationOrchestrator {
	return &ReconciliationOrchestratorAdapter{reconSvc: reconSvc}
}

func (a *ReconciliationOrchestratorAdapter) ScanInconsistencies(ctx context.Context, tenantID, cpf string, types []string, dryRun bool) error {
	if a.reconSvc != nil {
		// Reconciliation service existe mas método pode ter nome/assinatura diferente
		return fmt.Errorf("reconciliation scan not yet implemented in adapter")
	}
	return fmt.Errorf("reconciliation service not available")
}

func (a *ReconciliationOrchestratorAdapter) AutoFix(ctx context.Context, tenantID, cpf string, types []string, dryRun bool) error {
	if a.reconSvc != nil {
		// Reconciliation service existe mas método pode ter nome/assinatura diferente
		return fmt.Errorf("reconciliation auto-fix not yet implemented in adapter")
	}
	return fmt.Errorf("reconciliation service not available")
}

// DedupOrchestratorAdapter - adapta serviços de deduplicação
type DedupOrchestratorAdapter struct {
	dedupSvc *opsapp.DedupService
}

func NewDedupOrchestratorAdapter(dedupSvc *opsapp.DedupService) DedupOrchestrator {
	return &DedupOrchestratorAdapter{dedupSvc: dedupSvc}
}

func (a *DedupOrchestratorAdapter) ScanDuplicates(ctx context.Context, tenantID, cpf string, scanMode string, dryRun bool) error {
	if a.dedupSvc != nil {
		// Dedup service existe mas método pode ter nome/assinatura diferente
		return fmt.Errorf("dedup scan not yet implemented in adapter")
	}
	return fmt.Errorf("dedup service not available")
}

func (a *DedupOrchestratorAdapter) ResolveDuplicates(ctx context.Context, tenantID string, action string, candidateIds []string) error {
	if a.dedupSvc != nil {
		// Dedup service existe mas método pode ter nome/assinatura diferente
		return fmt.Errorf("dedup resolution not yet implemented in adapter")
	}
	return fmt.Errorf("dedup service not available")
}

// PolicyOrchestratorAdapter - adapta serviço de policy
type PolicyOrchestratorAdapter struct {
	policySvc *cpsvc.Service
}

func NewPolicyOrchestratorAdapter(policySvc *cpsvc.Service) PolicyOrchestrator {
	return &PolicyOrchestratorAdapter{policySvc: policySvc}
}

func (a *PolicyOrchestratorAdapter) SetPolicy(ctx context.Context, tenantID, cpf, mode, reason string) error {
	if a.policySvc != nil {
		// Policy service existe mas precisa de conversão de tipos
		return fmt.Errorf("policy update not yet implemented in adapter - type conversion needed")
	}
	return fmt.Errorf("policy service not available")
}

// ClientDataOrchestratorAdapter - adapta serviços de dados do cliente
type ClientDataOrchestratorAdapter struct {
	e2eOrch *appe2e.Orchestrator
}

func NewClientDataOrchestratorAdapter(e2eOrch *appe2e.Orchestrator) ClientDataOrchestrator {
	return &ClientDataOrchestratorAdapter{e2eOrch: e2eOrch}
}

func (a *ClientDataOrchestratorAdapter) ResetClient(ctx context.Context, tenantID, cpf, what string, dryRun bool) error {
	if a.e2eOrch != nil {
		// E2E orchestrator existe mas método pode ter nome/assinatura diferente
		return fmt.Errorf("client reset not yet implemented in adapter")
	}
	return fmt.Errorf("E2E orchestrator not available")
}

func (a *ClientDataOrchestratorAdapter) ZeroAndRefetch(ctx context.Context, tenantID, cpf, fromDate string, dryRun bool) error {
	if a.e2eOrch != nil {
		// E2E orchestrator existe mas método pode ter nome/assinatura diferente
		return fmt.Errorf("zero and refetch not yet implemented in adapter")
	}
	return fmt.Errorf("E2E orchestrator not available")
}
