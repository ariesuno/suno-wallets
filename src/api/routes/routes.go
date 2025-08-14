package routes

import (
	"context"
	"suno-wallets/src/api/controllers"
	adminctl "suno-wallets/src/api/controllers/admin"
	b3controllers "suno-wallets/src/api/controllers/b3"
	cpctl "suno-wallets/src/api/controllers/clientpolicy"
	obsctl "suno-wallets/src/api/controllers/observability"
	opsctl "suno-wallets/src/api/controllers/ops"
	"suno-wallets/src/api/middlewares"
	adminapp "suno-wallets/src/application/admin"
	diagsvc "suno-wallets/src/application/b3/diagnostics"
	appe2e "suno-wallets/src/application/b3/e2e"
	incrsvc "suno-wallets/src/application/b3/incremental"
	ingest "suno-wallets/src/application/b3/ingest"
	appnorm "suno-wallets/src/application/b3/normalize"
	possvc "suno-wallets/src/application/b3/positions"
	reactivationsvc "suno-wallets/src/application/b3/reactivation"
	apprecon "suno-wallets/src/application/b3/reconciliation"
	repsvc "suno-wallets/src/application/b3/reports"
	appsync "suno-wallets/src/application/b3/sync"
	completesync "suno-wallets/src/application/b3/sync"
	b3svc "suno-wallets/src/application/b3/transactions"
	cpsvc "suno-wallets/src/application/clientpolicy"
	opsapp "suno-wallets/src/application/ops"
	"suno-wallets/src/application/usecases"
	workersvc "suno-wallets/src/application/worker"
	adminrepo "suno-wallets/src/infrastructure/admin"
	b3client "suno-wallets/src/infrastructure/b3/client"
	b3auth "suno-wallets/src/infrastructure/b3/client/auth"
	b3cfg "suno-wallets/src/infrastructure/b3/config"
	e2erepo "suno-wallets/src/infrastructure/b3/e2e"
	normrepo "suno-wallets/src/infrastructure/b3/normalize"
	"suno-wallets/src/infrastructure/b3/persistence"
	reconrepo "suno-wallets/src/infrastructure/b3/reconciliation"
	reprepo "suno-wallets/src/infrastructure/b3/reports"
	syncrepo "suno-wallets/src/infrastructure/b3/sync"
	syncrepo2 "suno-wallets/src/infrastructure/b3/sync"
	cprepo "suno-wallets/src/infrastructure/clientpolicy"
	"suno-wallets/src/infrastructure/observability"
	opsinfra "suno-wallets/src/infrastructure/ops"
	"suno-wallets/src/infrastructure/queue"
	"suno-wallets/src/infrastructure/repositories"
	"suno-wallets/src/shared/build"
	"suno-wallets/src/shared/config"
	"suno-wallets/src/shared/helpers"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRoutes configura todas as rotas da aplicação
func SetupRoutes(cfg *config.Config, db *gorm.DB) *gin.Engine {
	ttlSecondsHelper := func() int { return 60 }
	// Configurar Gin
	gin.SetMode(cfg.GinMode)
	router := gin.New()

	// Middlewares globais
	router.Use(gin.Recovery())
	router.Use(middlewares.CORSMiddleware(cfg))
	router.Use(middlewares.SecurityHeaders()) // Agora com lógica para pular Swagger
	router.Use(middlewares.TraceMiddleware())
	router.Use(observability.PrometheusMiddleware())

	// Configurar Swagger usando o padrão recomendado
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Configurar métricas Prometheus
	observability.RegisterMetricsRoute(router)
	// Registrar build_info
	observability.RegisterBuildInfo(build.Version, build.Commit, build.BuildDate)

	// Inicializar dependências
	walletRepo := repositories.NewWalletRepository(db)
	walletUseCase := usecases.NewWalletUseCase(walletRepo)

	// Inicializar controllers
	healthController := controllers.NewHealthController(db)

	// Inicializar Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Observability health (liveness/readiness/details) sem tenant
	obsHealth := obsctl.NewHealthController(db, redisClient)
	// B3 client + service
	b3Conf := &b3cfg.B3Config{
		URLData:        cfg.B3URLData,
		OAuthTokenURL:  cfg.B3OAuthTokenURL,
		ClientID:       cfg.B3ClientID,
		ClientSecret:   cfg.B3ClientSecret,
		Scope:          cfg.B3Scope,
		CertP12Path:    cfg.B3CertP12Path,
		CertPassphrase: cfg.B3CertPassphrase,
		LegacyCertPath: cfg.B3LegacyCertPath,
		Timeout:        time.Duration(cfg.B3TimeoutSeconds) * time.Second,
		MaxRetries:     cfg.B3MaxRetries,
		InitialBackoff: time.Duration(cfg.B3InitialBackoff) * time.Millisecond,
		MaxBackoff:     time.Duration(cfg.B3MaxBackoff) * time.Millisecond,
	}
	creds := b3auth.NewClientCredentials(b3Conf, nil)
	getBearer := func(ctx2 context.Context) (string, error) {
		tok, err := creds.GetToken(ctx2)
		if err != nil {
			return "", err
		}
		return tok.AccessToken, nil
	}
	b3Cli, err := b3client.NewB3OfficialClient(b3Conf, getBearer)
	if err != nil {
		helpers.LogError("Falha ao inicializar cliente B3", err, map[string]interface{}{
			"cert_path": cfg.B3CertP12Path,
			"url_data":  cfg.B3URLData,
		})
		// Continuar sem cliente B3 - serviços que precisam dele falharão graciosamente
		b3Cli = nil
	}
	transactionsService := b3svc.NewTransactionsService(b3Cli)
	transactionsController := b3controllers.NewTransactionsController(transactionsService)
	positionsService := possvc.NewPositionsService(b3Cli)
	positionsController := b3controllers.NewPositionsController(positionsService)
	// Test service para conectividade B3
	diagnosticsService := diagsvc.NewService(b3Cli)
	testController := b3controllers.NewTestController(diagnosticsService)
	// Ingest (RAW persistence)
	rawRepo := persistence.NewRawRepository(db)
	ingestSvc := ingest.NewService(b3Cli, rawRepo)
	ingestController := b3controllers.NewIngestController(ingestSvc)
	// Normalize
	normRepo := normrepo.NewNormalizedRepository(db)
	normSvc := appnorm.NewService(normRepo)
	// Reports (somente leitura)
	repRepo := reprepo.NewRepository(db)
	repSvc := repsvc.NewService(repRepo)
	repController := b3controllers.NewReportsController(repSvc)
	// Reconciliation (1.17) — scan e leitura
	reconRepo := reconrepo.NewRepository(db)
	reconSvc := apprecon.NewService(reconRepo)
	reconController := b3controllers.NewReconciliationController(reconRepo)
	// Manual Ops (1.19)
	opsRepo := opsinfra.NewOperationsRepository(db)
	manualOpsSvc := opsapp.NewManualOperationsService(opsRepo)
	manualOpsController := opsctl.NewManualOperationsController(manualOpsSvc)
	// Client Policy wiring (1.21)
	polRepo := cprepo.NewRepository(db)
	polCache := cprepo.NewInMemoryCache(time.Duration(ttlSecondsHelper()) * time.Second)
	polSvc := cpsvc.NewService(polRepo, polCache)
	polCtl := cpctl.NewController(polSvc)
	// Dedup wiring
	dedupRepo := opsinfra.NewDedupRepo(db)
	dedupSvc := opsapp.NewDedupService(dedupRepo)
	dedupController := opsctl.NewDedupController(dedupSvc)
	// Timeline & Summary (1.20)
	timelineRepo := opsinfra.NewTimelineRepo(db)
	timelineSvc := opsapp.NewTimelineService(timelineRepo)
	timelineController := opsctl.NewTimelineControllerWithPolicy(timelineSvc, polSvc)
	reconSumRepo := opsinfra.NewReconRepo(db)
	reconSumSvc := opsapp.NewReconSummaryService(reconSumRepo)
	reconSumController := opsctl.NewReconSummaryController(reconSumSvc)
	// Backoffice Admin (1.22)
	admRepo := adminrepo.NewRepository(db)
	admQuery := adminapp.NewQueryService(admRepo)
	admActions := adminapp.NewActionsService(admRepo)
	backofficeController := adminctl.NewController(admQuery, admActions)
	// Auto-fix (1.18)
	sysRepo := reconrepo.NewSysOpsRepository(db)
	// price lookup placeholder: nil (serviço pode derivar de posições em iteração futura)
	autoFixController := b3controllers.NewAutoFixController(sysRepo, nil)
	normController := b3controllers.NewNormalizeController(normSvc)
	// E2E orchestrator (admin)
	resetRepo := e2erepo.NewRepository(db)
	e2eOrch := appe2e.NewOrchestrator(resetRepo, ingestSvc, normSvc)
	// incremental service wiring
	syncRepoIncr := syncrepo2.NewRepository(db)
	incrementalSvc := incrsvc.NewService(syncRepoIncr, ingestSvc, normSvc).WithPolicy(polSvc)
	adminController := b3controllers.NewAdminController(e2eOrch, incrementalSvc)
	utilController := b3controllers.NewUtilController(incrementalSvc)
	// Reactivation service (smart historical sync)
	reactivationSvc := reactivationsvc.NewService(syncRepoIncr, ingestSvc, incrementalSvc)
	reactivationController := b3controllers.NewReactivationController(reactivationSvc)
	// Sync diário
	syncRepo := syncrepo.NewRepository(db)
	syncSvc := appsync.NewService(syncRepo, ingestSvc)
	syncController := b3controllers.NewSyncController(syncSvc)
	// Complete Sync Orchestrator (fluxo completo inteligente)
	completeSyncOrch := completesync.NewCompleteSyncOrchestrator(reactivationSvc, e2eOrch, incrementalSvc, reconSvc, syncRepoIncr)
	completeSyncController := b3controllers.NewCompleteSyncController(completeSyncOrch)

	// Job Queue para processamento assíncrono (Fase 2)
	jobQueue := queue.NewRedisJobQueue(redisClient)
	asyncSyncController := b3controllers.NewAsyncSyncController(jobQueue)

	// Worker Pool para background processing (Fase 2)
	workerPoolSize := 5 // 5 workers paralelos por padrão
	workerPool := workersvc.NewWorkerPool(workerPoolSize, jobQueue, completeSyncOrch)

	// Start worker pool em background
	go func() {
		bgCtx := context.Background()
		workerPool.Start(bgCtx)
		helpers.LogInfo("worker pool started for async processing", map[string]interface{}{
			"poolSize": workerPoolSize,
		})
	}()

	walletController := controllers.NewWalletController(walletUseCase)

	// Rotas de saúde (sem middleware de tenant)
	healthGroup := router.Group("/")
	{
		healthGroup.GET("health", healthController.HealthCheck)
		healthGroup.GET("ready", healthController.ReadinessCheck)
		healthGroup.GET("live", healthController.LivenessCheck)
		// novos endpoints p/ orquestradores
		healthGroup.GET("health/live", obsHealth.Live)
		healthGroup.GET("health/ready", obsHealth.Ready)
		healthGroup.GET("health/details", obsHealth.Details)
	}

	// Grupo de rotas da API com middleware de tenant
	apiV1 := router.Group("/api/v1")
	apiV1.Use(middlewares.TenantMiddleware())
	{
		// Client Policy (admin)
		admin := apiV1.Group("/admin")
		admin.Use(middlewares.RateLimitMiddleware(30, time.Minute))
		admin.GET("/client/policy", polCtl.Get)
		admin.POST("/client/policy", polCtl.Upsert)
		admin.GET("/client/policy/audit", polCtl.Audit)
		admin.POST("/client/policy/dry-run", polCtl.DryRun)
		// Backoffice Admin
		admin.GET("/backoffice/profile", backofficeController.Profile)
		admin.GET("/backoffice/search", backofficeController.Search)
		admin.GET("/backoffice/actions", backofficeController.ListActions)
		admin.GET("/backoffice/actions/:id", backofficeController.GetAction)
		admin.GET("/backoffice/export/ledger", backofficeController.ExportLedger)
		admin.POST("/backoffice/actions/request", backofficeController.RequestAction)
		admin.POST("/backoffice/actions/confirm", backofficeController.ConfirmAction)
		// Rotas de carteiras
		walletsGroup := apiV1.Group("/wallets")
		{
			walletsGroup.POST("", walletController.CreateWallet)
			walletsGroup.GET("", walletController.ListWallets)
			walletsGroup.GET("/:id", walletController.GetWallet)
			walletsGroup.PUT("/:id", walletController.UpdateWallet)
			walletsGroup.DELETE("/:id", walletController.DeleteWallet)
			walletsGroup.GET("/owner/:owner_id", walletController.GetWalletsByOwner)
		}
		// B3
		b3Group := apiV1.Group("/b3")
		{
			// Controller B3 com cfg/creds (para health/auth)
			b3Controller := controllers.NewB3Controller(b3Conf, creds)
			b3Group.GET("/health/auth", b3Controller.HealthAuth)
			// Teste de conectividade B3
			b3Group.GET("/test/connection", testController.TestConnection)
			// Teste rápido de CPF sem dependências externas
			b3Group.GET("/test/cpf-quick", testController.QuickValidateCPF)
			// Preview de transações v2 (equity)
			b3Group.GET("/fetch/transactions/preview", transactionsController.FetchTransactionsPreview)
			// Alias compatível com v_p_1_7 (sem persistência, apenas proxy/preview)
			b3Group.GET("/transactions/:cpf", func(c *gin.Context) {
				// mapear path param para query param
				q := c.Request.URL.Query()
				q.Set("cpf", c.Param("cpf"))
				c.Request.URL.RawQuery = q.Encode()
				transactionsController.FetchTransactionsPreview(c)
			})
			// Preview de posições v3 (equity)
			b3Group.GET("/fetch/positions/preview", positionsController.FetchPositionsPreview)
			// RAW historical ingest
			b3Group.POST("/fetch/historical", ingestController.PostHistorical)
			// Normalization run
			b3Group.POST("/normalize/run", normController.Run)
			// Sync endpoints
			b3Group.POST("/sync/run", syncController.Run)
			b3Group.GET("/client/status", syncController.Status)
			b3Group.GET("/client/last-sync", syncController.LastSync)
			// Complete Sync endpoints (fluxo completo inteligente)
			b3Group.POST("/sync/complete-ingestion", completeSyncController.ExecuteCompleteSync)
			b3Group.GET("/sync/status-analysis", completeSyncController.GetSyncStatus)

			// Async endpoints (Fase 2 - processamento assíncrono)
			asyncGroup := b3Group.Group("/async")
			{
				asyncGroup.POST("/sync/complete-ingestion", asyncSyncController.QueueCompleteSync)
				asyncGroup.GET("/jobs/:jobId/status", asyncSyncController.GetJobStatus)
				asyncGroup.GET("/jobs", asyncSyncController.ListJobs)
			}
			// Admin: reset and refetch E2E
			b3Group.POST("/admin/reset-and-refetch", adminController.ResetAndRefetch)
			// Admin: incremental from last
			b3Group.POST("/admin/incremental-from-last", adminController.IncrementalFromLast)
			// Admin: reactivation service (smart historical sync)
			b3Group.POST("/admin/reactivation/analyze", reactivationController.AnalyzeReactivation)
			b3Group.POST("/admin/reactivation/execute", reactivationController.ExecuteReactivation)
			// Public (autenticado): sync window inspection (somente leitura)
			b3Group.GET("/client/sync-window", utilController.SyncWindow)
			// Public: reactivation status
			b3Group.GET("/client/reactivation/status", reactivationController.GetReactivationStatus)
			// Reports (somente leitura) com rate limit defensivo
			reportsGroup := b3Group.Group("/client")
			reportsGroup.Use(middlewares.RateLimitMiddleware(120, time.Minute))
			reportsGroup.GET("/raw-date-range", repController.RawDateRange)
			reportsGroup.GET("/summary", repController.Summary)
			reportsGroup.GET("/tickers", repController.Tickers)

			// Reconciliation endpoints (admin + read)
			// aplicar rate limit defensivo na rota de scan
			scanGroup := b3Group.Group("/reconciliation")
			scanGroup.Use(middlewares.RateLimitMiddleware(10, time.Minute))
			scanGroup.POST("/scan", reconController.Scan)
			b3Group.GET("/reconciliation/inconsistencies", reconController.List)
			b3Group.GET("/reconciliation/inconsistencies/:id", reconController.Get)
			// Auto-fix endpoints (admin-only)
			af := b3Group.Group("/reconciliation")
			af.Use(middlewares.RateLimitMiddleware(10, time.Minute))
			af.POST("/auto-fix", autoFixController.AutoFix)
			af.POST("/auto-fix/:id", autoFixController.AutoFixByID)

			// Manual Ops endpoints
			opsGroup := apiV1.Group("/ops")
			opsGroup.Use(middlewares.RateLimitMiddleware(120, time.Minute))
			opsGroup.POST("/manual", manualOpsController.Create)
			opsGroup.PUT("/manual/:id", manualOpsController.Update)
			opsGroup.DELETE("/manual/:id", manualOpsController.Delete)
			opsGroup.GET("/manual", manualOpsController.List)
			// Timeline & Summary endpoints
			opsGroup.GET("/timeline", timelineController.Get)
			opsGroup.GET("/timeline/export", timelineController.Export)
			opsGroup.GET("/reconciliation/summary", reconSumController.Get)
			// Dedup endpoints
			dedupGroup := opsGroup.Group("/dedup")
			dedupGroup.Use(middlewares.RateLimitMiddleware(60, time.Minute))
			dedupGroup.POST("/scan", dedupController.Scan)
			dedupGroup.POST("/resolve", dedupController.Resolve)
			dedupGroup.GET("/candidates", dedupController.List)
		}
	}

	return router
}
