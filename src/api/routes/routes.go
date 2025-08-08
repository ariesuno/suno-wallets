package routes

import (
	"context"
	"suno-wallets/src/api/controllers"
	b3controllers "suno-wallets/src/api/controllers/b3"
	"suno-wallets/src/api/middlewares"
	b3svc "suno-wallets/src/application/b3/transactions"
	"suno-wallets/src/application/usecases"
	b3client "suno-wallets/src/infrastructure/b3/client"
	b3auth "suno-wallets/src/infrastructure/b3/client/auth"
	b3cfg "suno-wallets/src/infrastructure/b3/config"
	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/infrastructure/repositories"
	"suno-wallets/src/shared/config"

	"time"

	"github.com/gin-gonic/gin"
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
	walletController := controllers.NewWalletController(walletUseCase)

	// Rotas de saúde (sem middleware de tenant)
	healthGroup := router.Group("/")
	{
		healthGroup.GET("health", healthController.HealthCheck)
		healthGroup.GET("ready", healthController.ReadinessCheck)
		healthGroup.GET("live", healthController.LivenessCheck)
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
		}
	}

	return router
}
