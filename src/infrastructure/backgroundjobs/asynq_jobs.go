package backgroundjobs

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
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

// EnqueueExample enfileira um job de exemplo
func EnqueueExample(ctx context.Context, client *asynq.Client, payload []byte) (*asynq.TaskInfo, error) {
	task := asynq.NewTask("example:do", payload, asynq.MaxRetry(3))
	return client.EnqueueContext(ctx, task, asynq.Queue("default"))
}
