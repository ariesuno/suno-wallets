package main

import (
	"log"
	"os"

	"suno-wallets/src/api/routes"
	"suno-wallets/src/infrastructure/database"
	"suno-wallets/src/infrastructure/migration"
	"suno-wallets/src/shared/config"
	"suno-wallets/src/shared/helpers"

	"github.com/joho/godotenv"
)

// @title Suno Wallets API
// @version 1.0
// @description Sistema de carteiras digitais multi-tenant desenvolvido em Go
// @termsOfService http://swagger.io/terms/

// @contact.name Suporte API
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey TenantID
// @in header
// @name X-Tenant-ID
// @description ID do inquilino para multi-tenancy

func main() {
	// Carregar variáveis de ambiente
	if err := godotenv.Overload(); err != nil {
		helpers.LogInfo("Arquivo .env não encontrado, usando variáveis de sistema", map[string]interface{}{})
	}

	// Inicializar configurações
	cfg := config.Load()

	// Configurar logger
	helpers.InitLogger(cfg.LogLevel, cfg.LogFormat)

	helpers.LogInfo("Iniciando aplicação Suno Wallets", map[string]interface{}{
		"app_env": cfg.AppEnv,
		"port":    cfg.AppPort,
	})

	// Conectar ao banco de dados
	db, err := database.Connect(cfg)
	if err != nil {
		helpers.LogError("Falha ao conectar com o banco de dados", err, map[string]interface{}{})
		log.Fatal(err)
	}

	// Executar migrações automáticas com fallback (EnsureSchema)
	if err := migration.AutoMigrate(db); err != nil {
		helpers.LogError("Falha ao executar migrações (tentando fallback EnsureSchema)", err, map[string]interface{}{})
		_ = migration.CreateExtensions(db)
		if e2 := migration.EnsureSchema(db); e2 != nil {
			helpers.LogError("Fallback EnsureSchema falhou", e2, map[string]interface{}{})
			log.Fatal(e2)
		}
	}

	// Configurar rotas
	router := routes.SetupRoutes(cfg, db)

	// Iniciar servidor
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	helpers.LogInfo("Servidor iniciado com sucesso", map[string]interface{}{
		"port": port,
		"docs": "http://localhost:" + port + "/swagger/index.html",
	})

	if err := router.Run(":" + port); err != nil {
		helpers.LogError("Falha ao iniciar o servidor", err, map[string]interface{}{
			"port": port,
		})
		log.Fatal(err)
	}
}
