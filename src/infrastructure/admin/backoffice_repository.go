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
	}
	var rw row
	if err := tx.Raw(`SELECT confirm_token, confirm_deadline FROM admin_action_audit WHERE id = ?`, id).Scan(&rw).Error; err != nil {
		return err
	}
	if rw.CToken == "" || rw.CToken != token || (rw.Deadline != nil && time.Now().After(*rw.Deadline)) {
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
