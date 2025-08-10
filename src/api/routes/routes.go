package routes

import (
	"context"
	"suno-wallets/src/api/controllers"
	b3controllers "suno-wallets/src/api/controllers/b3"
	obsctl "suno-wallets/src/api/controllers/observability"
	"suno-wallets/src/api/middlewares"
	appe2e "suno-wallets/src/application/b3/e2e"
	incrsvc "suno-wallets/src/application/b3/incremental"
	ingest "suno-wallets/src/application/b3/ingest"
	appnorm "suno-wallets/src/application/b3/normalize"
	possvc "suno-wallets/src/application/b3/positions"
	repsvc "suno-wallets/src/application/b3/reports"
	appsync "suno-wallets/src/application/b3/sync"
	b3svc "suno-wallets/src/application/b3/transactions"
	"suno-wallets/src/application/usecases"
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
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/infrastructure/repositories"
	"suno-wallets/src/shared/build"
	"suno-wallets/src/shared/config"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRoutes configura todas as rotas da aplicação
func SetupRoutes(cfg *config.Config, db *gorm.DB) *gin.Engine {
	// Configurar Gin
	gin.SetMode(cfg.GinMode)
	router := gin.New()

	// Middlewares globais
	router.Use(gin.Recovery())
	router.Use(middlewares.CORSMiddleware(cfg))
	router.Use(middlewares.SecurityHeaders())
	router.Use(middlewares.TraceMiddleware())
	router.Use(observability.PrometheusMiddleware())

	// Configurar Swagger
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

	// Observability health (liveness/readiness/details) sem tenant
	var redisClient *redis.Client
	obsHealth := obsctl.NewHealthController(db, redisClient)
	// B3 client + service
	b3Conf := &b3cfg.B3Config{
		URLData:        cfg.B3URLData,
		OAuthTokenURL:  cfg.B3OAuthTokenURL,
		ClientID:       cfg.B3ClientID,
		ClientSecret:   cfg.B3ClientSecret,
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
	b3Cli, _ := b3client.NewB3OfficialClient(b3Conf, getBearer)
	transactionsService := b3svc.NewTransactionsService(b3Cli)
	transactionsController := b3controllers.NewTransactionsController(transactionsService)
	positionsService := possvc.NewPositionsService(b3Cli)
	positionsController := b3controllers.NewPositionsController(positionsService)
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
    reconController := b3controllers.NewReconciliationController(reconRepo)
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
	incrementalSvc := incrsvc.NewService(syncRepoIncr, ingestSvc, normSvc)
	adminController := b3controllers.NewAdminController(e2eOrch, incrementalSvc)
	utilController := b3controllers.NewUtilController(incrementalSvc)
	// Sync diário
	syncRepo := syncrepo.NewRepository(db)
	syncSvc := appsync.NewService(syncRepo, ingestSvc)
	syncController := b3controllers.NewSyncController(syncSvc)
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
			// Admin: reset and refetch E2E
			b3Group.POST("/admin/reset-and-refetch", adminController.ResetAndRefetch)
			// Admin: incremental from last
			b3Group.POST("/admin/incremental-from-last", adminController.IncrementalFromLast)
			// Public (autenticado): sync window inspection (somente leitura)
			b3Group.GET("/client/sync-window", utilController.SyncWindow)
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
		}
	}

	return router
}
