package worker

import (
	"context"
	"time"

	"suno-wallets/src/infrastructure/queue"
	"suno-wallets/src/shared/helpers"
)

// StallDetector detecta e reprocessa jobs que ficaram stalled
type StallDetector struct {
	jobQueue      queue.JobQueue
	stallTimeout  time.Duration
	checkInterval time.Duration
	isRunning     bool
	stopChan      chan struct{}
}

// NewStallDetector cria uma nova instância do detector de stalls
func NewStallDetector(jobQueue queue.JobQueue, stallTimeout, checkInterval time.Duration) *StallDetector {
	return &StallDetector{
		jobQueue:      jobQueue,
		stallTimeout:  stallTimeout,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

// Start inicia o detector de stalls em background
func (d *StallDetector) Start(ctx context.Context) {
	if d.isRunning {
		return
	}

	d.isRunning = true
	go d.run(ctx)

	helpers.LogInfo("stall detector started", map[string]interface{}{
		"stallTimeout":  d.stallTimeout.String(),
		"checkInterval": d.checkInterval.String(),
	})
}

// Stop para o detector de stalls
func (d *StallDetector) Stop() {
	if !d.isRunning {
		return
	}

	close(d.stopChan)
	d.isRunning = false

	helpers.LogInfo("stall detector stopped", nil)
}

// run loop principal do detector
func (d *StallDetector) run(ctx context.Context) {
	ticker := time.NewTicker(d.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.detectAndRequeueStalled(ctx)
		}
	}
}

// detectAndRequeueStalled detecta jobs stalled e os recoloca na fila
func (d *StallDetector) detectAndRequeueStalled(ctx context.Context) {
	redisQueue, ok := d.jobQueue.(*queue.RedisJobQueue)
	if !ok {
		// Só funciona com RedisJobQueue
		return
	}

	requeuedCount, err := redisQueue.RequeueStalled(ctx, d.stallTimeout)
	if err != nil {
		helpers.LogError("failed to requeue stalled jobs", err, nil)
		return
	}

	if requeuedCount > 0 {
		helpers.LogInfo("requeued stalled jobs", map[string]interface{}{
			"count":        requeuedCount,
			"stallTimeout": d.stallTimeout.String(),
		})
	}
}
