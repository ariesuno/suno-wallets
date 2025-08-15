package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// JobStatus representa o status de um job com máquina de estados explícita
type JobStatus string

const (
	// Estados primários
	JobStatusWaiting       JobStatus = "waiting"        // Aguardando condições (ex: locks, dependencies)
	JobStatusQueued        JobStatus = "queued"         // Na fila, pronto para processamento
	JobStatusProcessing    JobStatus = "processing"     // Sendo processado por um worker
	JobStatusPartial       JobStatus = "partial"        // Processamento parcial (algumas etapas concluídas)
	JobStatusCompleted     JobStatus = "completed"      // Concluído com sucesso
	JobStatusFailed        JobStatus = "failed"         // Falhou permanentemente
	JobStatusCancelled     JobStatus = "cancelled"      // Cancelado pelo usuário
	JobStatusStalled       JobStatus = "stalled"        // Worker morreu, precisa ser reprocessado
	JobStatusNeedReprocess JobStatus = "need_reprocess" // Precisa ser reprocessado (ex: dados novos)
)

// IsTerminalState verifica se o estado é terminal (não pode mudar)
func (s JobStatus) IsTerminalState() bool {
	switch s {
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return true
	default:
		return false
	}
}

// IsActiveState verifica se o job está em processamento ativo
func (s JobStatus) IsActiveState() bool {
	switch s {
	case JobStatusProcessing, JobStatusPartial:
		return true
	default:
		return false
	}
}

// CanTransitionTo verifica se pode transicionar para outro estado
func (s JobStatus) CanTransitionTo(target JobStatus) bool {
	// Estados terminais não podem mudar
	if s.IsTerminalState() {
		return false
	}

	// Definir transições válidas
	validTransitions := map[JobStatus][]JobStatus{
		JobStatusWaiting:       {JobStatusQueued, JobStatusCancelled, JobStatusFailed},
		JobStatusQueued:        {JobStatusProcessing, JobStatusWaiting, JobStatusCancelled},
		JobStatusProcessing:    {JobStatusPartial, JobStatusCompleted, JobStatusFailed, JobStatusStalled, JobStatusCancelled},
		JobStatusPartial:       {JobStatusProcessing, JobStatusCompleted, JobStatusFailed, JobStatusNeedReprocess},
		JobStatusStalled:       {JobStatusQueued, JobStatusFailed, JobStatusCancelled},
		JobStatusNeedReprocess: {JobStatusQueued, JobStatusProcessing, JobStatusCancelled},
	}

	allowedTargets, exists := validTransitions[s]
	if !exists {
		return false
	}

	for _, allowed := range allowedTargets {
		if target == allowed {
			return true
		}
	}
	return false
}

// JobType define os tipos de jobs disponíveis
type JobType string

const (
	JobTypeCompleteSync JobType = "complete_sync"
)

// Job representa um trabalho na fila com máquina de estados explícita
type Job struct {
	ID          string                 `json:"id"`
	Type        JobType                `json:"type"`
	Status      JobStatus              `json:"status"`
	Parameters  map[string]interface{} `json:"parameters"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Progress    float64                `json:"progress"` // 0.0 a 1.0

	// Novos campos para máquina de estados
	WorkerID        string            `json:"worker_id,omitempty"`        // ID do worker processando
	LastHeartbeat   *time.Time        `json:"last_heartbeat,omitempty"`   // Último heartbeat (detectar stalls)
	RetryCount      int               `json:"retry_count"`                // Número de tentativas
	MaxRetries      int               `json:"max_retries"`                // Máximo de tentativas permitidas
	StateHistory    []StateTransition `json:"state_history,omitempty"`    // Histórico de mudanças de estado
	Dependencies    []string          `json:"dependencies,omitempty"`     // IDs de jobs dependentes
	ReprocessReason string            `json:"reprocess_reason,omitempty"` // Motivo para reprocessamento
}

// StateTransition representa uma mudança de estado
type StateTransition struct {
	FromStatus JobStatus `json:"from_status"`
	ToStatus   JobStatus `json:"to_status"`
	Timestamp  time.Time `json:"timestamp"`
	WorkerID   string    `json:"worker_id,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}

// TransitionTo muda o estado do job com validação
func (j *Job) TransitionTo(newStatus JobStatus, workerID, reason string) error {
	if !j.Status.CanTransitionTo(newStatus) {
		return fmt.Errorf("invalid transition from %s to %s", j.Status, newStatus)
	}

	// Registrar transição no histórico
	transition := StateTransition{
		FromStatus: j.Status,
		ToStatus:   newStatus,
		Timestamp:  time.Now(),
		WorkerID:   workerID,
		Reason:     reason,
	}
	j.StateHistory = append(j.StateHistory, transition)

	// Atualizar status
	oldStatus := j.Status
	j.Status = newStatus

	// Atualizações específicas por estado
	now := time.Now()
	switch newStatus {
	case JobStatusProcessing:
		if oldStatus != JobStatusPartial { // Não resetar se vem de partial
			j.StartedAt = &now
		}
		j.WorkerID = workerID
		j.LastHeartbeat = &now
	case JobStatusPartial:
		j.LastHeartbeat = &now
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		j.CompletedAt = &now
		j.WorkerID = ""
		if newStatus == JobStatusCompleted {
			j.Progress = 1.0
		}
	case JobStatusStalled:
		j.WorkerID = ""
	case JobStatusNeedReprocess:
		j.ReprocessReason = reason
	}

	return nil
}

// ShouldBeRequeued verifica se job deve ser colocado na fila novamente
func (j *Job) ShouldBeRequeued() bool {
	return j.Status == JobStatusStalled || j.Status == JobStatusNeedReprocess
}

// IsEligibleForRetry verifica se job pode ser tentado novamente
func (j *Job) IsEligibleForRetry() bool {
	return j.RetryCount < j.MaxRetries
}

// JobQueue interface para operações de fila
type JobQueue interface {
	// Enqueue adiciona um job à fila
	Enqueue(ctx context.Context, jobType JobType, parameters map[string]interface{}) (*Job, error)

	// Dequeue retira o próximo job da fila para processamento
	Dequeue(ctx context.Context, workerID string) (*Job, error)

	// UpdateStatus atualiza o status de um job
	UpdateStatus(ctx context.Context, jobID string, status JobStatus) error

	// UpdateProgress atualiza o progresso de um job
	UpdateProgress(ctx context.Context, jobID string, progress float64) error

	// Complete marca um job como completo com resultado
	Complete(ctx context.Context, jobID string, result map[string]interface{}) error

	// Fail marca um job como falho com erro
	Fail(ctx context.Context, jobID string, errorMsg string) error

	// GetJob retorna informações de um job específico
	GetJob(ctx context.Context, jobID string) (*Job, error)

	// ListJobs lista jobs com filtros opcionais
	ListJobs(ctx context.Context, status JobStatus, limit int) ([]*Job, error)

	// SendToDeadLetter move um job para a dead letter queue
	SendToDeadLetter(ctx context.Context, jobID string, reason DeadLetterReason, metadata map[string]interface{}) error
}

// RedisJobQueue implementa JobQueue usando Redis
type RedisJobQueue struct {
	client          *redis.Client
	queueKey        string           // chave da fila principal
	statusKey       string           // prefixo para chaves de status
	deadLetterQueue *DeadLetterQueue // dead letter queue
}

// NewRedisJobQueue cria uma nova instância do job queue Redis
func NewRedisJobQueue(client *redis.Client) *RedisJobQueue {
	// Inicializar dead letter queue com TTL de 30 dias
	deadLetterQueue := NewDeadLetterQueue(client, 30*24*time.Hour)

	return &RedisJobQueue{
		client:          client,
		queueKey:        "suno:jobs:queue",
		statusKey:       "suno:jobs:status:",
		deadLetterQueue: deadLetterQueue,
	}
}

// Enqueue adiciona um novo job à fila
func (r *RedisJobQueue) Enqueue(ctx context.Context, jobType JobType, parameters map[string]interface{}) (*Job, error) {
	job := &Job{
		ID:         uuid.New().String(),
		Type:       jobType,
		Status:     JobStatusQueued,
		Parameters: parameters,
		CreatedAt:  time.Now(),
		Progress:   0.0,
		RetryCount: 0,
		MaxRetries: 3, // Default 3 tentativas
		StateHistory: []StateTransition{{
			FromStatus: "", // Job criado
			ToStatus:   JobStatusQueued,
			Timestamp:  time.Now(),
			Reason:     "job_created",
		}},
	}

	// Serializar job
	jobData, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job: %w", err)
	}

	// Adicionar à fila e salvar status
	pipe := r.client.Pipeline()
	pipe.LPush(ctx, r.queueKey, jobData)
	pipe.Set(ctx, r.statusKey+job.ID, jobData, 24*time.Hour) // TTL de 24h

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	return job, nil
}

// Dequeue retira o próximo job da fila
func (r *RedisJobQueue) Dequeue(ctx context.Context, workerID string) (*Job, error) {
	// Verificar se o client Redis está inicializado
	if r.client == nil {
		return nil, fmt.Errorf("redis client is not initialized")
	}

	// Operação atômica: mover da fila principal para fila de processamento
	result, err := r.client.BRPopLPush(ctx, r.queueKey, r.queueKey+":processing", 30*time.Second).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Timeout, sem jobs disponíveis
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	var job Job
	if err := json.Unmarshal([]byte(result), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Transicionar para processing usando a máquina de estados
	if err := job.TransitionTo(JobStatusProcessing, workerID, "dequeued_by_worker"); err != nil {
		return nil, fmt.Errorf("failed to transition job to processing: %w", err)
	}

	if err := r.saveJob(ctx, &job); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	return &job, nil
}

// UpdateStatus atualiza o status de um job usando máquina de estados
func (r *RedisJobQueue) UpdateStatus(ctx context.Context, jobID string, status JobStatus) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	if err := job.TransitionTo(status, job.WorkerID, "status_update"); err != nil {
		return fmt.Errorf("failed to transition job status: %w", err)
	}

	return r.saveJob(ctx, job)
}

// UpdateProgress atualiza o progresso de um job
func (r *RedisJobQueue) UpdateProgress(ctx context.Context, jobID string, progress float64) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Progress = progress
	return r.saveJob(ctx, job)
}

// Complete marca um job como completo
func (r *RedisJobQueue) Complete(ctx context.Context, jobID string, result map[string]interface{}) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Result = result
	if err := job.TransitionTo(JobStatusCompleted, job.WorkerID, "job_completed"); err != nil {
		return fmt.Errorf("failed to transition job to completed: %w", err)
	}

	return r.saveJob(ctx, job)
}

// Fail marca um job como falho
func (r *RedisJobQueue) Fail(ctx context.Context, jobID string, errorMsg string) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Error = errorMsg
	job.RetryCount++

	// Se ainda pode retry, marcar para reprocessamento
	if job.IsEligibleForRetry() {
		if err := job.TransitionTo(JobStatusNeedReprocess, job.WorkerID, fmt.Sprintf("retry_%d: %s", job.RetryCount, errorMsg)); err != nil {
			return fmt.Errorf("failed to transition job to reprocess: %w", err)
		}
		return r.saveJob(ctx, job)
	} else {
		// Falha permanente - mover para dead letter queue
		reason := r.categorizeError(errorMsg)
		metadata := map[string]interface{}{
			"final_error":    errorMsg,
			"total_attempts": job.RetryCount,
			"job_duration": func() int64 {
				if job.StartedAt != nil {
					return time.Since(*job.StartedAt).Milliseconds()
				}
				return 0
			}(),
		}

		// Enviar para dead letter queue
		if err := r.SendToDeadLetter(ctx, jobID, reason, metadata); err != nil {
			// Se falhar ao enviar para DLQ, apenas marcar como failed
			if err := job.TransitionTo(JobStatusFailed, job.WorkerID, fmt.Sprintf("max_retries_exceeded: %s", errorMsg)); err != nil {
				return fmt.Errorf("failed to transition job to failed: %w", err)
			}
			return r.saveJob(ctx, job)
		}

		return nil
	}
}

// categorizeError categoriza o erro para determinar o motivo da dead letter
func (r *RedisJobQueue) categorizeError(errorMsg string) DeadLetterReason {
	errorLower := strings.ToLower(errorMsg)

	switch {
	case strings.Contains(errorLower, "timeout") || strings.Contains(errorLower, "deadline"):
		return DeadLetterReasonTimeout
	case strings.Contains(errorLower, "permission") || strings.Contains(errorLower, "unauthorized") || strings.Contains(errorLower, "forbidden"):
		return DeadLetterReasonPermissionDenied
	case strings.Contains(errorLower, "not found") || strings.Contains(errorLower, "missing"):
		return DeadLetterReasonResourceNotFound
	case strings.Contains(errorLower, "rate limit") || strings.Contains(errorLower, "too many requests"):
		return DeadLetterReasonRateLimited
	case strings.Contains(errorLower, "invalid") || strings.Contains(errorLower, "malformed") || strings.Contains(errorLower, "bad request"):
		return DeadLetterReasonInvalidParameters
	case strings.Contains(errorLower, "circuit breaker"):
		return DeadLetterReasonCircuitBreakerOpen
	default:
		return DeadLetterReasonSystemError
	}
}

// GetJob retorna informações de um job específico
func (r *RedisJobQueue) GetJob(ctx context.Context, jobID string) (*Job, error) {
	result, err := r.client.Get(ctx, r.statusKey+jobID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	var job Job
	if err := json.Unmarshal([]byte(result), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// ListJobs lista jobs com filtros opcionais
func (r *RedisJobQueue) ListJobs(ctx context.Context, status JobStatus, limit int) ([]*Job, error) {
	// Implementação simplificada - em produção poderia usar índices Redis
	// Por ora, retorna jobs por scan das chaves de status
	iter := r.client.Scan(ctx, 0, r.statusKey+"*", int64(limit)).Iterator()

	var jobs []*Job
	for iter.Next(ctx) {
		key := iter.Val()
		result, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue // Skip erros
		}

		var job Job
		if err := json.Unmarshal([]byte(result), &job); err != nil {
			continue // Skip erros
		}

		if status == "" || job.Status == status {
			jobs = append(jobs, &job)
		}

		if len(jobs) >= limit {
			break
		}
	}

	return jobs, nil
}

// Heartbeat atualiza o último heartbeat de um job ativo
func (r *RedisJobQueue) Heartbeat(ctx context.Context, jobID string, workerID string) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	if !job.Status.IsActiveState() {
		return fmt.Errorf("job %s is not in active state: %s", jobID, job.Status)
	}

	if job.WorkerID != workerID {
		return fmt.Errorf("job %s is not assigned to worker %s", jobID, workerID)
	}

	now := time.Now()
	job.LastHeartbeat = &now

	return r.saveJob(ctx, job)
}

// MarkPartial marca um job como parcialmente processado
func (r *RedisJobQueue) MarkPartial(ctx context.Context, jobID string, workerID string, progress float64) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Progress = progress
	if err := job.TransitionTo(JobStatusPartial, workerID, fmt.Sprintf("partial_progress_%.2f", progress)); err != nil {
		return fmt.Errorf("failed to transition job to partial: %w", err)
	}

	return r.saveJob(ctx, job)
}

// DetectStalledJobs encontra jobs que não enviaram heartbeat há muito tempo
func (r *RedisJobQueue) DetectStalledJobs(ctx context.Context, timeout time.Duration) ([]*Job, error) {
	var stalledJobs []*Job

	// Buscar todos os jobs ativos
	activeJobs, err := r.ListJobs(ctx, JobStatusProcessing, 1000)
	if err != nil {
		return nil, err
	}

	// Adicionar jobs partial também
	partialJobs, err := r.ListJobs(ctx, JobStatusPartial, 1000)
	if err != nil {
		return nil, err
	}
	activeJobs = append(activeJobs, partialJobs...)

	now := time.Now()
	for _, job := range activeJobs {
		if job.LastHeartbeat == nil {
			// Job sem heartbeat inicial, verificar StartedAt
			if job.StartedAt != nil && now.Sub(*job.StartedAt) > timeout {
				stalledJobs = append(stalledJobs, job)
			}
		} else if now.Sub(*job.LastHeartbeat) > timeout {
			stalledJobs = append(stalledJobs, job)
		}
	}

	return stalledJobs, nil
}

// RequeueStalled marca jobs stalled para reprocessamento
func (r *RedisJobQueue) RequeueStalled(ctx context.Context, timeout time.Duration) (int, error) {
	stalledJobs, err := r.DetectStalledJobs(ctx, timeout)
	if err != nil {
		return 0, err
	}

	requeuedCount := 0
	for _, job := range stalledJobs {
		if err := job.TransitionTo(JobStatusStalled, "", fmt.Sprintf("stalled_after_%v", timeout)); err == nil {
			if err := r.saveJob(ctx, job); err == nil {
				requeuedCount++
			}
		}
	}

	return requeuedCount, nil
}

// SendToDeadLetter move um job para a dead letter queue
func (r *RedisJobQueue) SendToDeadLetter(ctx context.Context, jobID string, reason DeadLetterReason, metadata map[string]interface{}) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job for dead letter: %w", err)
	}

	// Adicionar à dead letter queue
	if err := r.deadLetterQueue.Add(ctx, job, reason, metadata); err != nil {
		return fmt.Errorf("failed to add job to dead letter queue: %w", err)
	}

	// Remover da fila principal (se ainda estiver lá)
	_ = r.client.LRem(ctx, r.queueKey, 0, jobID)
	_ = r.client.LRem(ctx, r.queueKey+":processing", 0, jobID)

	// Manter status job por um tempo para auditoria
	if err := job.TransitionTo(JobStatusFailed, job.WorkerID, fmt.Sprintf("moved_to_dead_letter: %s", reason)); err == nil {
		_ = r.saveJob(ctx, job)
	}

	return nil
}

// GetDeadLetterQueue retorna a instância da dead letter queue
func (r *RedisJobQueue) GetDeadLetterQueue() *DeadLetterQueue {
	return r.deadLetterQueue
}

// saveJob salva um job no Redis
func (r *RedisJobQueue) saveJob(ctx context.Context, job *Job) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return r.client.Set(ctx, r.statusKey+job.ID, jobData, 24*time.Hour).Err()
}
