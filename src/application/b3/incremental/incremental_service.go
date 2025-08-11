package incremental

import (
	"context"
	"time"

	"github.com/google/uuid"

	appingest "suno-wallets/src/application/b3/ingest"
	appnorm "suno-wallets/src/application/b3/normalize"
	polsvc "suno-wallets/src/application/clientpolicy"
	syncrepo "suno-wallets/src/infrastructure/b3/sync"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/helpers"
)

// Comentários em pt-BR: Serviço de execução incremental sob demanda (1.14)

// Ports de integração (para facilitar testes)
type IngestPort interface {
	Ingest(ctx context.Context, p appingest.IngestParams) (*appingest.Summary, error)
}
type NormalizePort interface {
	Run(ctx context.Context, p appnorm.RunParams) (*appnorm.Summary, error)
}

type SyncRepository interface {
	GetByTenantCPF(ctx context.Context, tenantID string, cpf string) (*syncrepo.SyncState, error)
	MarkResult(ctx context.Context, id uuid.UUID, success bool, needsReprocess bool, lastResult string, lastError *string, txSyncAt, posSyncAt *time.Time) error
}

type Service struct {
	repo      SyncRepository
	ingest    IngestPort
	normalize NormalizePort
	policy    *polsvc.Service
}

func NewService(repo SyncRepository, ingest IngestPort, normalize NormalizePort) *Service {
	return &Service{repo: repo, ingest: ingest, normalize: normalize}
}

func (s *Service) WithPolicy(p *polsvc.Service) *Service { s.policy = p; return s }

type Params struct {
	TenantID    string
	CPF         string
	DataTypes   []string // transactions | positions
	AssetTypes  []string // e.g., ["equity"]
	Since       *time.Time
	End         *time.Time
	Force       bool
	DryRun      bool
	Concurrency int // não usado nesta fase
}

type TypeSummary struct {
	Raw        appingest.Summary `json:"raw"`
	Normalized appnorm.Summary   `json:"normalized"`
	From       string            `json:"from"`
	To         string            `json:"to"`
}

type Summary struct {
	Transactions *TypeSummary `json:"transactions,omitempty"`
	Positions    *TypeSummary `json:"positions,omitempty"`
	StartedAt    time.Time    `json:"startedAt"`
	FinishedAt   time.Time    `json:"finishedAt"`
	DurationMs   int64        `json:"durationMs"`
	DryRun       bool         `json:"dryRun"`
	Force        bool         `json:"force"`
}

func (s *Service) Run(ctx context.Context, p Params) (*Summary, error) {
	started := time.Now()
	out := &Summary{StartedAt: started, DryRun: p.DryRun, Force: p.Force}
	// Enforce policy: MANUAL_ONLY → skipar jobs B3
	if s.policy != nil {
		mode := s.policy.GetMode(ctx, p.TenantID, p.CPF)
		if string(mode) == "MANUAL_ONLY" {
			observability.IncPolicySkipped("B3_Incremental")
			helpers.LogInfo("incremental skipped by client policy", map[string]any{"tenantId": p.TenantID, "cpfMasked": maskCPF(p.CPF), "mode": "MANUAL_ONLY"})
			out.FinishedAt = time.Now()
			out.DurationMs = time.Since(started).Milliseconds()
			return out, nil
		}
	}

	st, _ := s.repo.GetByTenantCPF(ctx, p.TenantID, p.CPF)
	now := time.Now().UTC()
	to := now
	if p.End != nil {
		to = *p.End
	}

	needsReprocess := false
	success := true
	var lastErr *string
	var txSyncAt, posSyncAt *time.Time

	logBase := map[string]any{"tenantId": p.TenantID, "cpfMasked": maskCPF(p.CPF), "types": p.DataTypes, "force": p.Force, "dryRun": p.DryRun}
	for _, dt := range p.DataTypes {
		assetType := firstOrDefault(p.AssetTypes, "equity")
		var from time.Time
		switch dt {
		case "transactions":
			if p.Since != nil {
				from = *p.Since
			} else {
				from = nextOrEpoch(st, true)
			}
		case "positions":
			if p.Since != nil {
				from = *p.Since
			} else {
				from = nextOrEpoch(st, false)
			}
		default:
			continue
		}
		// Normalizar bordas
		from = dayStart(from)
		to = dayStart(to)
		if !from.Before(to) {
			// nada a fazer
			s.attachType(out, dt, &TypeSummary{From: from.Format("2006-01-02"), To: to.Format("2006-01-02")})
			helpers.LogInfo("incremental no-op", merge(logBase, map[string]any{"type": dt, "from": from.Format("2006-01-02"), "to": to.Format("2006-01-02")}))
			continue
		}

		months := monthWindows(from, to)
		// dry run: estimativa simples de páginas = meses
		if p.DryRun {
			ts := &TypeSummary{From: from.Format("2006-01-02"), To: to.Format("2006-01-02")}
			ts.Raw.MonthsProcessed = len(months)
			ts.Raw.PagesProcessed = len(months)
			s.attachType(out, dt, ts)
			helpers.LogInfo("incremental dryrun plan", merge(logBase, map[string]any{"type": dt, "from": ts.From, "to": ts.To, "months": ts.Raw.MonthsProcessed, "pages": ts.Raw.PagesProcessed}))
			continue
		}

		// execução real: chamar ingest somente para meses faltantes (ou tudo se force)
		rawSaved := 0
		for _, w := range months {
			res, err := s.ingest.Ingest(ctx, appingest.IngestParams{
				TenantID:  p.TenantID,
				CPF:       p.CPF,
				DataType:  dt,
				AssetType: assetType,
				Start:     w[0].Format("2006-01-02"),
				End:       w[1].Format("2006-01-02"),
				FetchAll:  true,
				Force:     p.Force,
				DryRun:    false,
			})
			if err != nil {
				success = false
				e := err.Error()
				lastErr = &e
				observability.IncIncrementalError("ingest")
				helpers.LogError("incremental ingest failed", err, merge(logBase, map[string]any{"type": dt}))
				break
			}
			if res != nil {
				rawSaved += res.Saved
			}
		}
		ts := &TypeSummary{From: from.Format("2006-01-02"), To: to.Format("2006-01-02")}
		if rawSaved > 0 {
			needsReprocess = true
		}

		// normalizar somente se houver novidade
		if rawSaved > 0 {
			nsum, err := s.normalize.Run(ctx, appnorm.RunParams{
				TenantID:  p.TenantID,
				CPF:       p.CPF,
				DataType:  dt,
				AssetType: assetType,
				Start:     from,
				End:       to,
				Force:     p.Force,
				DryRun:    false,
			})
			if err != nil {
				success = false
				e := err.Error()
				lastErr = &e
				observability.IncIncrementalError("normalize")
				helpers.LogError("incremental normalize failed", err, merge(logBase, map[string]any{"type": dt}))
			} else if nsum != nil {
				ts.Normalized = *nsum
			}
		}
		s.attachType(out, dt, ts)
		// avançar marcos em sucesso
		if success {
			switch dt {
			case "transactions":
				txSyncAt = &to
			case "positions":
				posSyncAt = &to
			}
			helpers.LogInfo("incremental type finished", merge(logBase, map[string]any{"type": dt, "from": ts.From, "to": ts.To, "saved": rawSaved}))
		}
	}

	// marcar resultado no sync_state
	if st != nil {
		_ = s.repo.MarkResult(ctx, st.ID, success, needsReprocess, tern(success, "OK", "ERROR"), lastErr, txSyncAt, posSyncAt)
	}
	out.FinishedAt = time.Now()
	out.DurationMs = time.Since(started).Milliseconds()
	observability.ObserveIncrementalRun(tern(success, "success", "error"), joinedTypes(p.DataTypes), started)
	helpers.LogInfo("incremental finished", merge(logBase, map[string]any{"durationMs": out.DurationMs, "result": tern(success, "success", "error")}))
	return out, nil
}

func (s *Service) attachType(out *Summary, dt string, ts *TypeSummary) {
	switch dt {
	case "transactions":
		out.Transactions = ts
	case "positions":
		out.Positions = ts
	}
}

func monthWindows(start, end time.Time) [][2]time.Time {
	var out [][2]time.Time
	cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cur.After(last) {
		next := cur.AddDate(0, 1, 0).Add(-24 * time.Hour)
		if next.After(end) {
			next = end
		}
		winStart := cur
		if winStart.Before(start) {
			winStart = start
		}
		out = append(out, [2]time.Time{winStart, next})
		cur = cur.AddDate(0, 1, 0)
	}
	return out
}

func firstOrDefault(list []string, def string) string {
	if len(list) == 0 || list[0] == "" {
		return def
	}
	return list[0]
}
func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
func nextOrEpoch(st *syncrepo.SyncState, isTx bool) time.Time {
	if st == nil {
		// Data de início da API B3: 01 de novembro de 2019
		return time.Date(2019, 11, 1, 0, 0, 0, 0, time.UTC)
	}
	var t *time.Time
	if isTx {
		t = st.LastTxSyncAt
	} else {
		t = st.LastPosSyncAt
	}
	if t == nil {
		return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return t.Add(24 * time.Hour)
}
func tern[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
func joinedTypes(list []string) string {
	if len(list) == 0 {
		return "none"
	}
	if len(list) == 1 {
		return list[0]
	}
	return list[0]
}

// helpers de log locais
func maskCPF(cpf string) string {
	if len(cpf) != 11 {
		return "invalid"
	}
	return "*********" + cpf[9:]
}
func merge(base map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
