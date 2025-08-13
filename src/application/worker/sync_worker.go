package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	completesync "suno-wallets/src/application/b3/sync"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/infrastructure/queue"
	"suno-wallets/src/shared/helpers"
)

// SyncWorker processa jobs de sincronização em background
type SyncWorker struct {
	jobQueue         queue.JobQueue
	syncOrchestrator *completesync.CompleteSyncOrchestrator
	workerID         string
	isRunning        bool
	stopChan         chan struct{}
	wg               sync.WaitGroup
	pollInterval     time.Duration
}

// NewSyncWorker cria uma nova instância do worker
func NewSyncWorker(
	jobQueue queue.JobQueue,
	syncOrchestrator *completesync.CompleteSyncOrchestrator,
	workerID string,
) *SyncWorker {
	return &SyncWorker{
		jobQueue:         jobQueue,
		syncOrchestrator: syncOrchestrator,
		workerID:         workerID,
		stopChan:         make(chan struct{}),
		pollInterval:     2 * time.Second, // Poll a cada 2 segundos
	}
}

// Start inicia o worker em background
func (w *SyncWorker) Start(ctx context.Context) {
	if w.isRunning {
		return
	}

	w.isRunning = true
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		w.run(ctx)
	}()

	helpers.LogInfo("sync worker started", map[string]interface{}{
		"workerID": w.workerID,
	})
}

// Stop para o worker gracefully
func (w *SyncWorker) Stop() {
	if !w.isRunning {
		return
	}

	close(w.stopChan)
	w.wg.Wait()
	w.isRunning = false

	helpers.LogInfo("sync worker stopped", map[string]interface{}{
		"workerID": w.workerID,
	})
}

// run loop principal do worker
func (w *SyncWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			helpers.LogInfo("sync worker stopping", map[string]interface{}{
				"workerID": w.workerID,
			})
			return

		case <-ticker.C:
			w.processNextJob(ctx)

		case <-ctx.Done():
			helpers.LogInfo("sync worker context cancelled", map[string]interface{}{
				"workerID": w.workerID,
			})
			return
		}
	}
}

// processNextJob processa o próximo job disponível na fila
func (w *SyncWorker) processNextJob(ctx context.Context) {
	job, err := w.jobQueue.Dequeue(ctx, w.workerID)
	if err != nil {
		helpers.LogError("failed to dequeue job", err, map[string]interface{}{
			"workerID": w.workerID,
		})
		return
	}

	if job == nil {
		// Sem jobs disponíveis
		return
	}

	helpers.LogInfo("processing job", map[string]interface{}{
		"workerID": w.workerID,
		"jobID":    job.ID,
		"jobType":  job.Type,
	})

	// Processar job baseado no tipo
	switch job.Type {
	case queue.JobTypeCompleteSync:
		w.processCompleteSyncJob(ctx, job)
	default:
		w.failJob(ctx, job.ID, fmt.Sprintf("unknown job type: %s", job.Type))
	}
}

// processCompleteSyncJob processa um job de sincronização completa
func (w *SyncWorker) processCompleteSyncJob(ctx context.Context, job *queue.Job) {
	started := time.Now()
	observability.IncCompleteSyncOrchestrator("worker_started")

	// Extrair parâmetros do job
	params, err := w.extractSyncParams(job.Parameters)
	if err != nil {
		w.failJob(ctx, job.ID, fmt.Sprintf("invalid job parameters: %v", err))
		return
	}

	// Atualizar progresso: iniciando
	w.updateProgress(ctx, job.ID, 0.1)

	// Executar sincronização
	result, err := w.syncOrchestrator.ExecuteCompleteSync(ctx, *params)
	if err != nil {
		observability.IncCompleteSyncOrchestrator("worker_error")
		w.failJob(ctx, job.ID, fmt.Sprintf("sync failed: %v", err))
		return
	}

	// Atualizar progresso: finalizado
	w.updateProgress(ctx, job.ID, 1.0)

	// Preparar resultado para retorno
	jobResult := map[string]interface{}{
		"success":              result.Success,
		"strategy":             result.Strategy,
		"clientStatus":         result.ClientStatus,
		"newDataIngested":      result.NewDataIngested,
		"durationMs":           result.DurationMs,
		"dataSummary":          result.DataSummary,
		"processedDataTypes":   result.ProcessedDataTypes,
		"reconciliationResult": result.ReconciliationResult,
	}

	if result.ErrorMessage != "" {
		jobResult["errorMessage"] = result.ErrorMessage
	}

	// Marcar job como completo
	if err := w.jobQueue.Complete(ctx, job.ID, jobResult); err != nil {
		helpers.LogError("failed to mark job as complete", err, map[string]interface{}{
			"workerID": w.workerID,
			"jobID":    job.ID,
		})
		return
	}

	observability.IncCompleteSyncOrchestrator("worker_success")
	helpers.LogInfo("job completed successfully", map[string]interface{}{
		"workerID":   w.workerID,
		"jobID":      job.ID,
		"durationMs": time.Since(started).Milliseconds(),
		"success":    result.Success,
	})
}

// extractSyncParams extrai parâmetros de sincronização do job
func (w *SyncWorker) extractSyncParams(params map[string]interface{}) (*completesync.CompleteSyncParams, error) {
	cpf, ok := params["cpf"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid cpf")
	}

	tenantID, ok := params["tenantID"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid tenantID")
	}

	// Extrair arrays com conversão de interface{}
	assetTypesRaw, ok := params["assetTypes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid assetTypes")
	}
	assetTypes := make([]string, len(assetTypesRaw))
	for i, v := range assetTypesRaw {
		assetTypes[i] = v.(string)
	}

	dataTypesRaw, ok := params["dataTypes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid dataTypes")
	}
	dataTypes := make([]string, len(dataTypesRaw))
	for i, v := range dataTypesRaw {
		dataTypes[i] = v.(string)
	}

	// Parâmetros opcionais com valores padrão
	includeReconciliation, _ := params["includeReconciliation"].(bool)
	dryRun, _ := params["dryRun"].(bool)
	force, _ := params["force"].(bool)

	return &completesync.CompleteSyncParams{
		CPF:                   cpf,
		TenantID:              tenantID,
		AssetTypes:            assetTypes,
		DataTypes:             dataTypes,
		IncludeReconciliation: includeReconciliation,
		DryRun:                dryRun,
		Force:                 force,
	}, nil
}

// updateProgress atualiza o progresso de um job
func (w *SyncWorker) updateProgress(ctx context.Context, jobID string, progress float64) {
	if err := w.jobQueue.UpdateProgress(ctx, jobID, progress); err != nil {
		// Log mas não falha o job por isso
		helpers.LogError("failed to update job progress", err, map[string]interface{}{
			"workerID": w.workerID,
			"jobID":    jobID,
			"progress": progress,
		})
	}
}

// failJob marca um job como falho
func (w *SyncWorker) failJob(ctx context.Context, jobID string, errorMsg string) {
	if err := w.jobQueue.Fail(ctx, jobID, errorMsg); err != nil {
		helpers.LogError("failed to mark job as failed", err, map[string]interface{}{
			"workerID":      w.workerID,
			"jobID":         jobID,
			"originalError": errorMsg,
		})
	}

	helpers.LogError("job failed", fmt.Errorf(errorMsg), map[string]interface{}{
		"workerID": w.workerID,
		"jobID":    jobID,
	})
}

// WorkerPool gerencia múltiplos workers
type WorkerPool struct {
	workers []*SyncWorker
	size    int
}

// NewWorkerPool cria um pool de workers
func NewWorkerPool(
	size int,
	jobQueue queue.JobQueue,
	syncOrchestrator *completesync.CompleteSyncOrchestrator,
) *WorkerPool {
	workers := make([]*SyncWorker, size)
	for i := 0; i < size; i++ {
		workerID := fmt.Sprintf("worker-%d", i+1)
		workers[i] = NewSyncWorker(jobQueue, syncOrchestrator, workerID)
	}

	return &WorkerPool{
		workers: workers,
		size:    size,
	}
}

// Start inicia todos os workers do pool
func (p *WorkerPool) Start(ctx context.Context) {
	for _, worker := range p.workers {
		worker.Start(ctx)
	}

	helpers.LogInfo("worker pool started", map[string]interface{}{
		"size": p.size,
	})
}

// Stop para todos os workers do pool
func (p *WorkerPool) Stop() {
	for _, worker := range p.workers {
		worker.Stop()
	}

	helpers.LogInfo("worker pool stopped", map[string]interface{}{
		"size": p.size,
	})
}
