package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// JobStatus representa o status de um job
type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

// JobType define os tipos de jobs disponíveis
type JobType string

const (
	JobTypeCompleteSync JobType = "complete_sync"
)

// Job representa um trabalho na fila
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
}

// RedisJobQueue implementa JobQueue usando Redis
type RedisJobQueue struct {
	client    *redis.Client
	queueKey  string // chave da fila principal
	statusKey string // prefixo para chaves de status
}

// NewRedisJobQueue cria uma nova instância do job queue Redis
func NewRedisJobQueue(client *redis.Client) *RedisJobQueue {
	return &RedisJobQueue{
		client:    client,
		queueKey:  "suno:jobs:queue",
		statusKey: "suno:jobs:status:",
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

	// Atualizar status para processing
	now := time.Now()
	job.Status = JobStatusProcessing
	job.StartedAt = &now

	if err := r.saveJob(ctx, &job); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	return &job, nil
}

// UpdateStatus atualiza o status de um job
func (r *RedisJobQueue) UpdateStatus(ctx context.Context, jobID string, status JobStatus) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Status = status
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

	now := time.Now()
	job.Status = JobStatusCompleted
	job.Result = result
	job.CompletedAt = &now
	job.Progress = 1.0

	return r.saveJob(ctx, job)
}

// Fail marca um job como falho
func (r *RedisJobQueue) Fail(ctx context.Context, jobID string, errorMsg string) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	now := time.Now()
	job.Status = JobStatusFailed
	job.Error = errorMsg
	job.CompletedAt = &now

	return r.saveJob(ctx, job)
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

// saveJob salva um job no Redis
func (r *RedisJobQueue) saveJob(ctx context.Context, job *Job) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return r.client.Set(ctx, r.statusKey+job.ID, jobData, 24*time.Hour).Err()
}
