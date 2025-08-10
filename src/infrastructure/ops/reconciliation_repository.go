package ops

import (
	"context"
	appops "suno-wallets/src/application/ops"
	"time"

	"gorm.io/gorm"
)

// Comentários em pt-BR: repositório que consolida contagens e últimos marcos para o resumo

type ReconRepo struct{ db *gorm.DB }

func NewReconRepo(db *gorm.DB) *ReconRepo { return &ReconRepo{db: db} }

func (r *ReconRepo) LoadSummary(ctx context.Context, tenantID string, cpf string) (*appops.ReconSummary, error) {
	// períodos: placeholder por agora, poderia vir de janelas persistidas
	period := map[string]string{"from": "2010-01-01", "to": time.Now().Format("2006-01-02")}
	// modo de dados
	var mode string
	_ = r.db.WithContext(ctx).Raw(`SELECT COALESCE(mode,'HYBRID') FROM dedup_policies WHERE tenant_id = ? AND cpf = ? LIMIT 1`, tenantID, cpf).Scan(&mode).Error

	// inconsistências
	type c1 struct {
		Open       int
		Resolved   int
		LastScanAt *time.Time
	}
	var cc c1
	_ = r.db.WithContext(ctx).Raw(`
      SELECT
        SUM(CASE WHEN status = 'OPEN' THEN 1 ELSE 0 END) AS open,
        SUM(CASE WHEN status IN ('RESOLVED','OVERRIDDEN') THEN 1 ELSE 0 END) AS resolved,
        MAX(updated_at) AS last_scan_at
      FROM b3_inconsistencies WHERE tenant_id = ? AND cpf = ?
    `, tenantID, cpf).Scan(&cc).Error
	// byType
	type bt struct {
		Type string
		Cnt  int
	}
	var btRows []bt
	_ = r.db.WithContext(ctx).Raw(`
      SELECT type, COUNT(*) AS cnt FROM b3_inconsistencies WHERE tenant_id = ? AND cpf = ? AND status = 'OPEN' GROUP BY type
    `, tenantID, cpf).Scan(&btRows).Error
	byType := map[string]int{}
	for _, rw := range btRows {
		byType[rw.Type] = rw.Cnt
	}

	// ops de sistema
	type c2 struct {
		Created       int
		LastAutoFixAt *time.Time
	}
	var cs c2
	_ = r.db.WithContext(ctx).Raw(`
      SELECT COUNT(*) AS created, MAX(updated_at) AS last_auto_fix_at
      FROM b3_operations_ledger WHERE tenant_id = ? AND cpf = ? AND source = 'SYSTEM_SYNTHETIC'
    `, tenantID, cpf).Scan(&cs).Error

	// overrides (USER_MANUAL + dedup decisions)
	type c3 struct {
		Manual       int
		LastActionAt *time.Time
	}
	var co c3
	_ = r.db.WithContext(ctx).Raw(`
      SELECT COUNT(*) AS manual, MAX(updated_at) AS last_action_at
      FROM b3_operations_ledger WHERE tenant_id = ? AND cpf = ? AND source = 'USER_MANUAL'
    `, tenantID, cpf).Scan(&co).Error

	// misto por ticker
	type tr struct {
		Ticker    string
		AssetType string
		B3        int
		Sys       int
		Man       int
		LastOpAt  *time.Time
		OpenInc   int
	}
	var trows []tr
	_ = r.db.WithContext(ctx).Raw(`
      WITH ops AS (
        SELECT ticker, asset_type,
          SUM(CASE WHEN source='B3_RAW' THEN 1 ELSE 0 END) AS b3,
          SUM(CASE WHEN source='SYSTEM_SYNTHETIC' THEN 1 ELSE 0 END) AS sys,
          SUM(CASE WHEN source='USER_MANUAL' THEN 1 ELSE 0 END) AS man,
          MAX(operation_date) AS last_op_at
        FROM b3_operations_ledger
        WHERE tenant_id = ? AND cpf = ? AND is_active = true
        GROUP BY ticker, asset_type
      ), inc AS (
        SELECT ticker, COUNT(*) AS open_inc FROM b3_inconsistencies WHERE tenant_id = ? AND cpf = ? AND status='OPEN' GROUP BY ticker
      )
      SELECT o.ticker, o.asset_type, o.b3, o.sys, o.man, o.last_op_at, COALESCE(i.open_inc,0) AS open_inc
      FROM ops o LEFT JOIN inc i ON o.ticker = i.ticker
    `, tenantID, cpf, tenantID, cpf).Scan(&trows).Error

	tickers := make([]map[string]interface{}, 0, len(trows))
	for _, rw := range trows {
		tickers = append(tickers, map[string]interface{}{
			"ticker":              rw.Ticker,
			"assetType":           rw.AssetType,
			"operations":          map[string]int{"B3_RAW": rw.B3, "SYSTEM_SYNTHETIC": rw.Sys, "USER_MANUAL": rw.Man},
			"inconsistenciesOpen": rw.OpenInc,
			"lastOperationAt":     rw.LastOpAt,
		})
	}

	out := &appops.ReconSummary{
		CPFMasked:  maskCPFLocal(cpf),
		Period:     period,
		DataSource: mode,
		B3:         map[string]interface{}{"lastFullFetchAt": lastFullFetchAt(r, ctx, tenantID, cpf), "lastIncrementalAt": lastIncrementalAt(r, ctx, tenantID, cpf), "latestWindow": latestWindow(r, ctx, tenantID, cpf, period)},
		Incons:     map[string]interface{}{"open": map[string]interface{}{"total": cc.Open, "byType": byType}, "resolved": cc.Resolved, "lastScanAt": cc.LastScanAt},
		SystemOps:  map[string]interface{}{"created": cs.Created, "byReason": map[string]int{}, "lastAutoFixAt": cs.LastAutoFixAt},
		Overrides:  map[string]interface{}{"manualOps": co.Manual, "dedup": map[string]int{"autoMerged": 0, "overridden": 0, "ignored": 0}, "lastActionAt": co.LastActionAt},
		Tickers:    tickers,
		Notes:      []string{},
	}
	return out, nil
}

// últimos marcos vindos do estado de sync/checkpoints (fallback simples)
func lastFullFetchAt(r *ReconRepo, ctx context.Context, tenantID string, cpf string) interface{} {
	var ts *time.Time
	_ = r.db.WithContext(ctx).Raw(`SELECT MAX(last_pos_sync_at) FROM b3_sync_state WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Scan(&ts).Error
	return ts
}
func lastIncrementalAt(r *ReconRepo, ctx context.Context, tenantID string, cpf string) interface{} {
	var ts *time.Time
	_ = r.db.WithContext(ctx).Raw(`SELECT MAX(last_tx_sync_at) FROM b3_sync_state WHERE tenant_id = ? AND cpf = ?`, tenantID, cpf).Scan(&ts).Error
	return ts
}
func latestWindow(r *ReconRepo, ctx context.Context, tenantID string, cpf string, period map[string]string) map[string]string {
	// placeholder: devolve período padrão; pode ser refinado via tabela de janelas
	return map[string]string{"from": period["from"], "to": period["to"]}
}

func maskCPFLocal(cpf string) string {
	if len(cpf) < 4 {
		return "***"
	}
	return "***" + cpf[len(cpf)-4:]
}
