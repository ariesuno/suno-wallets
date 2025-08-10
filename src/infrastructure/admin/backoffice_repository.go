package admin

import (
	"context"
	"encoding/json"
	appadm "suno-wallets/src/application/admin"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Comentários em pt-BR: repositório para leitura agregada e persistência de auditoria/admin actions

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

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
	// marca CONFIRMED e RUNNING, e finaliza como SUCCESS (placeholder)
	tx := r.db.WithContext(ctx).Begin()
	defer func() { _ = tx.Rollback() }()
	type row struct {
		CToken   string
		Deadline *time.Time
		TenantID string
		CPF      string
		Status   string
	}
	var rw row
	if err := tx.Raw(`SELECT tenant_id::text AS tenant_id, cpf, status, confirm_token, confirm_deadline FROM admin_action_audit WHERE id = ?`, id).Scan(&rw).Error; err != nil {
		return err
	}
	if rw.CToken == "" || rw.CToken != token || (rw.Deadline != nil && time.Now().After(*rw.Deadline)) {
		return nil
	}
	if rw.Status == "SUCCESS" {
		return nil
	}
	if err := tx.Exec(`UPDATE admin_action_audit SET status = 'CONFIRMED', updated_at = now() WHERE id = ?`, id).Error; err != nil {
		return err
	}
	// executar ação real aqui (orquestrar serviços)…
	res := map[string]interface{}{"result": "OK"}
	js, _ := json.Marshal(res)
	if err := tx.Exec(`UPDATE admin_action_audit SET status = 'SUCCESS', result_payload = ?, updated_at = now() WHERE id = ?`, string(js), id).Error; err != nil {
		return err
	}
	return tx.Commit().Error
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
