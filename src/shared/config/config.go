package config

import (
	"os"
	"strconv"
	"time"
)

// Config estrutura de configuração da aplicação
type Config struct {
	// Configurações da aplicação
	AppPort string
	AppEnv  string
	GinMode string

	// Configurações do banco de dados
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Configurações do Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// Configurações de JWT
	JWTSecret          string
	JWTExpirationHours int

	// Configurações de Log
	LogLevel  string
	LogFormat string

	// Configurações de Multi-tenant
	DefaultTenantID string

	// Configurações de CORS
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// Load carrega as configurações do ambiente
func Load() *Config {
	return &Config{
		AppPort: getEnv("APP_PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),
		GinMode: getEnv("GIN_MODE", "debug"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "suno_user"),
		DBPassword: getEnv("DB_PASSWORD", "suno_password"),
		DBName:     getEnv("DB_NAME", "suno_wallets"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		JWTSecret:          getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
		JWTExpirationHours: getEnvInt("JWT_EXPIRATION_HOURS", 24),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		DefaultTenantID: getEnv("DEFAULT_TENANT_ID", "00000000-0000-0000-0000-000000000001"),

		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID"},
	}
}

// getEnv busca variável de ambiente ou retorna valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt busca variável de ambiente como inteiro ou retorna valor padrão
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetJWTDuration retorna a duração do token JWT
func (c *Config) GetJWTDuration() time.Duration {
	return time.Duration(c.JWTExpirationHours) * time.Hour
}
