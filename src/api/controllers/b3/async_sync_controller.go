package b3

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"suno-wallets/src/api/middlewares"
	"suno-wallets/src/infrastructure/queue"
	"suno-wallets/src/shared/validation"
)

// AsyncSyncController gerencia endpoints de sincronização assíncrona
type AsyncSyncController struct {
	jobQueue queue.JobQueue
}

// NewAsyncSyncController cria uma nova instância do controller
func NewAsyncSyncController(jobQueue queue.JobQueue) *AsyncSyncController {
	return &AsyncSyncController{
		jobQueue: jobQueue,
	}
}

// asyncSyncRequest representa a estrutura de requisição para sincronização assíncrona
type asyncSyncRequest struct {
	CPF                   string   `json:"cpf" binding:"required"`
	AssetTypes            []string `json:"assetTypes" binding:"required"`
	DataTypes             []string `json:"dataTypes" binding:"required"`
	IncludeReconciliation bool     `json:"includeReconciliation"`
	DryRun                bool     `json:"dryRun"`
	Force                 bool     `json:"force"`
}

// asyncSyncResponse representa a resposta do endpoint assíncrono
type asyncSyncResponse struct {
	JobID                    string `json:"jobId"`
	Status                   string `json:"status"`
	Message                  string `json:"message"`
	QueuedAt                 string `json:"queuedAt"`
	EstimatedDurationMinutes int    `json:"estimatedDurationMinutes"`
}

// QueueCompleteSync godoc
// @Summary Enfileirar sincronização completa B3 (Assíncrono)
// @Description Inicia sincronização completa assíncrona. Retorna imediatamente um job ID para acompanhar o progresso. Ideal para processamento em lote ou quando não é necessário aguardar o resultado.
// @Tags B3 Async
// @Accept json
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param request body asyncSyncRequest true "Parâmetros da sincronização assíncrona" example({"cpf": "12345678901", "assetTypes": ["equity"], "dataTypes": ["transactions", "positions"], "includeReconciliation": true, "dryRun": false, "force": false})
// @Success 202 {object} asyncSyncResponse "Job enfileirado com sucesso"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /b3/async/sync/complete-ingestion [post]
func (c *AsyncSyncController) QueueCompleteSync(ctx *gin.Context) {
	tenantID := middlewares.MustGetTenantName(ctx)

	var req asyncSyncRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validar CPF
	if err := validation.ValidateCPF(req.CPF); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validar asset types e data types
	if len(req.AssetTypes) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "assetTypes cannot be empty"})
		return
	}
	if len(req.DataTypes) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "dataTypes cannot be empty"})
		return
	}

	// Preparar parâmetros do job
	jobParams := map[string]interface{}{
		"cpf":                   req.CPF,
		"tenantID":              tenantID,
		"assetTypes":            req.AssetTypes,
		"dataTypes":             req.DataTypes,
		"includeReconciliation": req.IncludeReconciliation,
		"dryRun":                req.DryRun,
		"force":                 req.Force,
	}

	// Enfileirar job
	job, err := c.jobQueue.Enqueue(ctx, queue.JobTypeCompleteSync, jobParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue job: " + err.Error()})
		return
	}

	// Estimar duração baseada na experiência (pode ser refinado)
	estimatedMinutes := 1 // Para um cliente típico
	if !req.DryRun {
		estimatedMinutes = 3 // Mais tempo para processamento real
	}

	response := asyncSyncResponse{
		JobID:                    job.ID,
		Status:                   string(job.Status),
		Message:                  "Job enfileirado com sucesso. Use o jobId para acompanhar o progresso.",
		QueuedAt:                 job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		EstimatedDurationMinutes: estimatedMinutes,
	}

	ctx.JSON(http.StatusAccepted, response)
}

// jobStatusResponse representa a resposta do status de um job
type jobStatusResponse struct {
	JobID       string                 `json:"jobId"`
	Status      string                 `json:"status"`
	Progress    float64                `json:"progress"`
	CreatedAt   string                 `json:"createdAt"`
	StartedAt   *string                `json:"startedAt,omitempty"`
	CompletedAt *string                `json:"completedAt,omitempty"`
	DurationMs  *int64                 `json:"durationMs,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// GetJobStatus godoc
// @Summary Consultar status de job de sincronização
// @Description Retorna o status atual, progresso e resultado de um job de sincronização assíncrona. Use este endpoint para acompanhar o progresso de jobs iniciados via /async/sync/complete-ingestion.
// @Tags B3 Async
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param jobId path string true "ID do job a consultar" example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {object} jobStatusResponse "Status do job"
// @Failure 404 {object} map[string]string "Job não encontrado"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /b3/async/jobs/{jobId}/status [get]
func (c *AsyncSyncController) GetJobStatus(ctx *gin.Context) {
	jobID := ctx.Param("jobId")
	if jobID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "jobId is required"})
		return
	}

	job, err := c.jobQueue.GetJob(ctx, jobID)
	if err != nil {
		if err.Error() == "job not found: "+jobID {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job status: " + err.Error()})
		}
		return
	}

	response := jobStatusResponse{
		JobID:     job.ID,
		Status:    string(job.Status),
		Progress:  job.Progress,
		CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Result:    job.Result,
		Error:     job.Error,
	}

	if job.StartedAt != nil {
		startedStr := job.StartedAt.Format("2006-01-02T15:04:05Z07:00")
		response.StartedAt = &startedStr
	}

	if job.CompletedAt != nil {
		completedStr := job.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		response.CompletedAt = &completedStr

		if job.StartedAt != nil {
			duration := job.CompletedAt.Sub(*job.StartedAt).Milliseconds()
			response.DurationMs = &duration
		}
	}

	ctx.JSON(http.StatusOK, response)
}

// jobListResponse representa a resposta da listagem de jobs
type jobListResponse struct {
	Jobs  []jobStatusResponse `json:"jobs"`
	Total int                 `json:"total"`
	Limit int                 `json:"limit"`
}

// ListJobs godoc
// @Summary Listar jobs de sincronização
// @Description Lista jobs de sincronização com filtros opcionais por status. Útil para monitoramento e debugging de operações assíncronas.
// @Tags B3 Async
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param status query string false "Filtrar por status" Enums(queued, processing, completed, failed, cancelled) example(processing)
// @Param limit query int false "Limite de resultados (máximo 100)" example(10)
// @Success 200 {object} jobListResponse "Lista de jobs"
// @Failure 400 {object} map[string]string "Parâmetros inválidos"
// @Failure 500 {object} map[string]string "Erro interno do servidor"
// @Router /b3/async/jobs [get]
func (c *AsyncSyncController) ListJobs(ctx *gin.Context) {
	// Parâmetros de query
	statusParam := ctx.Query("status")
	limitParam := ctx.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	var status queue.JobStatus
	if statusParam != "" {
		status = queue.JobStatus(statusParam)
		// Validar status válido
		validStatuses := []queue.JobStatus{
			queue.JobStatusQueued,
			queue.JobStatusProcessing,
			queue.JobStatusCompleted,
			queue.JobStatusFailed,
			queue.JobStatusCancelled,
		}
		isValid := false
		for _, validStatus := range validStatuses {
			if status == validStatus {
				isValid = true
				break
			}
		}
		if !isValid {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status parameter"})
			return
		}
	}

	jobs, err := c.jobQueue.ListJobs(ctx, status, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list jobs: " + err.Error()})
		return
	}

	// Converter para response format
	responseJobs := make([]jobStatusResponse, len(jobs))
	for i, job := range jobs {
		responseJobs[i] = jobStatusResponse{
			JobID:     job.ID,
			Status:    string(job.Status),
			Progress:  job.Progress,
			CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Result:    job.Result,
			Error:     job.Error,
		}

		if job.StartedAt != nil {
			startedStr := job.StartedAt.Format("2006-01-02T15:04:05Z07:00")
			responseJobs[i].StartedAt = &startedStr
		}

		if job.CompletedAt != nil {
			completedStr := job.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
			responseJobs[i].CompletedAt = &completedStr

			if job.StartedAt != nil {
				duration := job.CompletedAt.Sub(*job.StartedAt).Milliseconds()
				responseJobs[i].DurationMs = &duration
			}
		}
	}

	response := jobListResponse{
		Jobs:  responseJobs,
		Total: len(responseJobs),
		Limit: limit,
	}

	ctx.JSON(http.StatusOK, response)
}
