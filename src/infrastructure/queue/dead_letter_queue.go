package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"suno-wallets/src/shared/helpers"

	"github.com/redis/go-redis/v9"
)

// DeadLetterJob representa um job que falhou permanentemente
type DeadLetterJob struct {
	ID            string                 `json:"id"`
	OriginalJob   Job                    `json:"original_job"`
	FailureReason string                 `json:"failure_reason"`
	AttemptCount  int                    `json:"attempt_count"`
	LastAttemptAt time.Time              `json:"last_attempt_at"`
	CreatedAt     time.Time              `json:"created_at"`
	Tags          []string               `json:"tags,omitempty"`        // Para categorização
	Priority      string                 `json:"priority,omitempty"`    // HIGH, NORMAL, LOW
	Metadata      map[string]interface{} `json:"metadata,omitempty"`    // Informações adicionais
	RetryAfter    *time.Time             `json:"retry_after,omitempty"` // Quando pode tentar novamente
	ManualReview  bool                   `json:"manual_review"`         // Precisa revisão manual
}

// DeadLetterReason categoriza motivos de falha
type DeadLetterReason string

const (
	DeadLetterReasonMaxRetriesExceeded DeadLetterReason = "max_retries_exceeded"
	DeadLetterReasonInvalidParameters  DeadLetterReason = "invalid_parameters"
	DeadLetterReasonResourceNotFound   DeadLetterReason = "resource_not_found"
	DeadLetterReasonPermissionDenied   DeadLetterReason = "permission_denied"
	DeadLetterReasonSystemError        DeadLetterReason = "system_error"
	DeadLetterReasonTimeout            DeadLetterReason = "timeout"
	DeadLetterReasonRateLimited        DeadLetterReason = "rate_limited"
	DeadLetterReasonCircuitBreakerOpen DeadLetterReason = "circuit_breaker_open"
)

// IsRetriable indica se uma falha pode ser tentada novamente
func (r DeadLetterReason) IsRetriable() bool {
	switch r {
	case DeadLetterReasonTimeout, DeadLetterReasonRateLimited,
		DeadLetterReasonCircuitBreakerOpen, DeadLetterReasonSystemError:
		return true
	default:
		return false
	}
}

// DeadLetterQueue gerencia jobs que falharam permanentemente
type DeadLetterQueue struct {
	redisClient *redis.Client
	keyPrefix   string
	ttl         time.Duration
}

// NewDeadLetterQueue cria uma nova instância da dead letter queue
func NewDeadLetterQueue(redisClient *redis.Client, ttl time.Duration) *DeadLetterQueue {
	return &DeadLetterQueue{
		redisClient: redisClient,
		keyPrefix:   "suno:dead_letter:",
		ttl:         ttl,
	}
}

// Add adiciona um job à dead letter queue
func (dlq *DeadLetterQueue) Add(ctx context.Context, job *Job, reason DeadLetterReason, metadata map[string]interface{}) error {
	now := time.Now()

	deadJob := DeadLetterJob{
		ID:            job.ID,
		OriginalJob:   *job,
		FailureReason: string(reason),
		AttemptCount:  job.RetryCount,
		LastAttemptAt: now,
		CreatedAt:     now,
		Tags:          dlq.generateTags(job, reason),
		Priority:      dlq.determinePriority(job, reason),
		Metadata:      metadata,
		ManualReview:  dlq.needsManualReview(reason),
	}

	// Determinar se pode ser tentado novamente automaticamente
	if reason.IsRetriable() {
		retryDelay := dlq.calculateRetryDelay(job.RetryCount)
		retryAfter := now.Add(retryDelay)
		deadJob.RetryAfter = &retryAfter
	}

	// Serializar para JSON
	data, err := json.Marshal(deadJob)
	if err != nil {
		return fmt.Errorf("failed to marshal dead letter job: %w", err)
	}

	// Salvar no Redis
	key := dlq.keyPrefix + job.ID
	if err := dlq.redisClient.Set(ctx, key, data, dlq.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save dead letter job: %w", err)
	}

	// Adicionar às listas por categoria para facilitar consultas
	pipe := dlq.redisClient.Pipeline()
	pipe.LPush(ctx, dlq.keyPrefix+"list:all", job.ID)
	pipe.LPush(ctx, dlq.keyPrefix+"list:reason:"+string(reason), job.ID)
	pipe.LPush(ctx, dlq.keyPrefix+"list:priority:"+deadJob.Priority, job.ID)

	if deadJob.ManualReview {
		pipe.LPush(ctx, dlq.keyPrefix+"list:manual_review", job.ID)
	}

	if deadJob.RetryAfter != nil {
		pipe.LPush(ctx, dlq.keyPrefix+"list:retriable", job.ID)
		// Adicionar à lista ordenada por tempo de retry
		pipe.ZAdd(ctx, dlq.keyPrefix+"retry_schedule", redis.Z{
			Score:  float64(deadJob.RetryAfter.Unix()),
			Member: job.ID,
		})
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		helpers.LogError("failed to update dead letter lists", err, map[string]interface{}{
			"job_id": job.ID,
			"reason": reason,
		})
	}

	helpers.LogInfo("job added to dead letter queue", map[string]interface{}{
		"job_id":        job.ID,
		"job_type":      job.Type,
		"reason":        reason,
		"attempt_count": job.RetryCount,
		"manual_review": deadJob.ManualReview,
		"retriable":     deadJob.RetryAfter != nil,
	})

	return nil
}

// Get recupera um job da dead letter queue
func (dlq *DeadLetterQueue) Get(ctx context.Context, jobID string) (*DeadLetterJob, error) {
	key := dlq.keyPrefix + jobID
	data, err := dlq.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("dead letter job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get dead letter job: %w", err)
	}

	var deadJob DeadLetterJob
	if err := json.Unmarshal([]byte(data), &deadJob); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dead letter job: %w", err)
	}

	return &deadJob, nil
}

// List retorna jobs da dead letter queue com filtros
func (dlq *DeadLetterQueue) List(ctx context.Context, filter DeadLetterFilter) ([]*DeadLetterJob, error) {
	var listKey string

	// Determinar qual lista usar baseado no filtro
	switch {
	case filter.Reason != "":
		listKey = dlq.keyPrefix + "list:reason:" + filter.Reason
	case filter.Priority != "":
		listKey = dlq.keyPrefix + "list:priority:" + filter.Priority
	case filter.ManualReviewOnly:
		listKey = dlq.keyPrefix + "list:manual_review"
	case filter.RetriableOnly:
		listKey = dlq.keyPrefix + "list:retriable"
	default:
		listKey = dlq.keyPrefix + "list:all"
	}

	// Obter IDs da lista
	jobIDs, err := dlq.redisClient.LRange(ctx, listKey, 0, int64(filter.Limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list dead letter jobs: %w", err)
	}

	// Buscar jobs completos
	var jobs []*DeadLetterJob
	for _, jobID := range jobIDs {
		job, err := dlq.Get(ctx, jobID)
		if err != nil {
			helpers.LogError("failed to get dead letter job", err, map[string]interface{}{
				"job_id": jobID,
			})
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// GetRetriableJobs retorna jobs que podem ser tentados novamente
func (dlq *DeadLetterQueue) GetRetriableJobs(ctx context.Context, limit int) ([]*DeadLetterJob, error) {
	now := time.Now()

	// Buscar jobs que já passaram do tempo de retry
	jobIDs, err := dlq.redisClient.ZRangeByScore(ctx, dlq.keyPrefix+"retry_schedule", &redis.ZRangeBy{
		Min:   "0",
		Max:   fmt.Sprintf("%d", now.Unix()),
		Count: int64(limit),
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to get retriable jobs: %w", err)
	}

	var jobs []*DeadLetterJob
	for _, jobID := range jobIDs {
		job, err := dlq.Get(ctx, jobID)
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// Requeue recoloca um job da dead letter queue na fila principal
func (dlq *DeadLetterQueue) Requeue(ctx context.Context, jobID string, jobQueue JobQueue) error {
	deadJob, err := dlq.Get(ctx, jobID)
	if err != nil {
		return err
	}

	// Recriar job com retry count resetado (se desejado)
	newJob := deadJob.OriginalJob
	newJob.RetryCount = 0
	newJob.Status = JobStatusQueued
	newJob.Error = ""
	newJob.StartedAt = nil
	newJob.CompletedAt = nil
	newJob.WorkerID = ""
	newJob.LastHeartbeat = nil

	// Adicionar à fila principal
	_, err = jobQueue.Enqueue(ctx, newJob.Type, newJob.Parameters)
	if err != nil {
		return fmt.Errorf("failed to requeue job: %w", err)
	}

	// Remover da dead letter queue
	err = dlq.Remove(ctx, jobID)
	if err != nil {
		helpers.LogError("failed to remove job from dead letter queue after requeue", err, map[string]interface{}{
			"job_id": jobID,
		})
	}

	helpers.LogInfo("job requeued from dead letter queue", map[string]interface{}{
		"job_id":   jobID,
		"job_type": newJob.Type,
	})

	return nil
}

// Remove remove um job da dead letter queue
func (dlq *DeadLetterQueue) Remove(ctx context.Context, jobID string) error {
	key := dlq.keyPrefix + jobID

	// Remover job principal
	if err := dlq.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to remove dead letter job: %w", err)
	}

	// Remover das listas (melhor esforço)
	pipe := dlq.redisClient.Pipeline()
	pipe.LRem(ctx, dlq.keyPrefix+"list:all", 0, jobID)
	pipe.LRem(ctx, dlq.keyPrefix+"list:manual_review", 0, jobID)
	pipe.LRem(ctx, dlq.keyPrefix+"list:retriable", 0, jobID)
	pipe.ZRem(ctx, dlq.keyPrefix+"retry_schedule", jobID)

	_, err := pipe.Exec(ctx)
	if err != nil {
		helpers.LogError("failed to remove job from dead letter lists", err, map[string]interface{}{
			"job_id": jobID,
		})
	}

	return nil
}

// DeadLetterFilter define filtros para listar jobs
type DeadLetterFilter struct {
	Reason           string
	Priority         string
	ManualReviewOnly bool
	RetriableOnly    bool
	Limit            int
}

// generateTags gera tags para categorização do job
func (dlq *DeadLetterQueue) generateTags(job *Job, reason DeadLetterReason) []string {
	tags := []string{string(job.Type), string(reason)}

	// Adicionar tags baseadas nos parâmetros do job
	if tenantID, ok := job.Parameters["tenantID"].(string); ok && tenantID != "" {
		tags = append(tags, "tenant:"+tenantID)
	}

	return tags
}

// determinePriority determina a prioridade baseada no job e motivo
func (dlq *DeadLetterQueue) determinePriority(job *Job, reason DeadLetterReason) string {
	// Jobs com falhas de sistema têm prioridade alta para retry
	if reason == DeadLetterReasonSystemError || reason == DeadLetterReasonTimeout {
		return "HIGH"
	}

	// Jobs com parâmetros inválidos têm prioridade baixa
	if reason == DeadLetterReasonInvalidParameters {
		return "LOW"
	}

	return "NORMAL"
}

// needsManualReview determina se o job precisa de revisão manual
func (dlq *DeadLetterQueue) needsManualReview(reason DeadLetterReason) bool {
	switch reason {
	case DeadLetterReasonInvalidParameters, DeadLetterReasonPermissionDenied:
		return true
	default:
		return false
	}
}

// calculateRetryDelay calcula delay para próxima tentativa
func (dlq *DeadLetterQueue) calculateRetryDelay(attemptCount int) time.Duration {
	// Backoff exponencial com jitter: 1m, 5m, 15m, 1h, 4h, 12h
	delays := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		1 * time.Hour,
		4 * time.Hour,
		12 * time.Hour,
	}

	if attemptCount >= len(delays) {
		return delays[len(delays)-1] // Máximo de 12h
	}

	return delays[attemptCount]
}
