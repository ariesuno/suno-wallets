package backgroundjobs

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// NewAsynqClient cria um cliente Asynq conectado ao Redis
func NewAsynqClient(redisAddr, password string, db int) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr, Password: password, DB: db})
}

// NewAsynqServer cria um servidor Asynq com configurações de concorrência e timeouts
func NewAsynqServer(redisAddr, password string, db int) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: password, DB: db},
		asynq.Config{
			Concurrency:     5,
			ShutdownTimeout: 10 * time.Second,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)
}

// HandleWithMetrics registra handler de task com coleta de métricas
func HandleWithMetrics(mux *asynq.ServeMux, taskType string, handler func(context.Context, *asynq.Task) error) {
	jobDuration := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "asynq_job_duration_seconds",
			Help:    "Duração dos jobs Asynq",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type", "status"},
	)
	jobTotal := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "asynq_jobs_total",
			Help: "Total de execuções de jobs Asynq",
		},
		[]string{"type", "status"},
	)

	mux.HandleFunc(taskType, func(ctx context.Context, t *asynq.Task) error {
		start := time.Now()
		err := handler(ctx, t)
		status := "success"
		if err != nil {
			status = "error"
		}
		jobTotal.WithLabelValues(taskType, status).Inc()
		jobDuration.WithLabelValues(taskType, status).Observe(time.Since(start).Seconds())
		return err
	})
}

// EnqueueExample enfileira um job de exemplo
func EnqueueExample(ctx context.Context, client *asynq.Client, payload []byte) (*asynq.TaskInfo, error) {
	task := asynq.NewTask("example:do", payload, asynq.MaxRetry(3))
	return client.EnqueueContext(ctx, task, asynq.Queue("default"))
}
