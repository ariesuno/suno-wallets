package routes

import (
	"context"
	"suno-wallets/src/api/controllers"
	b3controllers "suno-wallets/src/api/controllers/b3"
	obsctl "suno-wallets/src/api/controllers/observability"
	"suno-wallets/src/api/middlewares"
	ingest "suno-wallets/src/application/b3/ingest"
	appnorm "suno-wallets/src/application/b3/normalize"
	possvc "suno-wallets/src/application/b3/positions"
	appsync "suno-wallets/src/application/b3/sync"
	b3svc "suno-wallets/src/application/b3/transactions"
	"suno-wallets/src/application/usecases"
	b3client "suno-wallets/src/infrastructure/b3/client"
	b3auth "suno-wallets/src/infrastructure/b3/client/auth"
	b3cfg "suno-wallets/src/infrastructure/b3/config"
	normrepo "suno-wallets/src/infrastructure/b3/normalize"
	"suno-wallets/src/infrastructure/b3/persistence"
	syncrepo "suno-wallets/src/infrastructure/b3/sync"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/infrastructure/repositories"
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

	// Inicializar dependências
	walletRepo := repositories.NewWalletRepository(db)
	walletUseCase := usecases.NewWalletUseCase(walletRepo)

	// Inicializar controllers
	healthController := controllers.NewHealthController(db)
	b3Controller := controllers.NewB3Controller()

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
	normController := b3controllers.NewNormalizeController(normSvc)
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
		}
	}

	return router
}
