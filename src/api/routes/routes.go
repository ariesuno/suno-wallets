package routes

import (
	"suno-wallets/src/api/controllers"
	"suno-wallets/src/api/middlewares"
	"suno-wallets/src/application/usecases"
	"suno-wallets/src/infrastructure/repositories"
	"suno-wallets/src/shared/config"

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

	// Configurar Swagger
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Inicializar dependências
	walletRepo := repositories.NewWalletRepository(db)
	walletUseCase := usecases.NewWalletUseCase(walletRepo)

	// Inicializar controllers
	healthController := controllers.NewHealthController(db)
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
	}

	return router
}
