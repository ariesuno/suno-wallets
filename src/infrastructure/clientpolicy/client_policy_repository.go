package clientpolicy

import (
	dom "suno-wallets/src/domain/clientpolicy"
	"time"

	"gorm.io/gorm"
)

// Comentários em pt-BR: repositório SQL para políticas por cliente + audit

type RepositoryImpl struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *RepositoryImpl { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) Get(tenantID, cpf string) (*dom.ClientPolicy, error) {
	type row struct {
		Mode      string
		Reason    *string
		UpdatedAt *time.Time
	}
	var rw row
	if err := r.db.Raw(`SELECT mode, reason, updated_at FROM client_data_source_policy WHERE tenant_id = ? AND cpf = ? LIMIT 1`, tenantID, cpf).Scan(&rw).Error; err != nil {
		return nil, err
	}
	if rw.Mode == "" {
		return nil, nil
	}
	pol := &dom.ClientPolicy{TenantID: tenantID, CPF: cpf, Mode: dom.Mode(rw.Mode)}
	if rw.Reason != nil {
		pol.Reason = *rw.Reason
	}
	if rw.UpdatedAt != nil {
		pol.UpdatedAt = *rw.UpdatedAt
	}
	return pol, nil
}

func (r *RepositoryImpl) Upsert(tenantID, cpf string, mode dom.Mode, reason, actor string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
          INSERT INTO client_data_source_policy (tenant_id, cpf, mode, reason, created_by, updated_by)
          VALUES (?,?,?,?,?,?)
          ON CONFLICT (tenant_id, cpf) DO UPDATE SET mode = EXCLUDED.mode, reason = EXCLUDED.reason, updated_at = now(), updated_by = EXCLUDED.updated_by
        `, tenantID, cpf, string(mode), reason, actor, actor).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
          INSERT INTO client_data_source_policy_audit (tenant_id, cpf, old_mode, new_mode, changed_by, reason)
          VALUES (?, ?, (SELECT mode FROM client_data_source_policy WHERE tenant_id = ? AND cpf = ?), ?, ?, ?)
        `, tenantID, cpf, tenantID, cpf, string(mode), actor, reason).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *RepositoryImpl) ListAudit(tenantID, cpf string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 50
	}
	type row struct {
		CreatedAt time.Time
		OldMode   *string
		NewMode   string
		ChangedBy *string
		Reason    *string
	}
	var rows []row
	if err := r.db.Raw(`SELECT created_at, old_mode, new_mode, changed_by, reason FROM client_data_source_policy_audit WHERE tenant_id = ? AND cpf = ? ORDER BY created_at DESC LIMIT ?`, tenantID, cpf, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, rw := range rows {
		m := map[string]interface{}{"createdAt": rw.CreatedAt, "newMode": rw.NewMode}
		if rw.OldMode != nil {
			m["oldMode"] = *rw.OldMode
		}
		if rw.ChangedBy != nil {
			m["changedBy"] = *rw.ChangedBy
		}
		if rw.Reason != nil {
			m["reason"] = *rw.Reason
		}
		out = append(out, m)
	}
	return out, nil
}
