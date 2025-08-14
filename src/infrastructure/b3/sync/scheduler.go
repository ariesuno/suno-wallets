package sync

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
)

// Comentários em pt-BR: agendador diário com Asynq (stub simples)

const TaskTypeDailySync = "b3:daily_sync"

func EnqueueDailySync(client *asynq.Client) (*asynq.TaskInfo, error) {
	task := asynq.NewTask(TaskTypeDailySync, nil)
	return client.Enqueue(task, asynq.Unique(24*time.Hour))
}

func ScheduleDaily(cron *asynq.Scheduler, spec string) (string, error) {
	id, err := cron.Register(spec, asynq.NewTask(TaskTypeDailySync, nil))
	if err != nil {
		return "", err
	}
	return id, nil
}

type Handler interface {
	Handle(ctx context.Context) error
}

func NewServer(mgr *asynq.ServeMux, handler Handler) {
	mgr.HandleFunc(TaskTypeDailySync, func(ctx context.Context, t *asynq.Task) error { return handler.Handle(ctx) })
}
