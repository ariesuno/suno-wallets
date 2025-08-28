package admin

import (
	"context"
	"encoding/json"
	"fmt"
	appadm "suno-wallets/src/application/admin"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Comentários em pt-BR: repositório para leitura agregada e persistência de auditoria/admin actions

// Interfaces para orquestração dos módulos 1.11-1.21
type B3SyncOrchestrator interface {
	ProcessFullHistorical(ctx context.Context, tenantID, cpf, fromMonth, toMonth string, dryRun bool) error
	ProcessIncremental(ctx context.Context, tenantID, cpf, date string, dryRun bool) error
}

type ReconciliationOrchestrator interface {
	ScanInconsistencies(ctx context.Context, tenantID, cpf string, types []string, dryRun bool) error
	AutoFix(ctx context.Context, tenantID, cpf string, types []string, dryRun bool) error
}

type DedupOrchestrator interface {
	ScanDuplicates(ctx context.Context, tenantID, cpf string, scanMode string, dryRun bool) error
	ResolveDuplicates(ctx context.Context, tenantID string, action string, candidateIds []string) error
}

type PolicyOrchestrator interface {
	SetPolicy(ctx context.Context, tenantID, cpf, mode, reason string) error
}

type ClientDataOrchestrator interface {
	ResetClient(ctx context.Context, tenantID, cpf, what string, dryRun bool) error
	ZeroAndRefetch(ctx context.Context, tenantID, cpf, fromDate string, dryRun bool) error
}

// Estrutura principal com locks de concorrência
type Repository struct {
	db                     *gorm.DB
	b3Orchestrator         B3SyncOrchestrator
	reconOrchestrator      ReconciliationOrchestrator
	dedupOrchestrator      DedupOrchestrator
	policyOrchestrator     PolicyOrchestrator
	clientDataOrchestrator ClientDataOrchestrator

	// Locks por (tenantId, cpf) para prevenir execuções concorrentes
	executionLocks map[string]*sync.Mutex
	locksMutex     sync.Mutex
}

func NewRepository(db *gorm.DB, b3Orch B3SyncOrchestrator, reconOrch ReconciliationOrchestrator, dedupOrch DedupOrchestrator, policyOrch PolicyOrchestrator, clientDataOrch ClientDataOrchestrator) *Repository {
	return &Repository{
		db:                     db,
		b3Orchestrator:         b3Orch,
		reconOrchestrator:      reconOrch,
		dedupOrchestrator:      dedupOrch,
		policyOrchestrator:     policyOrch,
		clientDataOrchestrator: clientDataOrch,
		executionLocks:         make(map[string]*sync.Mutex),
		locksMutex:             sync.Mutex{},
	}
}

func (r *Repository) LoadProfile(ctx context.Context, tenantID, cpf string) (*appadm.Profile, error) {
	// agregação básica reusando dados já disponíveis (placeholders razoáveis)
	type rec struct {
		LastFull *time.Time
		LastIncr *time.Time
	}
	var rw rec
	_ = r.db.WithContext(ctx).Raw(`SELECT MAX(last_pos_sync_at) AS last_full, MAX(last_tx_sync_at) AS last_incr FROM b3_sync_state WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Scan(&rw).Error
	prof := &appadm.Profile{
		CPFMasked:      maskCPF(cpf),
		DataSourceMode: "HYBRID",
		B3:             map[string]interface{}{"lastFullFetchAt": rw.LastFull, "lastIncrementalAt": rw.LastIncr, "latestWindow": map[string]string{"from": "2010-01-01", "to": time.Now().Format("2006-01-02")}},
		Reconciliation: map[string]interface{}{"inconsistencies": map[string]interface{}{"open": 0, "byType": map[string]int{}}, "systemOps": map[string]interface{}{"created": 0, "byReason": map[string]int{}}, "manualOps": 0, "dedup": map[string]int{"open": 0, "autoMerged": 0, "overridden": 0, "ignored": 0}},
		Performance:    map[string]interface{}{"avgFullFetchSec": nil, "avgIncrementalSec": nil},
		LastActions:    []map[string]interface{}{},
	}
	return prof, nil
}

func (r *Repository) CreateRequest(ctx context.Context, ar appadm.ActionRequest) (uuid.UUID, error) {
	js := toJSON(ar.Payload)
	return ar.ID, r.db.WithContext(ctx).Exec(`
      INSERT INTO admin_action_audit (id, tenant_id, cpf, action, status, requested_by, confirm_token, confirm_deadline, request_payload)
      VALUES (?,?,?,?, 'REQUESTED', ?, ?, ?, ?)
    `, ar.ID, ar.TenantID, ar.CPF, ar.Action, ar.RequestedBy, ar.ConfirmToken, ar.ConfirmDeadline, js).Error
}

func (r *Repository) ConfirmAndRun(ctx context.Context, id uuid.UUID, token string) error {
	// busca info da ação para validação e execução
	tx := r.db.WithContext(ctx).Begin()
	defer func() { _ = tx.Rollback() }()

	type row struct {
		CToken         string
		Deadline       *time.Time
		TenantID       string
		CPF            string
		Action         string
		Status         string
		RequestPayload *string
	}
	var rw row
	if err := tx.Raw(`SELECT tenant_id::text AS tenant_id, cpf, action, status, confirm_token, confirm_deadline, request_payload::text FROM admin_action_audit WHERE id = ?`, id).Scan(&rw).Error; err != nil {
		return err
	}

	// validações
	if rw.CToken == "" || rw.CToken != token {
		return fmt.Errorf("invalid confirm token")
	}
	if rw.Deadline != nil && time.Now().After(*rw.Deadline) {
		return fmt.Errorf("confirm token expired")
	}
	if rw.Status == "SUCCESS" {
		return nil // idempotente
	}

	// marca como CONFIRMED
	if err := tx.Exec(`UPDATE admin_action_audit SET status = 'CONFIRMED', updated_at = now() WHERE id = ?`, id).Error; err != nil {
		return err
	}

	// obtem lock para execução
	lockKey := fmt.Sprintf("%s:%s", rw.TenantID, rw.CPF)
	lock := r.getLock(lockKey)

	// executa em goroutine separada para não bloquear response
	go func() {
		lock.Lock()
		defer lock.Unlock()

		// marca como RUNNING
		_ = r.db.Exec(`UPDATE admin_action_audit SET status = 'RUNNING', updated_at = now() WHERE id = ?`, id)

		// executa ação real
		result, err := r.executeAction(context.Background(), rw.Action, rw.TenantID, rw.CPF, rw.RequestPayload)

		// atualiza resultado
		if err != nil {
			_ = r.db.Exec(`UPDATE admin_action_audit SET status = 'ERROR', error_message = ?, updated_at = now() WHERE id = ?`, err.Error(), id)
		} else {
			resultJson, _ := json.Marshal(result)
			_ = r.db.Exec(`UPDATE admin_action_audit SET status = 'SUCCESS', result_payload = ?, updated_at = now() WHERE id = ?`, string(resultJson), id)
		}
	}()

	return tx.Commit().Error
}

// getLock - obtem ou cria lock para chave específica (thread-safe)
func (r *Repository) getLock(key string) *sync.Mutex {
	r.locksMutex.Lock()
	defer r.locksMutex.Unlock()

	if lock, exists := r.executionLocks[key]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	r.executionLocks[key] = lock
	return lock
}

// executeAction - orquestra a execução real das ações conforme tipo
func (r *Repository) executeAction(ctx context.Context, action, tenantID, cpf string, requestPayloadJson *string) (map[string]interface{}, error) {
	// parseia payload de entrada
	var payload map[string]interface{}
	if requestPayloadJson != nil {
		_ = json.Unmarshal([]byte(*requestPayloadJson), &payload)
	}

	switch action {
	case "B3_FULL_FETCH":
		return r.executeB3FullFetch(ctx, tenantID, cpf, payload)
	case "B3_INCREMENTAL_FETCH":
		return r.executeB3Incremental(ctx, tenantID, cpf, payload)
	case "RECON_SCAN":
		return r.executeReconScan(ctx, tenantID, cpf, payload)
	case "AUTO_FIX":
		return r.executeAutoFix(ctx, tenantID, cpf, payload)
	case "DEDUPE_SCAN":
		return r.executeDedupScan(ctx, tenantID, cpf, payload)
	case "DEDUPE_RESOLVE":
		return r.executeDedupResolve(ctx, tenantID, payload)
	case "POLICY_UPDATE":
		return r.executePolicyUpdate(ctx, tenantID, cpf, payload)
	case "CLIENT_RESET":
		return r.executeClientReset(ctx, tenantID, cpf, payload)
	case "CLIENT_ZERO_AND_REFETCH":
		return r.executeZeroAndRefetch(ctx, tenantID, cpf, payload)
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}
}

// Implementações específicas das ações
func (r *Repository) executeB3FullFetch(ctx context.Context, tenantID, cpf string, payload map[string]interface{}) (map[string]interface{}, error) {
	fromMonth, _ := payload["from"].(string)
	toMonth, _ := payload["to"].(string)
	dryRun, _ := payload["dryRun"].(bool)

	if r.b3Orchestrator == nil {
		return map[string]interface{}{"result": "B3 orchestrator not available"}, nil
	}

	err := r.b3Orchestrator.ProcessFullHistorical(ctx, tenantID, cpf, fromMonth, toMonth, dryRun)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result":    "SUCCESS",
		"fromMonth": fromMonth,
		"toMonth":   toMonth,
		"dryRun":    dryRun,
	}, nil
}

func (r *Repository) executeB3Incremental(ctx context.Context, tenantID, cpf string, payload map[string]interface{}) (map[string]interface{}, error) {
	date, _ := payload["date"].(string)
	dryRun, _ := payload["dryRun"].(bool)

	if r.b3Orchestrator == nil {
		return map[string]interface{}{"result": "B3 orchestrator not available"}, nil
	}

	err := r.b3Orchestrator.ProcessIncremental(ctx, tenantID, cpf, date, dryRun)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result": "SUCCESS",
		"date":   date,
		"dryRun": dryRun,
	}, nil
}

func (r *Repository) executeReconScan(ctx context.Context, tenantID, cpf string, payload map[string]interface{}) (map[string]interface{}, error) {
	typesInterface, _ := payload["types"].([]interface{})
	types := make([]string, len(typesInterface))
	for i, t := range typesInterface {
		types[i], _ = t.(string)
	}
	dryRun, _ := payload["dryRun"].(bool)

	if r.reconOrchestrator == nil {
		return map[string]interface{}{"result": "Reconciliation orchestrator not available"}, nil
	}

	err := r.reconOrchestrator.ScanInconsistencies(ctx, tenantID, cpf, types, dryRun)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result": "SUCCESS",
		"types":  types,
		"dryRun": dryRun,
	}, nil
}

func (r *Repository) executeAutoFix(ctx context.Context, tenantID, cpf string, payload map[string]interface{}) (map[string]interface{}, error) {
	typesInterface, _ := payload["types"].([]interface{})
	types := make([]string, len(typesInterface))
	for i, t := range typesInterface {
		types[i], _ = t.(string)
	}
	dryRun, _ := payload["dryRun"].(bool)

	if r.reconOrchestrator == nil {
		return map[string]interface{}{"result": "Reconciliation orchestrator not available"}, nil
	}

	err := r.reconOrchestrator.AutoFix(ctx, tenantID, cpf, types, dryRun)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result": "SUCCESS",
		"types":  types,
		"dryRun": dryRun,
	}, nil
}

func (r *Repository) executeDedupScan(ctx context.Context, tenantID, cpf string, payload map[string]interface{}) (map[string]interface{}, error) {
	scanMode, _ := payload["scanMode"].(string)
	dryRun, _ := payload["dryRun"].(bool)

	if r.dedupOrchestrator == nil {
		return map[string]interface{}{"result": "Dedup orchestrator not available"}, nil
	}

	err := r.dedupOrchestrator.ScanDuplicates(ctx, tenantID, cpf, scanMode, dryRun)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result":   "SUCCESS",
		"scanMode": scanMode,
		"dryRun":   dryRun,
	}, nil
}

func (r *Repository) executeDedupResolve(ctx context.Context, tenantID string, payload map[string]interface{}) (map[string]interface{}, error) {
	action, _ := payload["action"].(string)
	candidatesInterface, _ := payload["candidateIds"].([]interface{})
	candidates := make([]string, len(candidatesInterface))
	for i, c := range candidatesInterface {
		candidates[i], _ = c.(string)
	}

	if r.dedupOrchestrator == nil {
		return map[string]interface{}{"result": "Dedup orchestrator not available"}, nil
	}

	err := r.dedupOrchestrator.ResolveDuplicates(ctx, tenantID, action, candidates)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result":       "SUCCESS",
		"action":       action,
		"candidateIds": candidates,
	}, nil
}

func (r *Repository) executePolicyUpdate(ctx context.Context, tenantID, cpf string, payload map[string]interface{}) (map[string]interface{}, error) {
	mode, _ := payload["mode"].(string)
	reason, _ := payload["reason"].(string)

	if r.policyOrchestrator == nil {
		return map[string]interface{}{"result": "Policy orchestrator not available"}, nil
	}

	err := r.policyOrchestrator.SetPolicy(ctx, tenantID, cpf, mode, reason)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result": "SUCCESS",
		"mode":   mode,
		"reason": reason,
	}, nil
}

func (r *Repository) executeClientReset(ctx context.Context, tenantID, cpf string, payload map[string]interface{}) (map[string]interface{}, error) {
	what, _ := payload["what"].(string)
	dryRun, _ := payload["dryRun"].(bool)

	if r.clientDataOrchestrator == nil {
		return map[string]interface{}{"result": "Client data orchestrator not available"}, nil
	}

	err := r.clientDataOrchestrator.ResetClient(ctx, tenantID, cpf, what, dryRun)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result": "SUCCESS",
		"what":   what,
		"dryRun": dryRun,
	}, nil
}

func (r *Repository) executeZeroAndRefetch(ctx context.Context, tenantID, cpf string, payload map[string]interface{}) (map[string]interface{}, error) {
	fromDate, _ := payload["from"].(string)
	dryRun, _ := payload["dryRun"].(bool)

	if r.clientDataOrchestrator == nil {
		return map[string]interface{}{"result": "Client data orchestrator not available"}, nil
	}

	err := r.clientDataOrchestrator.ZeroAndRefetch(ctx, tenantID, cpf, fromDate, dryRun)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result":   "SUCCESS",
		"fromDate": fromDate,
		"dryRun":   dryRun,
	}, nil
}

func maskCPF(cpf string) string {
	if len(cpf) < 4 {
		return "***"
	}
	return "***" + cpf[len(cpf)-4:]
}
func toJSON(m map[string]interface{}) string { b, _ := json.Marshal(m); return string(b) }

// Search: retorna CPFs mascarados por prefixo
func (r *Repository) Search(ctx context.Context, tenantID, query string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	type row struct{ CPF string }
	var rows []row
	qb := r.db.WithContext(ctx).Table("b3_operations_ledger").Select("DISTINCT cpf").Where("tenant_id = ?", tenantID)
	if query != "" {
		qb = qb.Where("cpf LIKE ?", query+"%")
	}
	if err := qb.Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, rw := range rows {
		out = append(out, map[string]interface{}{"cpfMasked": maskCPF(rw.CPF)})
	}
	return out, nil
}

// ListActions: lista auditoria com filtros
func (r *Repository) ListActions(ctx context.Context, tenantID, cpf, action, status string, page, pageSize int) ([]map[string]interface{}, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	type row struct {
		ID                          uuid.UUID
		Action, Status, RequestedBy string
		CreatedAt, UpdatedAt        time.Time
	}
	var rows []row
	qb := r.db.WithContext(ctx).Table("admin_action_audit").Select("id, action, status, requested_by, created_at, updated_at").Where("tenant_id = ?", tenantID)
	if cpf != "" {
		qb = qb.Where("cpf = ?", cpf)
	}
	if action != "" {
		qb = qb.Where("action = ?", action)
	}
	if status != "" {
		qb = qb.Where("status = ?", status)
	}
	if err := qb.Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, rw := range rows {
		out = append(out, map[string]interface{}{"id": rw.ID, "action": rw.Action, "status": rw.Status, "requestedBy": rw.RequestedBy, "createdAt": rw.CreatedAt, "updatedAt": rw.UpdatedAt})
	}
	return out, nil
}

func (r *Repository) GetAction(ctx context.Context, tenantID string, id uuid.UUID) (map[string]interface{}, error) {
	type row struct {
		ID                          uuid.UUID
		Action, Status, RequestedBy string
		ConfirmedBy                 *string
		RequestPayload              *string
		ResultPayload               *string
		ErrorMessage                *string
		CreatedAt, UpdatedAt        time.Time
	}
	var rw row
	if err := r.db.WithContext(ctx).Raw(`SELECT id, action, status, requested_by, confirmed_by, request_payload::text, result_payload::text, error_message, created_at, updated_at FROM admin_action_audit WHERE tenant_id = ? AND id = ?`, tenantID, id).Scan(&rw).Error; err != nil {
		return nil, err
	}
	out := map[string]interface{}{"id": rw.ID, "action": rw.Action, "status": rw.Status, "requestedBy": rw.RequestedBy, "confirmedBy": rw.ConfirmedBy, "createdAt": rw.CreatedAt, "updatedAt": rw.UpdatedAt}
	if rw.RequestPayload != nil {
		out["requestPayload"] = *rw.RequestPayload
	}
	if rw.ResultPayload != nil {
		out["resultPayload"] = *rw.ResultPayload
	}
	if rw.ErrorMessage != nil {
		out["errorMessage"] = *rw.ErrorMessage
	}
	return out, nil
}

// ExportLedger: leitura simples respeitando policy (excluir B3 quando necessário)
func (r *Repository) ExportLedger(ctx context.Context, tenantID, cpf string, excludeB3 bool, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 50000
	}
	type row struct {
		ID, Ticker, AssetType, OperationType, Source, Currency, ReasonCode, PriceConfidence string
		OperationDate                                                                       string
		Quantity                                                                            float64
		UnitPrice                                                                           *float64
	}
	var rows []row
	qb := r.db.WithContext(ctx).Table("b3_operations_ledger").Select("id, ticker, asset_type, operation_date, operation_type, source, quantity, unit_price, currency, reason_code, price_confidence").Where("tenant_id = ? AND cpf = ? AND is_active = true", tenantID, cpf)
	if excludeB3 {
		qb = qb.Where("source <> 'B3_RAW'")
	}
	if err := qb.Order("operation_date DESC, id DESC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, rw := range rows {
		out = append(out, map[string]interface{}{
			"id": rw.ID, "ticker": rw.Ticker, "assetType": rw.AssetType, "operationDate": rw.OperationDate, "operationType": rw.OperationType, "source": rw.Source, "quantity": rw.Quantity, "unitPrice": rw.UnitPrice, "currency": rw.Currency, "reasonCode": rw.ReasonCode, "priceConfidence": rw.PriceConfidence,
		})
	}
	return out, nil
}

// ExportLedgerStream: leitura em lotes para exports grandes com streaming
func (r *Repository) ExportLedgerStream(ctx context.Context, tenantID, cpf string, excludeB3 bool, limit int, callback func([]map[string]interface{}) error) error {
	if limit <= 0 {
		limit = 50000
	}

	const batchSize = 1000 // processar em lotes de 1000 registros
	offset := 0
	totalProcessed := 0

	for totalProcessed < limit {
		// calcula tamanho do lote atual
		currentBatchSize := batchSize
		if totalProcessed+batchSize > limit {
			currentBatchSize = limit - totalProcessed
		}

		type row struct {
			ID, Ticker, AssetType, OperationType, Source, Currency, ReasonCode, PriceConfidence string
			OperationDate                                                                       string
			Quantity                                                                            float64
			UnitPrice                                                                           *float64
		}
		var rows []row

		qb := r.db.WithContext(ctx).Table("b3_operations_ledger").
			Select("id, ticker, asset_type, operation_date, operation_type, source, quantity, unit_price, currency, reason_code, price_confidence").
			Where("tenant_id = ? AND cpf = ? AND is_active = true", tenantID, cpf)

		if excludeB3 {
			qb = qb.Where("source <> 'B3_RAW'")
		}

		if err := qb.Order("operation_date DESC, id DESC").
			Limit(currentBatchSize).
			Offset(offset).
			Scan(&rows).Error; err != nil {
			return err
		}

		// se não há mais registros, para
		if len(rows) == 0 {
			break
		}

		// converte para formato de saída
		out := make([]map[string]interface{}, 0, len(rows))
		for _, rw := range rows {
			out = append(out, map[string]interface{}{
				"id": rw.ID, "ticker": rw.Ticker, "assetType": rw.AssetType, "operationDate": rw.OperationDate,
				"operationType": rw.OperationType, "source": rw.Source, "quantity": rw.Quantity, "unitPrice": rw.UnitPrice,
				"currency": rw.Currency, "reasonCode": rw.ReasonCode, "priceConfidence": rw.PriceConfidence,
			})
		}

		// chama callback com o lote
		if err := callback(out); err != nil {
			return err
		}

		offset += len(rows)
		totalProcessed += len(rows)

		// se retornou menos registros que esperado, não há mais dados
		if len(rows) < currentBatchSize {
			break
		}

		// verifica se contexto foi cancelado
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}
