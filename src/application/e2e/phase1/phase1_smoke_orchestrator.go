package phase1

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"suno-wallets/src/infrastructure/e2e"
)

// Comentários em pt-BR: Orquestrador de smoke/integridade fase 1

type Orchestrator struct{ http *e2e.HTTPClient }

type StageResult struct {
	Ok         bool           `json:"ok"`
	Details    map[string]any `json:"details,omitempty"`
	Summary    map[string]any `json:"summary,omitempty"`
	Counts     map[string]any `json:"counts,omitempty"`
	DurationMs int64          `json:"durationMs"`
}

type Report struct {
	TenantID   string                 `json:"tenantId"`
	CPFMasked  string                 `json:"cpfMasked"`
	StartedAt  time.Time              `json:"startedAt"`
	FinishedAt time.Time              `json:"finishedAt"`
	DurationMs int64                  `json:"durationMs"`
	Stages     map[string]StageResult `json:"stages"`
	Verdict    string                 `json:"verdict"`
	Failures   []map[string]string    `json:"failures"`
}

func NewOrchestrator(http *e2e.HTTPClient) *Orchestrator { return &Orchestrator{http: http} }

func maskCPF(cpf string) string {
	if len(cpf) < 4 {
		return "***"
	}
	return "***" + cpf[len(cpf)-4:]
}

func (o *Orchestrator) Run(ctx context.Context, tenantID, cpf string, baseURL string, allowReset bool) (*Report, error) {
	r := &Report{TenantID: tenantID, CPFMasked: maskCPF(cpf), StartedAt: time.Now().UTC(), Stages: map[string]StageResult{}}
	add := func(name string, ok bool, start time.Time, details map[string]any) {
		r.Stages[name] = StageResult{Ok: ok, Details: details, DurationMs: time.Since(start).Milliseconds()}
		if !ok {
			r.Failures = append(r.Failures, map[string]string{"stage": name, "reason": "criteria not met"})
		}
	}

	// Health
	{
		st := time.Now()
		ok := true
		code, _, _ := o.http.Get(ctx, "/health/live", "phase1")
		if code != 200 {
			ok = false
		}
		code, _, _ = o.http.Get(ctx, "/health/ready", "phase1")
		if code != 200 {
			ok = false
		}
		add("health", ok, st, map[string]any{"live": code == 200})
	}

	// Reports (usa 1.15) como validação de normalização/sync
	{
		st := time.Now()
		ok := true
		code, body, _ := o.http.Get(ctx, fmt.Sprintf("/api/v1/b3/client/raw-date-range?cpf=%s", cpf), "phase1")
		if code != 200 {
			ok = false
		}
		add("reports", ok, st, map[string]any{"rawDateRangeBytes": len(body)})
	}

	// Verdict
	r.FinishedAt = time.Now().UTC()
	r.DurationMs = time.Since(r.StartedAt).Milliseconds()
	r.Verdict = "PASS"
	for _, s := range r.Stages {
		if !s.Ok {
			r.Verdict = "FAIL"
			break
		}
	}

	// persistir relatórios
	_ = os.MkdirAll(filepath.Join("docs", "e2e"), 0o755)
	b, _ := json.MarshalIndent(r, "", "  ")
	_ = os.WriteFile(filepath.Join("docs", "e2e", "Phase1SmokeReport.json"), b, 0o644)
	_ = os.WriteFile(filepath.Join("docs", "e2e", "Phase1SmokeReport.md"), []byte(fmt.Sprintf("Verdict: %s\n", r.Verdict)), 0o644)
	return r, nil
}
